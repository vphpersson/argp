package argp

import (
	"fmt"
	"reflect"
)

type ArgumentScanner interface {
	Help() (string, string)
	Scan(string, []string) (int, error)
}

// Count is a counting option, e.g. -vvv sets the count to 3, or -v=3 sets it directly.
type Count struct {
	I any
}

func (count Count) Help() (string, string) {
	val := ""

	v := reflect.ValueOf(count.I).Elem()
	if !v.IsZero() {
		val = fmt.Sprint(v.Interface())
	}

	return val, TypeName(v.Type())
}

func (count Count) Scan(name string, s []string) (int, error) {
	if reflect.TypeOf(count.I).Kind() != reflect.Ptr {
		return 0, fmt.Errorf("variable must be a pointer to an integer type")
	}

	v := reflect.ValueOf(count.I).Elem()
	t := v.Type().Kind()
	isInt := t == reflect.Int || t == reflect.Int8 || t == reflect.Int16 || t == reflect.Int32 || t == reflect.Int64
	isUint := t == reflect.Uint || t == reflect.Uint8 || t == reflect.Uint16 || t == reflect.Uint32 || t == reflect.Uint64

	if !isInt && !isUint {
		return 0, fmt.Errorf("variable must be a pointer to an integer type")
	}

	if 0 < len(s) && 0 < len(s[0]) && '0' <= s[0][0] && s[0][0] <= '9' {
		// don't parse negatives or other options
		return scanValue(v, s)
	} else if isInt {
		v.SetInt(v.Int() + 1)
	} else {
		v.SetUint(v.Uint() + 1)
	}
	return 0, nil
}

// Append is an option that appends to a slice, e.g. -a 5 -a 6 sets the value to [5 6].
type Append struct {
	I any
}

func (a Append) Help() (string, string) {
	val := ""
	v := reflect.ValueOf(a.I).Elem()
	if !v.IsZero() && 0 < v.Len() {
		val = fmt.Sprint(v.Interface())
	}
	return val, TypeName(v.Type())
}

func (a Append) Scan(name string, s []string) (int, error) {
	if t := reflect.TypeOf(a.I); t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Slice {
		return 0, fmt.Errorf("variable must be a pointer to a slice")
	}
	slice := reflect.ValueOf(a.I).Elem()
	v := reflect.New(slice.Type()).Elem()
	n, err := scanValue(v, s)
	if err == nil {
		slice.Set(reflect.AppendSlice(slice, v))
	}
	return n, err
}
