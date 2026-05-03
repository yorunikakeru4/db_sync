# datadriven

Data-driven test framework for Go. Test cases live in plain text files — each case declares a
command, optional input, and expected output. Your handler returns the actual output as a string;
the framework diffs it against expected and fails on mismatch. Pass `-rewrite` to regenerate
expected output in-place instead of failing.

Forked from [cockroachdb/datadriven](https://github.com/cockroachdb/datadriven).

## Test file format

```
> eval
1 + 1
----
2
```

Lines starting with `#` are comments. `//` is also accepted for compatibility. A single file holds
many cases separated by blank lines:

```
# add two numbers
> eval
1 + 1
----
2

# multiply
> eval
6 * 7
----
42
```

**Arguments** follow the command on the same line:

```
> eval precision=2
1 / 3
----
0.33
```

Arguments are parsed into `tc.Args` — they do not appear in `tc.Input`. Multiple values are
supported via square brackets:

```
> eval cols=[x, y, z]
x + y + z
----
x(1) + y(2) + z(3) = 6
```

Scalar values may be quoted when they contain spaces:

```txt
> cmd search="alert 1" note='it''s ok'
```

Quoted values are preserved as YAML scalars, so `Arg.Scan(&dest)` decodes them exactly like other
datadriven argument values.

**Input is optional** — omit it when the command needs no body.

**Blank lines in expected output** — use `---- BEGIN` / `---- END` delimiters when the command
returns multi-paragraph text where blank lines are meaningful (e.g. formatted query plans, diff
output):

```
> cmd
---- BEGIN
first paragraph

second paragraph
---- END
```

**Long lines** can be wrapped with `\`:

```
> eval precision=2 \
    mode=floor
1 / 3
----
0
```

**Subtests** group cases into named `t.Run` blocks. Use `# SUBTEST: name` for a top-level subtest,
`## SUBTEST: name` for a nested subtest, and so on. The number of `#` characters determines nesting
depth, and subtests are closed automatically when a heading at the same or shallower depth appears
(or at EOF):

```
# SUBTEST: sum

> eval
1 + 1
----
2

# SUBTEST: product

> eval
2 * 3
----
6
```

Nested subtests use deeper headings:

```
# SUBTEST: arithmetic

## SUBTEST: addition

> eval
1 + 1
----
2

## SUBTEST: multiplication

> eval
2 * 3
----
6
```
## Writing a test

### Single file

```go
// handle is passed to RunTest and called once per test case.
func handle(t *testing.T, tc *datadriven.TestCase) string {
    switch tc.Name {
    case "eval":
        return evaluate(tc.Input)
    default:
        tc.Fatalf(t, "unsupported command: %s", tc.Name)
        return ""
    }
}

func TestEval(t *testing.T) {
    datadriven.RunTest(t, "testdata/eval", handle)
}
```

### Directory of files

`Walk` recurses into subdirectories and creates a `t.Run` subtest per file:

```go
func TestEval(t *testing.T) {
    datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
        if strings.HasSuffix(path, ".md") {
            t.Skip()
        }
        datadriven.RunTest(t, path, handle)
    })
}
```

## TestCase reference

```go
type TestCase struct {
    Pos      string          // "testdata/eval:12"
    Name     string          // command name
    Args     []Arg           // parsed arguments
    NamedArgs map[string]Arg // args indexed by key for fast lookup
    Input    string          // text between the command line and ----
    Expected string          // text after ----
    Rewrite  bool            // true when -rewrite is set
}
```

`tc.Fatalf` is preferred over `t.Fatalf` — it prepends the `file:line` position of the failing
case to the error message, making failures much easier to locate.

**Reading arguments:**

```go
// Positional argument (key only, no value): "cmd foo"
name := tc.Arg(0).Name

// Named argument, scalar: "cmd limit=10"
// Scan returns nil and leaves dest unchanged when the argument is absent.
var limit int
if err := tc.NamedArg("limit").Scan(&limit); err != nil {
    tc.Fatalf(t, "scan limit: %v", err)
}

// Named argument, slice: "cmd cols=[x, y, z]"
var cols []string
if err := tc.NamedArg("cols").Scan(&cols); err != nil {
    tc.Fatalf(t, "scan cols: %v", err)
}

// Named argument, JSON/YAML map: "cmd agg={"foo":"sum"}"
agg := make(map[string]string)
if err := tc.NamedArg("agg").Scan(&agg); err != nil {
    tc.Fatalf(t, "scan agg: %v", err)
}

// Check presence before scanning
if arg := tc.NamedArg("group"); !arg.IsZero() {
    var group string
    if err := arg.Scan(&group); err != nil {
        tc.Fatalf(t, "scan group: %v", err)
    }
}
```

## Flags

`-rewrite` is the flag you'll use most. The others are rarely needed.

| Flag | Description |
|---|---|
| `-rewrite` | Overwrite expected output with actual results instead of failing |
| `-datadriven-quiet` | Suppress per-case log output (also: `DATADRIVEN_QUIET_LOG=true`) |

Regenerate all expected output after a broad behaviour change:

```bash
go test -rewrite ./...
```

Always review the diff before committing.

## ClearResults

Strips expected output from a test file while keeping commands and inputs intact.

Use this when `-rewrite` would produce garbage rather than fix things — for example, if a
serialization bug causes every result to be malformed, running `-rewrite` would commit broken output
for every case. `ClearResults` lets you wipe expected output first, fix the bug, then rewrite
cleanly:

```go
err := datadriven.ClearResults("testdata/eval")
```
