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

// ParseArgs accepts argument pairs:
//    --set varname      (sets the varname true for conditional compilation)
//    -set varname       (same)
//    --source filename   (read for macro definitions; do not output lines)
//    -source filename    (same)
func ParseArgs() {
	args := os.Args[1:] // Leave off command name.

	for len(args) > 1 && strings.HasPrefix(args[0], "-") {
		key, value := args[0], args[1]
		switch key {
		case "-set", "--set":
			Switches[value] = true
		case "-source", "--source":
			Sources = append(Sources, value)
		default:
			log.Fatalf("Unknown command line flag: %q", key)
		}
		args = args[2:]
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

	// Old style switches: from env.
	// TODO: Delete.
	env := os.Getenv("PO")
	for _, s := range strings.Split(env, ",") {
		Switches[s] = true
	}

	po := &prego.Po{
		Macros:   make(map[string]*prego.Macro),
		Switches: Switches,
		Stack:    []bool{true},
		W:        os.Stdout,
		Enabled:  true,
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
