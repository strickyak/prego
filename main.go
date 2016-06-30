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

func main() {
	env := os.Getenv("PO")

	switches := make(map[string]bool)
	for _, s := range strings.Split(env, ",") {
		switches[s] = true
	}

	po := &Po{
		Macros:   Macros,
		Switches: switches,
		Stack:    []bool{true},
		W:        os.Stdout,
	}

	bs := bufio.NewScanner(os.Stdin)
	lineNum := 0
	for bs.Scan() {
		lineNum++
		po.DoLine(lineNum, bs.Text())
	}
}
