package prego

import . "fmt"

import (
	"bufio"
	"io"
	"log"
	"regexp"
	"strings"
)

// MatchCond looks for "//#word" (for some word) (as first nonwhite chars)
// followed by possibly identifier (after some whitespace).
var MatchCond = regexp.MustCompile(`[ \t]*//\s*[#]\s*([a-z]+)[ \t]*([A-Za-z0-9_]*)[ \t]*$`)

var MatchMacroDef = regexp.MustCompile(`^\s*func\s*[(]\s*macro\s*[)]\s*([A-Za-z0-9_]+)\s*[(]([^()]*)[)]\s*[{]`)
var MatchMacroReturn = regexp.MustCompile(`^\s*return\s*(.*)$`)
var MatchMacroFinal = regexp.MustCompile(`^\s*[}]\s*$`)
var MatchMacroCall = regexp.MustCompile(`\bmacro[.]([A-Za-z0-9_]+)[(]`)
var MatchIdentifier = regexp.MustCompile(`[A-Za-z0-9_]+`)

type Macro struct {
	Args   []string
	Body   []string
	Result string
}

type Po struct {
	Macros   map[string]*Macro
	Switches map[string]bool
	Stack    []bool
	W        io.Writer
	Serial   int
	Enabled  bool
	Lines    []string
}

func Fatalf(s string, args ...interface{}) {
	log.Fatalf("prego preprocessor: ERROR: "+s, args...)
}

func (po *Po) replaceFromMap(s string, subs map[string]string, serial int) string {
	if z, ok := subs[s]; ok {
		return z
	}
	if strings.HasPrefix(s, "___") {
		return Sprintf("_%d%s", serial, s)
	}
	return s
}

func (po *Po) SubstitueMacros(s string) string {
	serial := po.Serial
	po.Serial++

	m := MatchMacroCall.FindStringSubmatchIndex(s)
	if m == nil {
		return s
	}

	if len(m) != 4 {
		Fatalf("bad len from MatchMacroCall.FindStringSubmatchIndex")
	}

	front := s[:m[0]]
	name := s[m[2]:m[3]]
	rest := s[m[1]:]

	var argwords []string
	for {
		n := ParseArg(rest)
		word := po.SubstitueMacros(rest[:n])
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
	replacer := func(word string) string { return po.replaceFromMap(word, subs, serial) }

	for _, line := range macro.Body {
		if len(line) > 0 {
			l2 := MatchIdentifier.ReplaceAllStringFunc(line, replacer)
			l3 := po.SubstitueMacros(l2)
			Fprint(po.W, l3+";")
		}
	}

	z := MatchIdentifier.ReplaceAllStringFunc(macro.Result, replacer)
	return front + z + po.SubstitueMacros(rest)
}

func (po *Po) calculateIsEnabled() bool {
	for _, e := range po.Stack {
		if !e {
			return false
		}
	}
	return true
}

func (po *Po) DoLine(i int) int {
	s := po.Lines[i]
	lineNum := i + 1

	// First process cond (//#if & //#endif).
	m := MatchCond.FindStringSubmatch(s)
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
		// The directive becomes a blank line below.
		s = ""
		po.Enabled = po.calculateIsEnabled()
	}

	// Treat as a blank line, if not Enabled.
	if !po.Enabled {
		s = ""
	}

	// Next process macro definitions.
	mm := MatchMacroDef.FindStringSubmatch(s)
	if mm != nil {
		name := mm[1]
		arglist := mm[2]

		var argwords []string
		for _, argword := range strings.Split(arglist, ",") {
			a := strings.Trim(argword, " \t")
			if len(a) == 0 {
				continue
			}
			if MatchIdentifier.FindString(a) != a {
				panic("not an identifier: " + a)
			}
			argwords = append(argwords, a)
		}

		var body []string
		var result string
		for {
			i++
			lineNum++
			Fprintln(po.W, "")
			b := po.Lines[i]
			mr := MatchMacroReturn.FindStringSubmatch(b)
			if mr == nil {
				// Just a body line.
				body = append(body, b)
			} else {
				// It's the return line.
				result = mr[1]
				break
			}
		}

		// Read one more line, which must close the macro.
		i++
		lineNum++
		b := po.Lines[i]
		if MatchMacroFinal.FindString(b) == "" {
			panic("Expected final CloseBrace alone on a line after macro return line")
		}
		Fprintln(po.W, "")

		if _, ok := po.Macros[name]; ok {
			panic("macro already defined: " + name)
		}

		po.Macros[name] = &Macro{
			Args:   argwords,
			Body:   body,
			Result: result,
		}

		s = ""
	}

	Fprintln(po.W, po.SubstitueMacros(s))
	return i + 1
}

func (po *Po) Slurp(r io.Reader, w io.Writer) {
	bs := bufio.NewScanner(r)
	var lines []string
	for bs.Scan() {
		lines = append(lines, bs.Text())
	}

	po.W = w
	po.Lines = lines
	i := 0
	for i < len(lines) {
		i = po.DoLine(i)
	}
}
