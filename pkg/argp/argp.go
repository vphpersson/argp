package argp

import (
	"fmt"
	motmedelErrors "github.com/Motmedel/utils_go/pkg/errors"
	argpErrors "github.com/vphpersson/argp/pkg/errors"
	argpVariable "github.com/vphpersson/argp/pkg/types/variable"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Cmd is a command.
type Cmd interface {
	Run() error
}

// Argp is a (sub) command parser.
type Argp struct {
	Cmd
	Description string

	parent *Argp
	name   string
	vars   []*argpVariable.Variable
	cmds   map[string]*Argp
	help   bool
}

// New returns a new command parser that can set options and returns the remaining arguments from `Argp.Parse`.
func New(description string) *Argp {
	return NewCmd(nil, description)
}

// NewCmd returns a new command parser that invokes the Run method of the passed command structure. The `Argp.Parse()` function will not return and will call os.Exit() with 0, 1 or 2 as the argument.
func NewCmd(cmd Cmd, description string) *Argp {
	argp := &Argp{
		Cmd:         cmd,
		Description: description,
		name:        filepath.Base(os.Args[0]),
		cmds:        map[string]*Argp{},
	}
	if cmd != nil {
		v := reflect.ValueOf(cmd)
		if v.Type().Kind() != reflect.Ptr {
			panic("cmd: must pass a pointer to struct")
		}
		v = v.Elem()
		if v.Type().Kind() != reflect.Struct {
			panic("cmd: must pass a pointer to struct")
		}

		maxIndex := -1
		for j := range v.NumField() {
			tfield := v.Type().Field(j)
			vfield := v.Field(j)
			if vfield.IsValid() {
				variable := &argpVariable.Variable{}
				variable.Value = vfield
				variable.Name = fromFieldname(tfield.Name)
				variable.Index = -1
				option := reflect.TypeOf(cmd).String() + "." + tfield.Name

				if !isValidType(vfield.Type()) {
					panic(fmt.Sprintf("unsupported type %s", vfield.Type()))
				}

				name, hasName := tfield.Tag.Lookup("name")
				short := tfield.Tag.Get("short")
				index := tfield.Tag.Get("index")
				def, hasDef := tfield.Tag.Lookup("default")
				description := tfield.Tag.Get("desc")

				if hasName {
					variable.Name = strings.ToLower(name)
				}
				if variable.Name == "" {
					variable.Name = short
				}

				if !isValidName(variable.Name) {
					panic(fmt.Sprintf("%v: invalid option name: --%v", option, variable.Name))
				} else if argp.findName(variable.Name) != nil {
					panic(fmt.Sprintf("%v: option name already exists: --%v", option, variable.Name))
				}

				if short != "" {
					if !isValidName(short) {
						panic(fmt.Sprintf("%v: invalid short option name: --%v", option, short))
					}
					r, n := utf8.DecodeRuneInString(short)
					if len(short) != n || n == 0 {
						panic(fmt.Sprintf("%v: short option name must be one character: -%v", option, short))
					} else if argp.findShort(r) != nil {
						panic(fmt.Sprintf("%v: short option name already exists: -%v", option, string(r)))
					}
					variable.Short = r
				}

				if index != "" {
					if short != "" {
						panic(fmt.Sprintf("%v: can not set both an option short name and index", option))
					}
					if index == "*" {
						if argp.findRest() != nil {
							panic(fmt.Sprintf("%v: rest option already exists", option))
						} else if def != "" {
							panic(fmt.Sprintf("%v: rest option can not have a default value", option))
						} else if variable.Value.Kind() != reflect.Slice || variable.Value.Type().Elem().Kind() != reflect.String {
							panic(fmt.Sprintf("%v: rest option must be of type []string", option))
						}
						variable.Rest = true
					} else {
						i, err := strconv.Atoi(index)
						if err != nil || i < 0 {
							panic(fmt.Sprintf("%v: index must be a non-negative integer or *", option))
						} else if argp.findIndex(i) != nil {
							panic(fmt.Sprintf("%v: option index already exists: %v", option, i))
						}
						variable.Index = i
						if maxIndex < i {
							maxIndex = i
						}
					}
				}

				if hasDef {
					defVal := reflect.New(vfield.Type()).Elem()
					if _, err := scanVar(defVal, "", splitArguments(def)); err != nil {
						panic(fmt.Sprintf("%v: bad default value: %v", option, err))
					}
					variable.Default = defVal.Interface()
				} else if variable.Index != -1 {
					variable.Default = vfield.Interface()
				}
				if description != "" {
					variable.Description = description
				}
				argp.vars = append(argp.vars, variable)
			}
		}
		for i := 0; i <= maxIndex; i++ {
			if v := argp.findIndex(i); v == nil {
				panic(fmt.Sprintf("option indices must be continuous: index %v is missing", i))
			}
		}
	}
	if argp.findName("help") == nil {
		if argp.findShort('h') == nil {
			argp.AddOpt(&argp.help, "h", "help", "Help")
		} else {
			argp.AddOpt(&argp.help, "", "help", "Help")
		}
	}
	return argp
}

// AddCmd adds a sub command.
func (argp *Argp) AddCmd(cmd Cmd, name, description string) *Argp {
	if _, ok := argp.cmds[name]; ok {
		panic(fmt.Sprintf("command already exists: %v", name))
	} else if len(name) == 0 || name[0] == '-' {
		panic("invalid command name")
	}

	sub := NewCmd(cmd, description)
	sub.parent = argp
	sub.name = name
	argp.cmds[strings.ToLower(name)] = sub
	return sub
}

// AddOpt adds an option.
func (argp *Argp) AddOpt(dst any, short, name string, description string) {
	v := reflect.ValueOf(dst)
	_, isCustom := dst.(ArgumentScanner)
	if !isCustom && v.Type().Kind() != reflect.Ptr {
		panic("dst: must pass pointer to variable or comply with argp.ArgumentScanner interface")
	} else if !isCustom {
		v = v.Elem()
	}

	variable := &argpVariable.Variable{}
	variable.Value = v
	variable.Index = -1

	if !isValidType(v.Type()) {
		panic(fmt.Sprintf("unsupported type %s", v.Type()))
	} else if name == "" {
		name = short
		if name == "" {
			panic("must set option name")
		}
	}

	if !isValidName(name) {
		panic(fmt.Sprintf("invalid option name: --%v", name))
	} else if argp.findName(name) != nil {
		panic(fmt.Sprintf("option name already exists: --%v", name))
	}
	variable.Name = strings.ToLower(name)
	if short != "" {
		if !isValidName(short) {
			panic(fmt.Sprintf("invalid short option name: -%v", short))
		}
		r, n := utf8.DecodeRuneInString(short)
		if len(short) != n || n == 0 {
			panic(fmt.Sprintf("short option name must be one character: -%v", short))
		} else if argp.findShort(r) != nil {
			panic(fmt.Sprintf("short option name already exists: -%v", string(r)))
		}
		variable.Short = r
	}
	if !isCustom {
		variable.Default = v.Interface()
	}
	variable.Description = description
	argp.vars = append(argp.vars, variable)
}

// AddArg adds an indexed value.
func (argp *Argp) AddArg(dst any, name, description string) {
	v := reflect.ValueOf(dst)
	_, isCustom := dst.(ArgumentScanner)
	if !isCustom && v.Type().Kind() != reflect.Ptr {
		panic("dst: must pass pointer to variable or comply with argp.ArgumentScanner interface")
	} else if !isCustom {
		v = v.Elem()
	}

	variable := &argpVariable.Variable{}
	variable.Value = v
	variable.Name = strings.ToLower(name)
	variable.Index = 0
	if !isValidType(v.Type()) {
		panic(fmt.Sprintf("unsupported type %s", v.Type()))
	}
	for _, v := range argp.vars {
		// find the next free index
		if variable.Index <= v.Index {
			variable.Index = v.Index + 1
		}
	}
	if !isCustom {
		variable.Default = v.Interface()
	}
	variable.Description = description
	argp.vars = append(argp.vars, variable)
}

func (argp *Argp) AddRest(dst any, name, description string) {
	v := reflect.ValueOf(dst)
	_, isCustom := dst.(ArgumentScanner)
	if !isCustom && v.Type().Kind() != reflect.Ptr {
		panic("dst: must pass pointer to variable or comply with argp.ArgumentScanner interface")
	} else if !isCustom {
		v = v.Elem()
	}

	variable := &argpVariable.Variable{}
	variable.Value = v
	variable.Name = strings.ToLower(name)
	variable.Index = -1
	if argp.findRest() != nil {
		panic("rest option already exists")
	} else if v.Kind() != reflect.Slice || v.Type().Elem().Kind() != reflect.String {
		panic("rest option must be of type []string")
	}
	variable.Rest = true
	if !isCustom {
		variable.Default = v.Interface()
	}
	variable.Description = description
	argp.vars = append(argp.vars, variable)
}

type optionHelp struct {
	short, name, typ, desc string
}

func getOptionHelps(vs []*argpVariable.Variable) []optionHelp {
	var helps []optionHelp

	for _, v := range vs {
		var val, typ string
		if custom, ok := v.Value.Interface().(ArgumentScanner); ok {
			val, typ = custom.Help()
		} else {
			if v.Default != nil && !reflect.ValueOf(v.Default).IsZero() {
				val = fmt.Sprint(v.Default)
			}
			typ = TypeName(v.Value.Type())
		}

		var short, name string
		if v.Short != 0 {
			short = string(v.Short)
		}
		name = v.Name
		if val != "" {
			if space := strings.IndexByte(val, ' '); space != -1 {
				val = "'" + val + "'"
			}
			if name != "" {
				name += "=" + val
			} else {
				short += "=" + val
			}
		}
		helps = append(helps, optionHelp{
			short: short,
			name:  name,
			typ:   typ,
			desc:  v.Description,
		})

	}
	return helps
}

// PrintHelp prints the help overview. This is automatically called when unknown or bad options are passed, but you can call this explicitly in other cases.
func (argp *Argp) PrintHelp() {
	base := argp.name
	parent := argp.parent
	for parent != nil {
		base = parent.name + " " + base
		parent = parent.parent
	}

	var options []*argpVariable.Variable
	var arguments []*argpVariable.Variable
	for _, v := range argp.vars {
		if v.IsArgument() {
			arguments = append(arguments, v)
		} else {
			options = append(options, v)
		}
	}

	sort.Slice(options, sortOption(options))
	sort.Slice(arguments, sortArgument(arguments))

	args := ""
	if 0 < len(options) {
		args += " [options]"
	}
	if 0 < len(argp.cmds) {
		fmt.Printf("Usage: %s%s [command] ...\n", base, args)
	}
	if 0 < len(arguments) {
		for _, v := range arguments {
			if !v.Rest {
				args += " " + v.Name
			}
		}
		if rest := argp.findRest(); rest != nil {
			args += " " + rest.Name + "..."
		}
	}
	if 0 < len(arguments) || len(argp.cmds) == 0 {
		fmt.Printf("Usage: %s%s\n", base, args)
	}

	if 0 < len(options) {
		optionHelps := getOptionHelps(options)

		fmt.Printf("\nOptions:\n")
		nMax := 0
		for _, o := range optionHelps {
			n := 0
			if o.short != "" {
				n += 4
				if o.name != "" {
					n += 4 + len(o.name)
				}
			} else if o.name != "" {
				n += 8 + len(o.name)
			}
			if o.typ != "" {
				n += 1 + len(o.typ)
			}
			n++ // whitespace before description
			if nMax < n {
				nMax = n
			}
		}
		if 30 < nMax {
			nMax = 30
		} else if nMax < 10 {
			nMax = 10
		}
		for _, o := range optionHelps {
			n := 0
			if o.short != "" {
				fmt.Printf("  -%s, --%s", o.short, o.name)
				n += 8 + len(o.name)
			} else if o.name != "" {
				fmt.Printf("      --%s", o.name)
				n += 8 + len(o.name)
			}
			if o.typ != "" {
				fmt.Printf(" %s", o.typ)
				n += 1 + len(o.typ)
			}
			if nMax <= n {
				fmt.Printf("\n")
				n = 0
			}
			fmt.Printf("%s", strings.Repeat(" ", nMax-n))
			fmt.Printf("%s\n", o.desc)
		}
	}

	if 0 < len(argp.cmds) {
		fmt.Printf("\nCommands:\n")
		nMax := 0
		var cmds []string
		for cmd := range argp.cmds {
			if nMax < 2+len(cmd) {
				nMax = 2 + len(cmd)
			}
			cmds = append(cmds, cmd)
		}
		sort.Strings(cmds)

		if 28 < nMax {
			nMax = 28
		} else if nMax < 10 {
			nMax = 10
		}
		for _, cmd := range cmds {
			sub := argp.cmds[cmd]
			n := 2 + len(cmd)
			fmt.Printf("  %s", cmd)
			if nMax < n {
				fmt.Printf("\n")
				n = 0
			}
			fmt.Printf("%s  %s\n", strings.Repeat(" ", nMax-n), sub.Description)
		}
	}

	if 0 < len(arguments) {
		fmt.Printf("\nArguments:\n")
		nMax := 0
		for _, v := range arguments {
			n := 2 + len(v.Name)
			if nMax < n {
				nMax = n
			}
		}
		if 28 < nMax {
			nMax = 28
		} else if nMax < 10 {
			nMax = 10
		}
		for _, v := range arguments {
			n := 2 + len(v.Name)
			fmt.Printf("  %s", v.Name)
			if nMax < n {
				fmt.Printf("\n")
				n = 0
			}
			fmt.Printf("%s  %s\n", strings.Repeat(" ", nMax-n), v.Description)
		}
	}
}

// Parse parses the command line arguments. When the main command was instantiated with `NewCmd`, this command will exit.
func (argp *Argp) Parse() error {
	arguments := os.Args[1:]
	cmd, rest, err := argp.parse(arguments)
	if err != nil {
		return motmedelErrors.New(fmt.Errorf("parse: %w", err), arguments)
	}

	// TODO: What do these conditions mean?
	if cmd.help || cmd != argp && cmd.Cmd == nil {
		return argpErrors.ErrShowHelp
	}

	// TODO: What does this condition mean?
	if cmd.Cmd == nil {
		return nil
	}

	if len(rest) != 0 {
		restString := strings.Join(rest, " ")
		return motmedelErrors.NewWithTrace(fmt.Errorf("%w: %s", argpErrors.ErrUnexpectedInput, restString))
	}

	if err := cmd.Cmd.Run(); err != nil {
		return motmedelErrors.New(fmt.Errorf("cmd run: %w", err), cmd.Cmd)
	}

	return nil
}

func (argp *Argp) findShort(short rune) *argpVariable.Variable {
	for _, v := range argp.vars {
		if v.Short != 0 && v.Short == short {
			return v
		}
	}
	return nil
}

func (argp *Argp) findName(name string) *argpVariable.Variable {
	if name == "" {
		return nil
	}

	name = strings.ToLower(name)
	if i := strings.IndexAny(name, ".["); i != -1 {
		name = name[:i]
	}

	for _, v := range argp.vars {
		if v.Name == name || v.Name == "" && string(v.Short) == name {
			return v
		}
	}
	return nil
}

func (argp *Argp) findIndex(index int) *argpVariable.Variable {
	for _, v := range argp.vars {
		if v.Index == index {
			return v
		}
	}
	return nil
}

func (argp *Argp) findRest() *argpVariable.Variable {
	for _, v := range argp.vars {
		if v.Rest {
			return v
		}
	}
	return nil
}

func (argp *Argp) parse(args []string) (*Argp, []string, error) {
	// sub commands
	if 0 < len(args) {
		for cmd, sub := range argp.cmds {
			if cmd == strings.ToLower(args[0]) {
				return sub.parse(args[1:])
			}
		}
	}

	// set defaults
	for _, v := range argp.vars {
		if v.Default != nil {
			if ok := v.Set(v.Default); !ok {
				return argp, nil, fmt.Errorf("default: expected type %v", v.Value.Type())
			}
		}
	}

	var rest []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		if 1 < len(arg) && arg[0] == '-' {
			if 1 < len(arg) && arg[1] == '-' {
				split := false
				s := args[i+1:]
				name := arg[2:]
				if idx := strings.IndexByte(arg, '='); idx != -1 {
					name = arg[2:idx]
					if idx+1 < len(arg) {
						s = append([]string{arg[idx+1:]}, args[i+1:]...)
						split = true
					}
				}

				v := argp.findName(name)
				if v == nil {
					return argp, nil, motmedelErrors.NewWithTrace(
						fmt.Errorf("%w: %s", argpErrors.ErrUnknownOption, name),
					)
				}

				value := v.Value
				n, err := scanVar(value, name, s)
				if err != nil {
					return argp, nil, motmedelErrors.New(fmt.Errorf("scan var: %w", err), value, name, s)
				} else {
					i += n
					if split {
						i--
					}
				}
				v.IsSet = true
			} else {
				for j := 1; j < len(arg); {
					name, n := utf8.DecodeRuneInString(arg[j:])
					j += n

					v := argp.findShort(name)
					if v == nil {
						return argp, nil, motmedelErrors.NewWithTrace(
							fmt.Errorf("%w: %s", argpErrors.ErrUnknownOption, string(name)),
							name,
						)
					} else {
						s := append([]string{arg[j:]}, args[i+1:]...)
						hasEquals := j < len(arg) && arg[j] == '='
						if hasEquals {
							s[0] = s[0][1:]
						}
						valueGlued := 0 < len(s[0])
						if !valueGlued {
							s = s[1:]
						}

						nameString := string(name)
						value := v.Value

						n, err := scanVar(value, nameString, s)
						if err != nil {
							return argp, nil, motmedelErrors.New(fmt.Errorf("scan var: %w", err), value, nameString, s)
						} else if n == 0 {
							continue // can be of the form -abc
						}
						if valueGlued {
							n--
						}
						i += n
						break
					}
					v.IsSet = true
				}
			}
		} else if 0 < len(arg) {
			rest = append(rest, arg)
		}
	}

	// indexed arguments
	index := 0
	for _, arg := range rest {
		v := argp.findIndex(index)
		if v == nil {
			break
		}
		if _, err := scanVar(v.Value, "", []string{arg}); err != nil {
			return argp, nil, fmt.Errorf("argument %d: %v", index, err)
		}
		v.IsSet = true
		index++
	}

	// rest arguments
	v := argp.findRest()
	rest = rest[index:]
	if v != nil {
		v.Set(rest)
		rest = rest[:0]
		v.IsSet = true
	}
	return argp, rest, nil
}

