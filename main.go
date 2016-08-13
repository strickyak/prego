// +build main

/*
Preprocess a .po file into a .go file.

Usage:

   PO=sym1,sym2,sym3...  go run main.go < program.po > program.go

Po Syntax:

  //#if sym1
  ...
  //#endif

*/
package main

import (
	"bufio"
	//"log"
	"os"
	//"regexp"
	"strings"
)
import . "github.com/strickyak/prego"

var Vars = make(map[string]bool)

/*
var Macros = map[string]*Macro{
	"DOUBLE": &Macro{
		Args:   []string{"X"},
		Body:   "",
		Result: "(X + X)",
	},
	"SUM": &Macro{
		Args:   []string{"A", "B"},
		Body:   "____z := A + B",
		Result: "(____z)",
	},
}
*/

func main() {
	env := os.Getenv("PO")

	switches := make(map[string]bool)
	for _, s := range strings.Split(env, ",") {
		switches[s] = true
	}

	po := &Po{
		Macros:   make(map[string]*Macro),
		Switches: switches,
		Stack:    []bool{true},
		W:        os.Stdout,
    Enabled:  true,
	}

	bs := bufio.NewScanner(os.Stdin)
  var lines []string
	for bs.Scan() {
    lines = append(lines, bs.Text())
	}

  po.Lines = lines
  i := 0
  for i < len(lines) {
    i = po.DoLine(i)
  }
}
