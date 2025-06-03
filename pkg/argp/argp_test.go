package argp

import (
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	argpErrors "github.com/vphpersson/argp/pkg/errors"
	"strconv"
	"strings"
	"testing"
)

var diffOpts = []cmp.Option{cmpopts.EquateEmpty()}

type STypesStruct struct {
	Bool   bool
	Struct struct {
		Float64 float64
	}
}

type STypes struct {
	String  string
	Bool    bool
	Int     int
	Int8    int8
	Int16   int16
	Int32   int32
	Int64   int64
	Uint    uint
	Uint8   uint8
	Uint16  uint16
	Uint32  uint32
	Uint64  uint64
	Float32 float32
	Float64 float64
	Struct  STypesStruct `short:"s"`
}

func (_ *STypes) Run() error {
	return nil
}

func TestArgpTypes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		arguments []string
		sTypes    STypes
	}{
		{[]string{"--string", "val"}, STypes{String: "val"}},
		{[]string{"--string", ""}, STypes{String: ""}},
		{[]string{"--bool"}, STypes{Bool: true}},
		{[]string{"--int", "36"}, STypes{Int: 36}},
		{[]string{"--int8", "36"}, STypes{Int8: 36}},
		{[]string{"--int16", "36"}, STypes{Int16: 36}},
		{[]string{"--int32", "36"}, STypes{Int32: 36}},
		{[]string{"--int64", "36"}, STypes{Int64: 36}},
		{[]string{"--uint", "36"}, STypes{Uint: 36}},
		{[]string{"--uint8", "36"}, STypes{Uint8: 36}},
		{[]string{"--uint16", "36"}, STypes{Uint16: 36}},
		{[]string{"--uint32", "36"}, STypes{Uint32: 36}},
		{[]string{"--uint64", "36"}, STypes{Uint64: 36}},
		{[]string{"--float32", "36"}, STypes{Float32: 36}},
		{[]string{"--float64", "36"}, STypes{Float64: 36}},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", testCase.arguments), func(t *testing.T) {
			t.Parallel()

			sTypes := STypes{}
			_, rest, err := NewCmd(&sTypes, "description").parse(testCase.arguments)
			if err != nil {
				t.Fatalf("argp parse: %v", err)
			}

			if diff := cmp.Diff(testCase.sTypes, sTypes, diffOpts...); diff != "" {
				t.Errorf("mismatch (-expected +got):\n%s", diff)
			}

			if len(rest) != 0 {
				t.Errorf("non-empty rest: %v", rest)
			}
		})
	}
}

func TestArgpErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		arguments []string
		error     error
	}{
		{[]string{"--int"}, argpErrors.ErrMissingValue},
		{[]string{"--int", "string"}, strconv.ErrSyntax},
		{[]string{"--uint", "-1"}, strconv.ErrSyntax},
		{[]string{"--float64", "."}, strconv.ErrSyntax},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", testCase.arguments), func(t *testing.T) {
			t.Parallel()

			s := STypes{}
			argp := NewCmd(&s, "description")

			_, _, err := argp.parse(testCase.arguments)

			if !errors.Is(err, testCase.error) {
				t.Errorf("error mismatch: expected %q, got %q", testCase.error, err)
			}
		})
	}
}

type SOptions struct {
	Foo  string `short:"f"`
	Bar  string `name:"barbar"`
	Baz  string `default:"default"`
	A    bool   `short:"a"`
	B    bool   `short:"b"`
	C    int    `short:"c"`
	Name string `name:"N-a_më"`
}

func (_ *SOptions) Run() error {
	return nil
}

