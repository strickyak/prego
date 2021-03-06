# prego
PREGO:  A preprocessor for golang

## Usage
The command copies stdin to stdout,
making macro substitutions and observing `//#if` directives.
```
   go run main.go --set Foo --source /my/work/macros.po --noinline < /my/work/stuff.po > /my/work/stuff.go
```
Command line arguments can be `--set Label` or `--source filename`.
Both can be used more than once.

Another command line argument can be  --noinline` which will be described later.

Labels that are set this way are considered true for conditional complilation; others are false.

Macros can be defined in files that are sourced, but no output occurs for such files.
Finally stdin is read, and that is what causes output.

Lines in stdin are designed to match lines in stdout,
so line numbers on the generated `.go` file match the input `.po` file.


## Conditonal Compilation Syntax

```
      //#if flag1

      ... code ...

      //#endif
```

## Macro Definition Syntax

```
     func (macro) DOUBLE(x) {
       return ((x) + (x))
     }

     func (macro) SUM(A, B) {
       ___z := (A) + (B)
       return ___z
     }

     func (macro) ASSIGN(v, a) {
       /* Only slash-star comments are OK in a macro. */
       v = (a)
       return
     }
```

Identifiers starting with `___` (like the `___z` above)
are prefixed with `_%d` with some unique number, so temporary
variables can be declared that way, and the macros can be used
recursively.

Macro definitions must only have `return` on the final line
of the body, followed by a line with only `}`.

If the macro is a statement rather than an expression,
the `return` line should be only the word `return`.

As in C macros, you should fully parenthesize both the inputs
and the return value of the macro, to avoid operator priority errors.

## Macro Call Syntax

```
     var x, y int
     macro.ASSIGN(x, 23)
     macro.ASSIGN(y, 23)
     println(macro.DOUBLE(macro.SUM(x, y)))

```

## How macros work:

When a source line is processed, macro calls are processed.
What is emitted is the body lines of the macro, terminated with
semicolons (not newlines), and finally the source line with its
macro call syntax replaced by what comes after the word return.

## Inline Macros

If you say `func (inline)` instead of `func (macro)`,
it works just like `func (macro)`, unless you specify `--noinline`,
in which case it turns into a function defintion instead of a macro.
In this case you need to specify argument types and return type
like you do for a function.

## Number-Suffixed Identifiers

If syntax appears like `n[id]` where n in a decimal number and id
is an identifier, it turns into `id_n`.  This lets you turn one
parameter name into several related identifiers.  The choice of syntax
assumes that suffixing numbers does not appear in the target
langauge.  This is the case for Go (but not for C!).

Example:  `2[bird]` becomes `bird_2`.
