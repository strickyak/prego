package prego_test

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)
import . "github.com/strickyak/prego"

var SERIAL = regexp.MustCompile(`_[0-9]+_`)
var COMMENT = regexp.MustCompile(`[/][*].*?[*][/]`)
var WHITE = regexp.MustCompile(`[ \t\n\r]+`)

var Macros = map[string]*Macro{
	"DOUBLE": &Macro{
		Args:   []string{"A"},
		Body:   []string{"__a := A"},
		Result: "(__a + __a)",
	},
	"SUM": &Macro{
		Args:   []string{"A", "B"},
		Body:   []string{"___z := A + B"},
		Result: "(___z)",
	},
	"PRODUCT": &Macro{
		Args:   []string{"A", "B"},
		Body:   []string{"___z := A * B"},
		Result: "(___z)",
	},
}

func RemoveSerialMatches(s string) string {
	s = SERIAL.ReplaceAllLiteralString(s, "_0_")
	s = COMMENT.ReplaceAllLiteralString(s, " ")
	s = WHITE.ReplaceAllLiteralString(s, " ")
	return s
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

	s1 := `
package main
func main() {
  println(macro.DOUBLE(444))
  //#if alpha
  println(macro.SUM(100, 11))

  //#if beta
  println(macro.DOUBLE(macro.PRODUCT(1000, macro.SUM(50, 5))))
  //#endif

  //#if never
  println(666)
  //#endif
  //#endif
  //#if never
  println(666.666)
  //#endif
}
`

	e1 := `
package main
func main () {

  _0__a := 444 ;
  println ( (_0__a + _0__a))

  _0___z := 100 + 11 ;
  println ( (_0___z))

   _0___z := 50 + 5 ;
   _0___z := 1000 * (_0___z);
   _0__a := (_0___z);
   println ( (_0__a + _0__a))
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
	t.Log("Re1:", RemoveSerialMatches(e1), "$")
	t.Log("Rr1:", RemoveSerialMatches(r1), "$")
	if RemoveSerialMatches(e1) != RemoveSerialMatches(r1) {
		t.Errorf("e1 != r1")
	}
}