// scanVar parses a slice of strings into the given value.
func scanVar(v reflect.Value, name string, arguments []string) (int, error) {
	if scanner, ok := v.Interface().(ArgumentScanner); ok {
		n, err := scanner.Scan(name, arguments)
		if err != nil {
			return 0, motmedelErrors.New(fmt.Errorf("scanner scan: %w", err), scanner)
		}

		return n, nil
	}

	n, err := scanValue(v, arguments)
	if err != nil && v.Kind() == reflect.Bool {
		v.SetBool(true)
		return 0, nil
	}
	return n, err
}

func scanValue(v reflect.Value, arguments []string) (int, error) {
	if len(arguments) == 0 {
		if v.Kind() == reflect.String {
			v.SetString("")
			return 0, nil
		}

		return 0, motmedelErrors.NewWithTrace(argpErrors.ErrMissingValue)
	}

	n := 0
	switch kind := v.Kind(); kind {
	case reflect.String:
		v.SetString(arguments[0])
		n++
	case reflect.Bool:
		boolCandidate := arguments[0]
		i, err := strconv.ParseBool(boolCandidate)
		if err != nil {
			return 0, motmedelErrors.NewWithTrace(fmt.Errorf("strconv parse bool: %w", err), boolCandidate)
		}

		v.SetBool(i)
		n++
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intCandidate := arguments[0]
		i, err := strconv.ParseInt(intCandidate, 10, 64)
		if err != nil {
			return 0, motmedelErrors.NewWithTrace(fmt.Errorf("strconv parse int: %w", err), intCandidate)
		}

		v.SetInt(i)
		n++
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintCandidate := arguments[0]
		i, err := strconv.ParseUint(uintCandidate, 10, 64)
		if err != nil {
			return 0, motmedelErrors.NewWithTrace(fmt.Errorf("strconv parse uint: %w", err), uintCandidate)
		}

		v.SetUint(i)
		n++
	case reflect.Float32, reflect.Float64:
		floatCandidate := arguments[0]
		i, err := strconv.ParseFloat(floatCandidate, 64)
		if err != nil {
			return 0, motmedelErrors.NewWithTrace(fmt.Errorf("strconv parse float: %w", err), floatCandidate)
		}

		v.SetFloat(i)
		n++
	case reflect.Array, reflect.Slice:
		if len(arguments[0]) == 0 {
			return 1, nil
		}

		typ := "array"
		if v.Kind() == reflect.Slice {
			typ = "slice"
		}

		j := 0
		slice := reflect.Zero(reflect.SliceOf(v.Type().Elem()))
		if v.Kind() == reflect.Slice {
			slice = v
		}
		for {
			if j != 0 {
				// consume comma
				for 0 < len(arguments) && len(arguments[0]) == 0 {
					arguments = arguments[1:]
					n++
				}
				if len(arguments) == 0 || arguments[0][0] != ',' {
					break
				} else if len(arguments[0]) == 1 {
					arguments = arguments[1:]
					n++
				} else {
					arguments[0] = arguments[0][1:]
				}
			}

			// consume value
			var sVal []string
			if len(arguments) == 0 {
				if j == 0 {
					break
				}
				// empty value after final comma
				sVal = []string{""}
			} else if idx := strings.IndexByte(arguments[0], ','); idx != -1 {
				sVal = []string{arguments[0][:idx]}
				arguments[0] = arguments[0][idx:]
			} else {
				sVal = []string{arguments[0]}
				arguments = arguments[1:]
				n++
			}
			val := reflect.New(v.Type().Elem()).Elem()
			if _, err := scanValue(val, sVal); err != nil {
				return 0, fmt.Errorf("%v index %v: %v", typ, j, err)
			}
			slice = reflect.Append(slice, val)
			j++
		}
		if v.Kind() == reflect.Array {
			if j != v.Len() {
				return 0, fmt.Errorf("expected %v values for %v", v.Len(), typ)
			}
			v.Set(slice.Convert(v.Type()))
		} else {
			v.Set(slice)
		}
	default:
		return n, motmedelErrors.NewWithTrace(
			fmt.Errorf("%w: %v", argpErrors.ErrUnexpectedKind, kind),
		)
	}

	return n, nil
}

