# GNU command line argument parser
Command line argument parser following the GNU standard.

    ./test -vo out.png --size 256 input.txt

with the following features:

- build-in help (`-h` and `--help`) message
- scan arguments into struct fields with configuration in tags
- scan into composite field types (arrays, slices, structs)
- allow for nested sub commands

GNU command line argument rules:

- arguments are options when they begin with a hyphen `-`
- multiple options can be combined: `-abc` is the same as `-a -b -c`
- long options start with two hyphens: `--abc` is one option
- option names are alphanumeric characters
- options can have a value: `-a 1` means that `a` has value `1`
- option values can be separated by a space, equal sign, or nothing: `-a1 -a=1 -a 1` are all equal
- options and non-options can be interleaved
- options may appear in any order
- the argument `--` terminates all options so that all following arguments are treated as non-options
- an argument value of `-` is allowed, usually used to mean standard in or out streams
- options may be specified multiple times, only the last one determines its value

Additional features:

- counting options: `-vvv` sets `v = 3`
- appending options: `-v 1 -v 2` sets `v = []int{1, 2}`
- boolean options: `--var` enables and `--no-var` disables a boolean
- options can be composite types, such as structs, slices, or maps:
  - `-v 1,2,3` sets `v = []int{1, 2, 3}`
  - `-v [1 2 3]` sets `v = []int{1, 2, 3}`
  - `-v {1:one 2:two}` sets `map[int]string{1:"one", 2:"two"}`
  - `-v {string 42 [0 1]}` sets `struct{S string, I int, B [2]bool}{"string", 42, false, true}`
- options can retrieve their list/dict values from a source (such as SQL):
  - `-v file:email-addresses.txt` sets `v = []string{ /* lines in email-addresses.txt */ }`
  - `-v file:email-addresses.txt` sets `v = map[string]string{ /* key=value in email-addresses.txt lines */ }`

*See also [github.com/tdewolff/prompt](https://github.com/tdewolff/prompt) for a command line prompter.*

## Installation
Make sure you have [Git](https://git-scm.com/) and [Go](https://golang.org/dl/) (1.22 or higher) installed, run
```
mkdir Project
cd Project
go mod init
go get -u github.com/tdewolff/argp
```

Then add the following import
``` go
import (
    "github.com/tdewolff/argp"
)
```

## Examples
### Default usage
A regular command with short and long options.
See [`cmd/test/main.go`](cmd/test/main.go).

```go
package main

import "github.com/vphpersson/argp"

func main() {
    var verbose int
    var input string
    var output string
    var files []string
    size := 512 // default value

    cmd := argp.New("CLI tool description")
    cmd.AddOpt(argp.Count{&verbose}, "v", "verbose", "Increase verbosity, eg. -vvv")
    cmd.AddOpt(&output, "o", "output", "Output file name")
    cmd.AddOpt(&size, "", "size", "Image size")
    cmd.AddArg(&input, "input", "Input file name")
    cmd.AddRest(&files, "files", "Additional files")
    cmd.Parse()

    // ...
}
```

with help output

```
Usage: test [options] input files...

Options:
  -h, --help          Help
  -o, --output string Output file name
      --size=512 int  Image size
  -v, --verbose int   Increase verbosity, eg. -vvv

Arguments:
  input     Input file name
  files     Additional files
```

### Sub commands
Example with sub commands using a main command for when no sub command is used, and a sub command named "cmd". For the main command we can also use `New` and `AddOpt` instead and process the command after `argp.Parse()`.

```go
package main

import "github.com/tdewolff/argp"

func main() {
    cmd := argp.NewCmd(&Main{}, "CLI tool description")
    cmd.AddCmd(&Command{}, "cmd", "Sub command")
    cmd.Parse()
}

type Main struct {
    Version bool `short:"v"`
}

func (cmd *Main) Run() error {
    // ...
}

type Command struct {
    Verbose bool `short:"v" name:""`
    Output string `short:"o" desc:"Output file name"`
    Size int `default:"512" desc:"Image size"`
}

func (cmd *Command) Run() error {
    // ...
}
```

### Arguments
```go
var input string
cmd.AddArg(&input, "input", "Input file name")

var files []string
cmd.AddRest(&files, "files", "Additional input files")
```

### Options
Basic types
```go
var v string = "default"
cmd.AddOpt(&v, "v", "var", "description")

var v bool = true
cmd.AddOpt(&v, "v", "var", "description")

var v int = 42 // also: int8, int16, int32, int64
cmd.AddOpt(&v, "v", "var", "description")

var v uint = 42 // also: uint8, uint16, uint32, uint64
cmd.AddOpt(&v, "v", "var", "description")

var v float64 = 4.2 // also: float32
cmd.AddOpt(&v, "v", "var", "description")
```

Composite types
```go
v := [2]int{4, 2} // element can be any valid basic or composite type
cmd.AddOpt(&v, "v", "var", "description")
// --var [4 2]  =>  [2]int{4, 2}
// or: --var 4,2  =>  [2]int{4, 2}

v := []int{4, 2, 1} // element can be any valid basic or composite type
cmd.AddOpt(&v, "v", "var", "description")
// --var [4 2 1]  =>  []int{4, 2, 1}
// or: --var 4,2,1  =>  []int{4, 2, 1}

v := map[int]string{1:"one", 2:"two"} // key and value can be any valid basic or composite type
cmd.AddOpt(&v, "v", "var", "description")
// --var {1:one 2:two}  =>  map[int]string{1:"one", 2:"two"}

v := struct { // fields can be any valid basic or composite type
    S string
    I int
    B [2]bool
}{"string", 42, [2]bool{0, 1}}
cmd.AddOpt(&v, "v", "var", "description")
// --var {string 42 [0 1]}  =>  struct{S string, I int, B [2]bool}{"string", 42, false, true}
```

#### Count
Count the number of time a flag has been passed.

```go
var c int
cmd.AddOpt(argp.Count{&c}, "c", "count", "Count")
// Count the number of times flag is present
// -c -c / -cc / --count --count  =>  2
// or: -c 5  =>  5
```

#### Append
Append each flag to a list.

```go
var v []int
cmd.AddOpt(argp.Append{&v}, "v", "value", "Values")
// Append values for each flag
// -v 1 -v 2  =>  [1 2]
```

### Option tags
The following struct will accept the following options and arguments:
- `-v` or `--var` with a default value of 42
- The first argument called `first` with a default value of 4.2
- The other arguments called `rest`

```go
type Command struct {
    Var1 int `short:"v" name:"var" default:"42" desc:"Description"`
    Var2 float64 `name:"first" index:"0" default:"4.2"`
    Var3 []string `name:"rest" index:"*"`
}

func (cmd *Command) Run() error {
    // run command
    return nil
}
```

## License
Released under the [MIT license](LICENSE.md).
