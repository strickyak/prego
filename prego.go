package prego

import . "fmt"

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"text/scanner"
)

var MatchPlusBuildPrego = regexp.MustCompile(`^\s*//\s*[+]build\s+(prego)\s*$`)
var MatchPlusBuild = regexp.MustCompile(`^\s*//\s*[+]build\s+(.*)$`)

// MatchDirective looks for "//#word" (for some word) (as first nonwhite chars)
// possibly followed by other stuff.
var MatchDirective = regexp.MustCompile(`[ \t]*//\s*[#]\s*([a-z]+)[ \t]*(.*)*$`)

var MatchBeforeSlashSlash = regexp.MustCompile("(.*?)//")
var MatchBug = regexp.MustCompile("[\"'`\\\\]")
var MatchMacroDef = regexp.MustCompile(`^\s*func\s*[(]\s*(inline|macro)\s*[)]\s*([A-Za-z0-9_]+)\s*[(]([^()]*)[)]([^{}]*)[{]`)
var MatchMacroReturn = regexp.MustCompile(`^\s*return\s*(.*)$`)
var MatchMacroFinal = regexp.MustCompile(`^\s*[}]\s*$`)
var MatchMacroCall = regexp.MustCompile(`\b(?:inline|macro)\s*[.]\s*([A-Za-z0-9_]+)\s*[(]`)
var MatchIdentifier = regexp.MustCompile(`[A-Za-z0-9_]+`)
var MatchFormalArg = regexp.MustCompile(`([A-Za-z0-9_]+) *([^(),]*)`)

var MatchEndsWithOpenCurly = regexp.MustCompile(`.*{\s*`)
var MatchBeginsWithIf = regexp.MustCompile(`\s*if\s+.*`)
var MatchOneQuote = regexp.MustCompile(`^[^"]*["][^"]*$`)

var logWarning = log.New(os.Stderr, "", 0)

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
	I        int
	Inlining bool
}

func (po *Po) Fatalf(s string, args ...interface{}) {
	args = append(args, po.I+1)
	log.Fatalf("prego preprocessor: ERROR: "+s+" (at line %d)", args...)
}

func (po *Po) Warningf(s string, args ...interface{}) {
	args = append(args, po.I+1)
	logWarning.Printf("prego preprocessor: WARNING: "+s+" (at line %d)", args...)
}

func (po *Po) replaceFromMap(s string, subs map[string]string, serial int) string {
	if z, ok := subs[s]; ok {
		return z
	}
	if strings.HasPrefix(s, "__") {
		return Sprintf("_%d%s", serial, s)
	}
	return s
}

func (po *Po) SubstitueMacros(s string) string {
	for {
		z := po.SubstitueMacrosOnce(s)
		if z == s {
			return z
		}
		s = z
	}
}

func (po *Po) SubstitueMacrosOnce(s string) string {
	serial := po.Serial
	po.Serial++

	m := MatchMacroCall.FindStringSubmatchIndex(s)
	if m == nil {
		return s
	}

	if len(m) != 4 {
		po.Fatalf("bad len from MatchMacroCall.FindStringSubmatchIndex")
	}

	front := s[:m[0]]
	name := s[m[2]:m[3]]
	rest := s[m[1]:]

	if MatchOneQuote.FindStringSubmatch(front) != nil {
		// Grand hack.   We must be in a string, so don't expand what looked like a macro.
		return s
	}

	var argwords []string
	for {
		n := ParseArg(rest)
		// TODO: this needs work (what about white space, trailing `,` ... )
		if n == 0 && rest[0] == ')' {
			rest = rest[n+1:]
			break
		}
		word := rest[:n]
		argwords = append(argwords, word)
		delim := rest[n]
		rest = rest[n+1:]
		if delim == ')' {
			break
		}
	}

	macro, ok := po.Macros[name]
	if !ok {
		po.Fatalf("unknown macro: %q", name)
	}
	if len(argwords) != len(macro.Args) {
		po.Fatalf("got %d args for macro %q, but wanted %d args", len(argwords), name, len(macro.Args))
	}

	subs := make(map[string]string)
	for i, arg := range macro.Args {
		subs[arg] = argwords[i]
	}

	var z string
	if !macro.Inline || po.Inlining {

		replacer := func(word string) string { return po.replaceFromMap(word, subs, serial) }

		Fprintf(po.W, " /*macro:%s{*/ ", name)
		for _, line := range macro.Body {
			if len(line) > 0 {
				l2 := MatchIdentifier.ReplaceAllStringFunc(line, replacer)
				l3 := po.SubstitueMacros(l2)
				if MatchBeginsWithIf.FindStringSubmatch(l3) != nil {
					Fprint(po.W, " ;;; ")
				}
				if MatchEndsWithOpenCurly.FindStringSubmatch(l3) == nil {
					Fprint(po.W, l3+"; ")
				} else {
					// Helps with the newline between `switch` and `case`.
					Fprint(po.W, l3+" ")
				}
			}
		}
		Fprintf(po.W, " /*macro}*/ ")

		z = MatchIdentifier.ReplaceAllStringFunc(macro.Result, replacer)

	} else {
		z = Sprintf("/*noinline:*/%s(%s)", name, strings.Join(argwords, ", "))
	}
	return front + z + rest
}

