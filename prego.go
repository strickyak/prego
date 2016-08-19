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

var MatchMacroDef = regexp.MustCompile(`^\s*func\s*[(]\s*(inline|macro)\s*[)]\s*([A-Za-z0-9_]+)\s*[(]([^()]*)[)]([^{}]*)[{]`)
var MatchMacroReturn = regexp.MustCompile(`^\s*return\s*(.*)$`)
var MatchMacroFinal = regexp.MustCompile(`^\s*[}]\s*$`)
var MatchMacroCall = regexp.MustCompile(`\b(?:inline|macro)[.]([A-Za-z0-9_]+)[(]`)
var MatchIdentifier = regexp.MustCompile(`[A-Za-z0-9_]+`)
var MatchFormalArg = regexp.MustCompile(`([A-Za-z0-9_]+) *([^(),]*)`)

var MatchNumberSuffix = regexp.MustCompile(`^(.*)\b([0-9]+)\s*[[]\s*([A-Za-z0-9_]+)\s*[]](.*)$`)

type Macro struct {
	Inline  bool // T if `inline`, F if `macro`
	Args    []string
	Body    []string
	Result  string
	RetType string
}

type Po struct {
	Macros   map[string]*Macro
	Switches map[string]bool
	Stack    []bool
	W        io.Writer
	Serial   int
	Enabled  bool
	Lines    []string
	Inlining bool
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

func SuffixNumbers(s string) string {
	for {
		m := MatchNumberSuffix.FindStringSubmatch(s)
		if m == nil {
			break
		}
		s = Sprintf("%s %s_%s %s", m[1], m[3], m[2], m[4])
	}
	return s
}

func (po *Po) SubstitueMacros(s string) string {
	serial := po.Serial
	po.Serial++

	m := MatchMacroCall.FindStringSubmatchIndex(s)
	if m == nil {
		return SuffixNumbers(s)
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
	log.Printf("Applying macro %q formals %v got %v", name, macro.Args, argwords)
	if len(argwords) != len(macro.Args) {
		Fatalf("got %d args for macro %q, but wanted %d args", len(argwords), name, len(macro.Args))
	}

	subs := make(map[string]string)
	for i, arg := range macro.Args {
		subs[arg] = argwords[i]
	}

	var z string
	if !macro.Inline || po.Inlining {

		replacer := func(word string) string { return po.replaceFromMap(word, subs, serial) }

		for _, line := range macro.Body {
			if len(line) > 0 {
				l2 := MatchIdentifier.ReplaceAllStringFunc(line, replacer)
				l3 := po.SubstitueMacros(l2)
				Fprint(po.W, l3+";")
			}
		}

		z = MatchIdentifier.ReplaceAllStringFunc(macro.Result, replacer)

	} else {
		z = Sprintf("%s(%s)", name, strings.Join(argwords, ", "))
	}
	return SuffixNumbers(front + z + po.SubstitueMacros(rest))
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
		inline := mm[1] == "inline" // inline or macro
		name := mm[2]
		arglist := mm[3]
		retType := mm[4]
		log.Printf("Def: name %q arglist %#v", name, arglist)

		var argwords []string
		for _, argword := range strings.Split(arglist, ",") {
			a := strings.Trim(argword, " \t")
			log.Printf("argword: %q", argword)
			if len(a) == 0 {
				continue
			}

			mfa := MatchFormalArg.FindStringSubmatch(a)
			log.Printf("MatchFormalArg: %#v", mfa)
			if mfa == nil {
				Fatalf("MatchFormalArg fails on %q", a)
			}
			argwords = append(argwords, mfa[1])
		}

		if inline && !po.Inlining {
			Fprintf(po.W, "func %s ( %s ) %s {\n", name, arglist, retType)
		} else {
			Fprintln(po.W, "")
		}

		var body []string
		var result string
		for {
			i++
			lineNum++
			b := po.Lines[i]

			if inline && !po.Inlining {
				Fprintln(po.W, po.SubstitueMacros(b))
			} else {
				Fprintln(po.W, "")
			}

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
		if inline && !po.Inlining {
			Fprintln(po.W, "}")
		} else {
			Fprintln(po.W, "")
		}

		if _, ok := po.Macros[name]; ok {
			panic("macro already defined: " + name)
		}

		po.Macros[name] = &Macro{
			Inline:  inline,
			Args:    argwords,
			Body:    body,
			Result:  result,
			RetType: retType,
		}

	} else {
		Fprintln(po.W, po.SubstitueMacros(s))
	}
	return i + 1
}

func (po *Po) Slurp(r io.Reader, w io.Writer) {
	bs := bufio.NewScanner(r)
	bs.Buffer(make([]byte, 100000), 100000000)
	var lines []string
	for bs.Scan() {
		lines = append(lines, bs.Text())
	}
	if err := bs.Err(); err != nil {
		panic(err)
	}

	po.W = w
	po.Lines = lines
	i := 0
	for i < len(lines) {
		i = po.DoLine(i)
	}
}