func TestArgp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		argument []string
		sTypes   SOptions
		rest     string
	}{
		{[]string{"--foo", "val"}, SOptions{Foo: "val", Baz: "default"}, ""},
		{[]string{"-f", "val"}, SOptions{Foo: "val", Baz: "default"}, ""},
		{[]string{"--barbar", "val"}, SOptions{Bar: "val", Baz: "default"}, ""},
		{[]string{"--baz", "val"}, SOptions{Baz: "val"}, ""},
		{[]string{"input1", "input2"}, SOptions{Baz: "default"}, "input1 input2"},
		{[]string{"input1", "--baz", "val", "input2"}, SOptions{Baz: "val"}, "input1 input2"},
		{[]string{"-a"}, SOptions{Baz: "default", A: true}, ""},
		{[]string{"-a", "-b", "-c", "5"}, SOptions{Baz: "default", A: true, B: true, C: 5}, ""},
		{[]string{"-a", "-b", "-c=5"}, SOptions{Baz: "default", A: true, B: true, C: 5}, ""},
		{[]string{"-a", "-b", "-c5"}, SOptions{Baz: "default", A: true, B: true, C: 5}, ""},
		{[]string{"-abc5"}, SOptions{Baz: "default", A: true, B: true, C: 5}, ""},
		{[]string{"--", "-abc5"}, SOptions{Baz: "default"}, "-abc5"},
		{[]string{"--n-A_më", "val"}, SOptions{Baz: "default", Name: "val"}, ""},
		{[]string{"--Baz=-"}, SOptions{Baz: "-"}, ""},
		{[]string{"--Baz", "-"}, SOptions{Baz: "-"}, ""},
		{[]string{"-"}, SOptions{Baz: "default"}, "-"},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%v", testCase.argument), func(t *testing.T) {
			t.Parallel()

			sTypes := SOptions{}
			argp := NewCmd(&sTypes, "description")

			_, rest, err := argp.parse(testCase.argument)
			if err != nil {
				t.Fatalf("argp parse: %v", err)
			}

			if diff := cmp.Diff(testCase.sTypes, sTypes, diffOpts...); diff != "" {
				t.Errorf("mismatch (-expected +got):\n%s", diff)
			}

			expectedRest := testCase.rest
			gotRest := strings.Join(rest, " ")

			if expectedRest != gotRest {
				t.Errorf("mismatch: expected %q, got %q", expectedRest, gotRest)
			}
		})
	}
}

func TestArgpAdd(t *testing.T) {
	t.Parallel()

	o := int64(4)
	var v bool
	argp := New("description")
	argp.AddOpt(&o, "", "name", "description")
	argp.AddArg(&v, "", "description")

	_, _, err := argp.parse([]string{"--name", "8", "true"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}

	if expected := int64(8); o != expected {
		t.Errorf("expected %q, got %q", expected, o)
	}

	if expected := true; v != expected {
		t.Errorf("expected %v, got %v", expected, v)
	}

	_, _, err = argp.parse([]string{})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}
}

func TestArgpAddRest(t *testing.T) {
	t.Parallel()

	var rest []string
	argp := New("description")
	argp.AddRest(&rest, "rest", "description")

	_, _, err := argp.parse([]string{"file1.txt", "file2.txt", "file3.txt"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}

	expected := []string{"file1.txt", "file2.txt", "file3.txt"}
	if diff := cmp.Diff(expected, rest, diffOpts...); diff != "" {
		t.Errorf("mismatch (-expected +got):\n%s", diff)
	}

	_, _, err = argp.parse([]string{})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}

	expected = []string{}
	if diff := cmp.Diff(expected, rest, diffOpts...); diff != "" {
		t.Errorf("mismatch (-expected +got):\n%s", diff)
	}
}

