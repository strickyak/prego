/*
Preprocess a .po file into a .go file.

Usage:

   PO=sym1,sym2,sym3...  go run preprocess/main.go < runtime.po > runtime.go

Po Syntax:

  //#if sym1
  ...
  //#endif

*/
package prego

import . "fmt"
import (
	//"bufio"
	"io"
	"log"
	//"os"
	"regexp"
	"strings"
)

var F = Sprintf

var Match = regexp.MustCompile(`[ \t]*//#([a-z]+)[ \t]*([A-Za-z0-9_]*)[ \t]*$`).FindStringSubmatch

var MatchMacroCall = regexp.MustCompile(`\binline[.]([A-Za-z0-9_]+)[(]([^()]*)[)]`)
var MatchMacroCall2 = regexp.MustCompile(`\binline[.]([A-Za-z0-9_]+)[(]`)

var MatchIdentifier = regexp.MustCompile(`[A-Za-z0-9_]+`)

type Macro struct {
	Args   []string
	Body   string
	Result string
}

type Po struct {
	Macros   map[string]*Macro
	Switches map[string]bool
	Stack    []bool
	W        io.Writer
}

func Fatalf(s string, args ...interface{}) {
	log.Fatalf("po preprocessor: ERROR: "+s, args...)
}

func (po *Po) replaceFromMap(s string, subs map[string]string) string {
	if z, ok := subs[s]; ok {
		return z
	}
	return s
}

func (po *Po) SubstitueMacros(s string) string {
	println("// SubstitueMacros:", s)

	/////////////// old
	/*
			m := MatchMacroCall.FindStringSubmatch(s)
		 	if len(m) != 3 {
				Fatalf("bad len from MatchMacroCall.FindStringSubmatch")
			}
			name := m[1]
			argtext := m[2]
			argwords := strings.Split(argtext, ",")
	*/

	/////////////// new
	m := MatchMacroCall2.FindStringSubmatchIndex(s)
	if m == nil {
		println(F("No Match, returning %q", s))
		return s
	}

	if len(m) != 4 {
		Fatalf("bad len from MatchMacroCall2.FindStringSubmatchIndex")
	}

	front := s[:m[0]]
	name := s[m[2]:m[3]]
	rest := s[m[1]:]
	println(F("Match, %q ... %q ... %q", front, name, rest))

	var argwords []string
	for {
		n := ParseArg(rest)
		println(F("ParseArg < %q > %d", rest, n))
		word := po.SubstitueMacros(rest[:n])
		println(F("word=", word))
		argwords = append(argwords, word)
		delim := rest[n]
		rest = rest[n+1:]
		if delim == ')' {
			break
		}
	}

	macro, ok := po.Macros[name]
	if !ok {
		Fatalf("unknown macro: %q", name)
	}
	if len(argwords) != len(macro.Args) {
		Fatalf("got %d args for macro %q, but wanted %d args", len(argwords), name, len(macro.Args))
	}

	subs := make(map[string]string)
	for i, arg := range macro.Args {
		subs[arg] = argwords[i]
	}
	replacer := func(word string) string { return po.replaceFromMap(word, subs) }

	for _, line := range strings.Split(macro.Body, "\n") {
		if len(line) > 0 {
			l2 := MatchIdentifier.ReplaceAllStringFunc(line, replacer)
			l3 := po.SubstitueMacros(l2)
			Fprintln(po.W, l3)
		}
	}

	z := MatchIdentifier.ReplaceAllStringFunc(macro.Result, replacer)
	return front + z + po.SubstitueMacros(rest)
}

func (po *Po) DoLine(lineNum int, s string) {
	m := Match(s)

	if m != nil {
		switch m[1] {
		case "if":
			pred, _ := po.Switches[m[2]]
			po.Stack = append(po.Stack, pred)
		case "endif":
			n := len(po.Stack)
			if n < 2 {
				Fatalf("Line %d: Unmatched #endif", lineNum)
			}
			po.Stack = po.Stack[:n-1]
		default:
			Fatalf("Line %d: Unknown control: %q", lineNum, m[1])
		}
		Fprintln(po.W, "")

	} else {
		printing := true
		for _, e := range po.Stack {
			if !e {
				printing = false
			}
		}

		if printing {
			Fprintln(po.W, po.SubstitueMacros(s))
		} else {
			Fprintln(po.W, "")
		}
	}
}