// isValidName returns true if the short or long option name is valid.
func isValidName(s string) bool {
	for i, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '_' && (r != '-' || i == 0) {
			return false
		}
	}
	return true
}

// isValidType returns true if the destination variable type is supported. Either it implements the ArgumentScanner interface, or is a valid base type.
func isValidType(t reflect.Type) bool {
	if t.Implements(reflect.TypeOf((*ArgumentScanner)(nil)).Elem()) {
		// implements ArgumentScanner
		return true
	}
	return isValidBaseType(t)
}

func isValidBaseType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return true
	case reflect.Array, reflect.Slice:
		return isValidBaseType(t.Elem())
	case reflect.Map:
		return isValidBaseType(t.Key()) && isValidBaseType(t.Elem())
	case reflect.Struct:
		for i := range t.NumField() {
			if !isValidBaseType(t.Field(i).Type) {
				return false
			}
		}
		return true
	}
	return false
}

// TypeName returns the type's name.
func TypeName(t reflect.Type) string {
	k := t.Kind()
	if k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64 {
		return "int"
	} else if k == reflect.Uint || k == reflect.Uint8 || k == reflect.Uint16 || k == reflect.Uint32 || k == reflect.Uint64 {
		return "uint"
	} else if k == reflect.Float32 || k == reflect.Float64 {
		return "float"
	} else if k == reflect.Array || k == reflect.Slice {
		return "[]" + TypeName(t.Elem())
	} else if k == reflect.Map {
		return "map[" + TypeName(t.Key()) + "]" + TypeName(t.Elem())
	} else if k == reflect.String {
		return "string"
	} else if k == reflect.Struct {
		return "struct"
	}
	return ""
}