func TestArgpUTF8(t *testing.T) {
	t.Parallel()

	var v bool
	argp := New("description")
	argp.AddOpt(&v, "á", "", "description")

	_, _, err := argp.parse([]string{"-á"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}

	if expected := true; v != expected {
		t.Errorf("expected %v, got %v", expected, v)
	}
}

func TestArgpCount(t *testing.T) {
	t.Parallel()

	var i int
	argp := New("description")
	argp.AddOpt(Count{&i}, "i", "int", "description")

	_, _, err := argp.parse([]string{"-i", "-ii", "--int", "--int"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}
	if expected := 5; i != expected {
		t.Errorf("expected %v, got %v", expected, i)
	}

	_, _, err = argp.parse([]string{"-i", "3"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}
	if expected := 3; i != expected {
		t.Errorf("expected %v, got %v", expected, i)
	}

	_, _, err = argp.parse([]string{"--int", "3"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}
	if expected := 3; i != expected {
		t.Errorf("expected %v, got %v", expected, i)
	}
}

func TestArgpAppend(t *testing.T) {
	t.Parallel()

	var i []int
	var s []string
	argp := New("description")
	argp.AddOpt(Append{&i}, "i", "int", "description")
	argp.AddOpt(Append{&s}, "s", "string", "description")

	_, _, err := argp.parse([]string{"-i", "1", "--int", "2"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}

	expectedInts := []int{1, 2}
	if diff := cmp.Diff(expectedInts, i, diffOpts...); diff != "" {
		t.Errorf("mismatch (-expected +got):\n%s", diff)
	}

	_, _, err = argp.parse([]string{"-s", "foo", "--string", "bar"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}

	expectedStrings := []string{"foo", "bar"}
	if diff := cmp.Diff(expectedStrings, s, diffOpts...); diff != "" {
		t.Errorf("mismatch (-expected +got):\n%s", diff)
	}
}

type SSub1 struct {
	B int `short:"b"`
}

func (_ *SSub1) Run() error {
	return nil
}

type SSub2 struct {
	C int `short:"c"`
}

func (_ *SSub2) Run() error {
	return nil
}

func TestArgpSubCommand(t *testing.T) {
	t.Parallel()

	var v string
	var a int
	sub1 := SSub1{}
	sub2 := SSub2{}
	argp := New("description")
	argp.AddArg(&v, "", "description")
	argp.AddOpt(&a, "a", "", "description")
	argp.AddCmd(&sub1, "one", "description")
	argp.AddCmd(&sub2, "two", "description")

	_, _, err := argp.parse([]string{"val", "-a", "1"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}
	if expected := "val"; v != expected {
		t.Errorf("expected %v, got %v", expected, v)
	}
	if expected := 1; a != expected {
		t.Errorf("expected %v, got %v", expected, a)
	}

	_, _, err = argp.parse([]string{"one", "-b", "2"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}
	if expected := 2; sub1.B != expected {
		t.Errorf("expected %v, got %v", expected, sub1.B)
	}

	_, _, err = argp.parse([]string{"two", "-c", "3"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}
	if expected := 3; sub2.C != expected {
		t.Errorf("expected %v, got %v", expected, sub2.C)
	}
}

func TestSplitArguments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		str       string
		arguments []string
	}{
		{"foobar", []string{"foobar"}},
		{"foo bar", []string{"foo", "bar"}},
		{"'foo bar'", []string{"foo bar"}},
		{"'foo'\"bar\"", []string{"foobar"}},
		{"'foo\\'bar'", []string{"foo'bar"}},
		{"foo ' bar '", []string{"foo", " bar "}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.str, func(t *testing.T) {
			t.Parallel()

			args := splitArguments(testCase.str)
			if diff := cmp.Diff(testCase.arguments, args, diffOpts...); diff != "" {
				t.Errorf("mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

func TestCount(t *testing.T) {
	t.Parallel()

	var count int
	argp := New("count variable")
	argp.AddOpt(Count{&count}, "c", "count", "")

	_, _, err := argp.parse([]string{"-ccc"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}

	if expected := 3; count != expected {
		t.Errorf("expected %v, got %v", expected, count)
	}
}

func ExampleCount() {
	var count int
	argp := New("count variable")
	argp.AddOpt(Count{&count}, "c", "count", "")

	_, _, err := argp.parse([]string{"-ccc"})
	if err != nil {
		panic(err)
	}
	fmt.Println(count)
	// Output: 3
}

type CustomVar struct {
	Num, Div float64
}

func (e *CustomVar) Help() (string, string) {
	return fmt.Sprintf("%v/%v", e.Num, e.Div), ""
}

func (e *CustomVar) Scan(name string, s []string) (int, error) {
	n := 0
	num := s[0]
	if idx := strings.IndexByte(s[0], '/'); idx != -1 {
		num = s[0][:idx]
		if idx+1 == len(s[0]) {
			s = s[1:]
			n++
		} else {
			s[0] = s[0][idx+1:]
		}
	} else if 1 < len(s) && 0 < len(s[1]) && s[1][0] == '/' {
		s = s[1:]
		n++
		if len(s[0]) == 1 {
			s = s[1:]
			n++
		} else {
			s[0] = s[0][1:]
		}
	} else {
		return 0, fmt.Errorf("missing fraction")
	}
	div := s[0]
	fnum, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number '%v'", num)
	}
	fdiv, err := strconv.ParseFloat(div, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number '%v'", div)
	}
	e.Num = fnum
	e.Div = fdiv
	return n + 1, nil
}

func TestArgumentScanner(t *testing.T) {
	t.Parallel()

	custom := CustomVar{}
	argp := New("custom variable")
	argp.AddOpt(&custom, "", "custom", "")

	_, _, err := argp.parse([]string{"--custom", "1", "/", "2"})
	if err != nil {
		t.Fatalf("argp parse: %v", err)
	}

	expected := CustomVar{1.0, 2.0}

	if diff := cmp.Diff(expected, custom, diffOpts...); diff != "" {
		t.Errorf("mismatch (-expected +got):\n%s", diff)
	}
}

func ExampleArgumentScanner() {
	custom := CustomVar{}
	argp := New("custom variable")
	argp.AddOpt(&custom, "", "custom", "")

	_, _, err := argp.parse([]string{"--custom", "1", "/", "2"})
	if err != nil {
		panic(err)
	}
	fmt.Println(custom.Num, "/", custom.Div)
	// Output: 1 / 2
}
