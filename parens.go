package prego

// ParseArg parses one argument ending with ',' or ')'
// skipping over nested parentheses.  The number of bytes
// in the argument (not counting the terminating ',' or ')'
// is returned.
func ParseArg(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return i
			} else {
				depth--
			}
		case ',':
			if depth == 0 {
				return i
			}
		}
	}
	panic("argument not terminated by comma or close paren")
}
