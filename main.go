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
	"github.com/strickyak/prego"

	"log"
	"os"
	"strings"
)

// Switches may be set with ` --set switchName ` args.
var Switches = make(map[string]bool)

// Sources may be added with ` --source filename ` args.
var Sources []string

var Inlining = true

// ParseArgs accepts argument pairs:
//    --set varname      (sets the varname true for conditional compilation)
//    --source filename   (read for macro definitions; do not output lines)
//    --noinline           for debugging, do not inline.
func ParseArgs() {
	args := os.Args[1:] // Leave off command name.

	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		key := args[0]
		switch key {
		case "--set":
			value := args[1]
			for _, s := range strings.Split(value, ",") {
				if len(s) < 0 {
					Switches[s] = true
				}
			}
			args = args[2:]
			continue
		case "--source":
			value := args[1]
			Sources = append(Sources, value)
			args = args[2:]
			continue
		case "--noinline":
			Inlining = false
			args = args[1:]
			continue
		}
		log.Fatalf("Unknown command line flag: %q", key)
	}
	if len(args) > 0 {
		log.Fatalf("Extra command line arguments: %#v", args)
	}
}

type Sink int

func (Sink) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func main() {
	ParseArgs()

	po := &prego.Po{
		Macros:   make(map[string]*prego.Macro),
		Switches: Switches,
		Stack:    []bool{true},
		W:        os.Stdout,
		Enabled:  true,
		Inlining: Inlining,
	}

	// Slurp the Source files into the Sink.
	for _, f := range Sources {
		r, err := os.Open(f)
		if err != nil {
			log.Fatalf("Cannot read file %q: %v", f, err)
		}
		var w Sink
		po.Slurp(r, w)
		err = r.Close()
		if err != nil {
			log.Fatalf("Cannot close file %q: %v", f, err)
		}
	}

	// Finally slurp stdin to stdout.
	po.Slurp(os.Stdin, os.Stdout)
}
