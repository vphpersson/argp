package variable

import "reflect"

// Variable is a command option or argument.
type Variable struct {
	Value       reflect.Value
	Name        string
	Short       rune // 0 if not used
	Index       int  // -1 if not used
	Rest        bool
	Default     any // nil is not used
	Description string
	IsSet       bool
}

// IsOption returns true for an option.
func (v *Variable) IsOption() bool {
	return !v.IsArgument()
}

// IsArgument returns true for an argument.
func (v *Variable) IsArgument() bool {
	return v.Index != -1 || v.Rest
}

// Set sets the variable's value.
func (v *Variable) Set(i any) bool {
	val := reflect.ValueOf(i)
	if !val.CanConvert(v.Value.Type()) {
		return false
	}
	v.Value.Set(val.Convert(v.Value.Type()))
	return true
}
