package prego_test

import (
	"bytes"
	"strings"
	"testing"
)
import . "github.com/strickyak/prego"

var Vars = make(map[string]bool)

var Macros = map[string]*Macro{
	"DOUBLE": &Macro{
		Args:   []string{"X"},
		Body:   "",
		Result: "(X + X)",
	},
	"SUM": &Macro{
		Args:   []string{"A", "B"},
		Body:   "___z := A + B",
		Result: "(___z)",
	},
}

func TestMacros(t *testing.T) {
	w := bytes.NewBufferString("")
	po := &Po{
		Macros:   Macros,
		Switches: map[string]bool{"alpha": true, "beta": true},
		Stack:    []bool{true},
		W:        w,
	}

	s1 := `package main
func main() {
  println(inline.DOUBLE(444))
  println(inline.SUM(100, 11))
}`
	// TODO: println(inline.SUM(inline.DOUBLE(100), inline.DOUBLE(11)))

	e1 := `package main
func main() {
  println((444 + 444))
_5___z := 100 +  11;  println((_5___z))
}
`

	for i, line := range strings.Split(s1, "\n") {
		po.DoLine(i+1, line)
	}
	r1 := w.String()
	t.Log("s1:", s1, "$")
	t.Log("e1:", e1, "$")
	t.Log("r1:", r1, "$")
	if e1 != r1 {
		t.Errorf("e1 != r1")
	}
}
