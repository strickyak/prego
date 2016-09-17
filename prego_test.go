package prego_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)
import . "github.com/strickyak/prego"

var Vars = make(map[string]bool)

var Macros = map[string]*Macro{
	"DOUBLE": &Macro{
		Args:   []string{"X"},
		Body:   []string{},
		Result: "(X + X)",
	},
	"SUM": &Macro{
		Args:   []string{"A", "B"},
		Body:   []string{"___z := A + B"},
		Result: "(___z)",
	},
}

var SerialMatch = regexp.MustCompile("_[0-9]+_")

func RemoveSerialMatches(s string) string {
	return SerialMatch.ReplaceAllLiteralString(s, "_000_")
}

func TestMacros(t *testing.T) {
	w := bytes.NewBufferString("")
	po := &Po{
		Macros:   Macros,
		Switches: map[string]bool{"alpha": true, "beta": true},
		Stack:    []bool{true},
		W:        w,
		Enabled:  true,
	}

	s1 := `package main
func main() {
  println(macro.DOUBLE(444))
  println(macro.SUM(100, 11))
}`
	// TODO: println(macro.SUM(macro.DOUBLE(100), macro.DOUBLE(11)))

	e1 := `package main
func main() {
  println((444 + 444))
_5___z := 100 +  11;  println((_5___z))
}
`

	po.Lines = strings.Split(s1, "\n")
	i := 0
	for i < len(po.Lines) {
		i = po.DoLine(i)
	}
	r1 := w.String()
	t.Log("s1:", s1, "$")
	t.Log("e1:", e1, "$")
	t.Log("r1:", r1, "$")
	if RemoveSerialMatches(e1) != RemoveSerialMatches(r1) {
		t.Errorf("e1 != r1")
	}
}
