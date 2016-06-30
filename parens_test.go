package prego

import (
	"testing"
)

type Data struct {
	s string
	n int
}

var data = []Data{
	{"foo,", 3},
	{"(foo),", 5},
	{"(f()),", 5},
}

func TestParens(t *testing.T) {
	for _, a := range data {
		got := ParseArg(a.s)
		if got != a.n {
			t.Errorf("got %d wanted %d for ParseArg(%q)", got, a.n, a.s)
		}
	}
}