// sortOption sorts options by short and then name.
func sortOption(vars []*argpVariable.Variable) func(int, int) bool {
	return func(i, j int) bool {
		if vars[i].Short != 0 {
			if vars[j].Short != 0 {
				return vars[i].Short < vars[j].Short
			} else {
				return string(vars[i].Short) < vars[j].Name
			}
		} else if vars[j].Short != 0 {
			return vars[i].Name < string(vars[j].Short)
		}
		return vars[i].Name < vars[j].Name
	}
}

// sortArgument sorts arguments by index and then rest.
func sortArgument(vars []*argpVariable.Variable) func(int, int) bool {
	return func(i, j int) bool {
		if vars[i].Rest {
			return false
		} else if vars[j].Rest {
			return true
		}
		return vars[i].Index < vars[j].Index
	}
}

func fromFieldname(field string) string {
	name := make([]byte, 0, len(field))
	prevUpper := false
	for i, r := range field {
		if unicode.IsTitle(r) || unicode.IsUpper(r) {
			rNext, n := utf8.DecodeRuneInString(field[i+utf8.RuneLen(r):])
			if i != 0 && n != 0 && (!prevUpper || !unicode.IsTitle(rNext) && !unicode.IsUpper(rNext)) {
				name = append(name, '-')
			}
			name = utf8.AppendRune(name, unicode.ToLower(r))
			prevUpper = true
		} else {
			name = utf8.AppendRune(name, r)
			prevUpper = false
		}
	}
	return string(name)
}

func splitArguments(s string) []string {
	i := 0
	var esc bool
	var quote rune
	arg := ""
	var args []string
	for j, r := range s {
		if r == '\\' {
			if i < j {
				arg += s[i:j]
			}
			i = j + 1
			esc = true
		} else if esc {
			esc = false
		} else if (quote == 0 || quote == r) && r == '\'' || r == '"' {
			if quote == 0 {
				quote = r
			} else {
				quote = 0
			}
			if i < j {
				arg += s[i:j]
			}
			i = j + 1
		} else if quote == 0 && unicode.IsSpace(r) {
			if i < j {
				args = append(args, arg+s[i:j])
				arg = ""
			}
			i = j + utf8.RuneLen(r)
		}
	}
	if i < len(s) {
		args = append(args, arg+s[i:])
	} else {
		args = append(args, arg)
	}
	return args
}