func (po *Po) calculateIsEnabled() bool {
	for _, e := range po.Stack {
		if !e {
			return false
		}
	}
	return true
}

func Tidy(t string) string {
	r := bytes.NewBufferString(t)
	var s scanner.Scanner
	s.Mode = scanner.GoTokens
	s.Whitespace = scanner.GoWhitespace
	s.Init(r)
	var w bytes.Buffer
	for {
		r := s.Peek()
		if r == scanner.EOF {
			break
		}
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			w.WriteRune(r)
			s.Next()
			continue
		}

		token := s.Scan()
		if token == scanner.EOF {
			break
		}
		if token < 0 {
			w.WriteString(" ")
		}
		w.WriteString(s.TokenText())
		if token < 0 {
			w.WriteString(" ")
		}
	}
	return w.String()
}

func (po *Po) DoLine(i int) int {
	s := po.Lines[i]
	lineNum := i + 1
	mplusprego := MatchPlusBuildPrego.FindStringSubmatch(s)
	if mplusprego != nil {
		Fprintf(po.W, "//\n")
		return i + 1
	}
	mplus := MatchPlusBuild.FindStringSubmatch(s)
	if mplus != nil {
		// Delete all "prego" from the +build line.
		Fprintf(po.W, "// +build %s\n", strings.Replace(mplus[1], "prego", "", -1))
		return i + 1
	}

	// First process cond (//#if & //#endif).
	m := MatchDirective.FindStringSubmatch(s)
	if m != nil {
		switch m[1] {
		case "if":
			pred := false
			for _, term := range strings.Split(m[2], "||") {
				term = strings.Trim(term, " \t")
				if MatchIdentifier.FindString(term) != term {
					po.Fatalf("Line %d: not an identifier: %q", lineNum, term)
				}
				p, _ := po.Switches[term]
				if p {
					pred = true
				}
			}
			po.Stack = append(po.Stack, pred)
		case "endif":
			n := len(po.Stack)
			if n < 2 {
				po.Fatalf("Line %d: Unmatched #endif", lineNum)
			}
			po.Stack = po.Stack[:n-1]
		default:
			po.Warningf("Line %d: Unknown control: %q", lineNum, m[1])
			Fprintf(po.W, s+"\n")
			return i + 1
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

		var argwords []string
		for _, argword := range strings.Split(arglist, ",") {
			a := strings.Trim(argword, " \t")
			if len(a) == 0 {
				continue
			}

			mfa := MatchFormalArg.FindStringSubmatch(a)
			if mfa == nil {
				po.Fatalf("MatchFormalArg fails on %q", a)
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
		Fprintln(po.W, po.SubstitueMacros(Tidy(s)))
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
	po.I = 0
	for po.I < len(lines) {
		po.I = po.DoLine(po.I)
	}
}
