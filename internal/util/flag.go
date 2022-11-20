package util

import (
	"flag"
	"strings"
)

func Flag[T flag.Value](name string, value T, usage string) T {
	flag.Var(value, name, usage)
	return value
}

type stringsFlag []string

func (s stringsFlag) String() string {
	return strings.Join(s, ",")
}

func (s *stringsFlag) Set(v string) error {
	*s = strings.Split(v, ",")
	return nil
}

func StringsFlag(name string, value []string, usage string) *[]string {
	return (*[]string)(Flag(name, (*stringsFlag)(&value), usage))
}
