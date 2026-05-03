// Copyright 2019 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package datadriven

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pmezard/go-difflib/difflib"
)

func TestNewLineBetweenDirectives(t *testing.T) {
	RunTestFromString(t, `
# Some testing of sensitivity to newlines
> foo
----
unknown command

> bar
----
unknown command




> bar
----
unknown command
`, func(t *testing.T, d *TestCase) string {
		if d.Input != "sentence" {
			return "unknown command"
		}
		return ""
	})
}

func TestParseLine(t *testing.T) {
	RunTestFromString(t, `
> parse
> xx =
----
here: cannot parse directive at column 6: > xx =

> parse
> xx a=b a=c
----
"xx" [a=b a=c]

> parse
> xx a=b b=c c=[1,2,3]
----
"xx" [a=b b=c c=[1,2,3]]

> parse
> http GET ./alerts query.search="alert 1" header.X-Note='it''s ok'
----
"http" [GET ./alerts query.search="alert 1" header.X-Note='it''s ok']

> parse
> http GET ./alerts query.search="unterminated
----
here: cannot parse directive at column 34: > http GET ./alerts query.search="unterminated
`, func(t *testing.T, d *TestCase) string {
		cmd, args, err := ParseLine(d.Input)
		if err != nil {
			return fmt.Errorf("here: %w", err).Error()
		}
		return fmt.Sprintf("%q %+v", cmd, args)
	})
}

func TestScanQuotedArg(t *testing.T) {
	RunTestFromString(t, `
> cmd search="alert 1" note='it''s ok'
----
"alert 1" / "it's ok"
`, func(t *testing.T, d *TestCase) string {
		var search string
		if err := d.NamedArg("search").Scan(&search); err != nil {
			d.Fatalf(t, "scan search: %v", err)
		}

		var note string
		if err := d.NamedArg("note").Scan(&note); err != nil {
			d.Fatalf(t, "scan note: %v", err)
		}

		return fmt.Sprintf("%q / %q", search, note)
	})
}

func TestSkip(t *testing.T) {
	RunTestFromString(t, `
> skip
----

# This error should never happen.
> error
----
`, func(t *testing.T, d *TestCase) string {
		switch d.Name {
		case "skip":
			// Verify that calling t.Skip() does not fail with an API error on
			// testing.T.
			t.Skip("woo")
		case "error":
			// The skip should mask the error afterwards.
			t.Error("never reached")
		}
		return d.Expected
	})
}

func TestParseHeading(t *testing.T) {
	tests := []struct {
		line      string
		wantDepth int
		wantName  string
		wantOK    bool
		wantErr   string
	}{
		{line: "# comment"},
		{line: "# SUBTEST: top", wantDepth: 1, wantName: "top", wantOK: true},
		{line: "## SUBTEST: inner", wantDepth: 2, wantName: "inner", wantOK: true},
		{line: "### SUBTEST: deeper", wantDepth: 3, wantName: "deeper", wantOK: true},
		{line: "## comment", wantErr: `invalid subtest heading: expected "## SUBTEST: name"`},
		{line: "# SUBTEST:", wantErr: `invalid subtest heading: expected "# SUBTEST: name"`},
	}

	for _, test := range tests {
		t.Run(test.line, func(t *testing.T) {
			gotDepth, gotName, gotOK, err := parseHeading(test.line)
			if test.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", test.wantErr)
				}
				if err.Error() != test.wantErr {
					t.Fatalf("expected error %q, got %q", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotDepth != test.wantDepth || gotName != test.wantName || gotOK != test.wantOK {
				t.Fatalf(
					"expected depth=%d name=%q ok=%t, got depth=%d name=%q ok=%t",
					test.wantDepth,
					test.wantName,
					test.wantOK,
					gotDepth,
					gotName,
					gotOK,
				)
			}
		})
	}
}

func TestHeadingSubTest(t *testing.T) {
	RunTest(t, "testdata/heading_subtest", func(t *testing.T, d *TestCase) string {
		switch d.Name {
		case "hello":
			return d.Args[0].Name + " was said"
		default:
			d.Fatalf(t, "unknown directive: %s", d.Name)
		}
		return d.Expected
	})
}

func TestHeadingSubTestSiblings(t *testing.T) {
	RunTestFromString(t, `
# comment

# SUBTEST: first

# comment inside first

> cmd
----
first

# another comment

# SUBTEST: second

> cmd
----
second
`, func(t *testing.T, d *TestCase) string {
		return t.Name()[strings.LastIndex(t.Name(), "/")+1:]
	})
}

func TestHeadingSubTestNested(t *testing.T) {
	RunTestFromString(t, `
# SUBTEST: outer

# comment inside outer

## SUBTEST: inner

# comment inside inner

> cmd
----
inner
`, func(t *testing.T, d *TestCase) string {
		return t.Name()[strings.LastIndex(t.Name(), "/")+1:]
	})
}

func TestMultiLineTest(t *testing.T) {
	RunTest(t, "testdata/multiline", func(t *testing.T, d *TestCase) string {
		switch d.Name {
		case "small":
			return `just
two lines of output`
		case "large":
			return `more
than
five
lines
of
output`
		default:
			d.Fatalf(t, "unknown directive: %s", d.Name)
		}
		return d.Expected
	})
}

func TestRetry(t *testing.T) {
	var v atomic.Uint32
	RunTest(t, "testdata/retry", func(t *testing.T, d *TestCase) string {
		switch d.Name {
		case "inc":
			n := 1
			if err := d.NamedArg("n").Scan(&n); err != nil {
				d.Fatalf(t, "scan n: %v", err)
			}
			for i := 0; i < n; i++ {
				go func() {
					time.Sleep(time.Duration(rand.Intn(10)) * time.Microsecond)
					v.Add(1)
				}()
			}
			return ""

		case "read":
			return d.Retry(t, func() string {
				return fmt.Sprint(v.Load())
			})

		default:
			d.Fatalf(t, "unknown directive: %s", d.Name)
		}
		return d.Expected
	})
}

func TestDirective(t *testing.T) {
	RunTest(t, "testdata/directive", func(t *testing.T, d *TestCase) string {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "cmd: %s\n%d arguments\n", d.Name, len(d.Args))
		for _, a := range d.Args {
			fmt.Fprintf(&buf, "%s\n", a.String())
		}
		return buf.String()
	})
}

func TestWalk(t *testing.T) {
	Walk(t, "testdata/walk", func(t *testing.T, path string) {
		RunTest(t, path, func(t *testing.T, d *TestCase) string {
			return fmt.Sprintf("test name: %s\n", t.Name())
		})
	})
}

func TestRewrite(t *testing.T) {
	const testDir = "testdata/rewrite"
	files, err := ioutil.ReadDir(testDir)
	if err != nil {
		t.Fatal(err)
	}
	var tests []string
	for _, file := range files {
		if name := file.Name(); strings.HasSuffix(name, "-before") {
			tests = append(tests, strings.TrimSuffix(name, "-before"))
		} else if !strings.HasSuffix(name, "-after") {
			t.Fatalf("all files in %s must end in either -before or -after: %s", testDir, name)
		}
	}
	sort.Strings(tests)

	for _, test := range tests {
		subTest(t, test, func(t testing.TB) {
			path := filepath.Join(testDir, fmt.Sprintf("%s-before", test))
			file, err := os.OpenFile(path, os.O_RDONLY, 0644 /* irrelevant */)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = file.Close() }()

			// Implement a few simple directives.
			handler := func(t testing.TB, d *TestCase) string {
				switch d.Name {
				case "noop":
					return d.Input

				case "duplicate":
					return fmt.Sprintf("%s\n%s", d.Input, d.Input)

				case "duplicate-with-blank":
					return fmt.Sprintf("%s\n\n%s", d.Input, d.Input)

				case "no-output":
					return ""

				default:
					d.Fatalf(t, "unknown directive: %s", d.Name)
					return ""
				}
			}

			rewriteData := runTestInternal(t, path, file, handler, true /* rewrite */)

			afterPath := filepath.Join(testDir, fmt.Sprintf("%s-after", test))
			if *rewriteTestFiles {
				// We are rewriting the rewrite tests. Dump the output into -after files
				out, err := os.Create(afterPath)
				defer func() { _ = out.Close() }()
				if err != nil {
					t.Fatal(err)
				}
				if _, err := out.Write(rewriteData); err != nil {
					t.Fatal(err)
				}
			} else {
				after, err := os.Open(afterPath)
				defer func() { _ = after.Close() }()
				if err != nil {
					t.Fatal(err)
				}
				expected, err := ioutil.ReadAll(after)
				if err != nil {
					t.Fatal(err)
				}

				if string(rewriteData) != string(expected) {
					linesExp := difflib.SplitLines(string(expected))
					linesActual := difflib.SplitLines(string(rewriteData))
					diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
						A: linesExp,
						B: linesActual,
					})
					if err != nil {
						t.Fatal(err)
					}
					t.Errorf("expected didn't match actual:\n%s", diff)
				}
			}
		})
	}
}

func TestScanNoop(t *testing.T) {
	RunTestFromString(t, `
> cmd
----
0
`, func(t *testing.T, d *TestCase) string {
		n := 0
		if err := d.NamedArg("missing").Scan(&n); err != nil {
			d.Fatalf(t, "scan: %v", err)
		}
		return fmt.Sprint(n)
	})
}

func TestScanArgsExpansion(t *testing.T) {
	RunTestFromString(t, `
> cmd vals=[foo, bar, bax]
----
"foo", "bar", "bax"
`, func(t *testing.T, d *TestCase) string {
		var vals []string
		if err := d.NamedArg("vals").Scan(&vals); err != nil {
			d.Fatalf(t, "scan vals: %v", err)
		}
		if len(vals) != 3 {
			d.Fatalf(t, "expected 3 vals, got %d", len(vals))
		}
		return fmt.Sprintf("%q, %q, %q", vals[0], vals[1], vals[2])
	})
}

func TestScanArgsSingle(t *testing.T) {
	scan := func(d *TestCase, dest any) {
		if err := d.NamedArg("vals").Scan(dest); err != nil {
			d.Fatalf(t, "scan vals: %v", err)
		}
	}

	RunTest(t, "testdata/scan_args", func(t *testing.T, d *TestCase) string {
		switch d.Name {
		case "string":
			var dest string
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "bool":
			var dest bool
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "float64":
			var dest float64
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "time.Duration":
			var dest time.Duration
			scan(d, &dest)
			return fmt.Sprintf("%s", dest)
		case "[]string":
			var dest []string
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "[]int":
			var dest []int
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "[]int64":
			var dest []int64
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "[]uint64":
			var dest []uint64
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "[]float64":
			var dest []float64
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		case "map[string]string":
			var dest map[string]string
			scan(d, &dest)
			return fmt.Sprintf("%#v", dest)
		default:
			return fmt.Sprintf("unrecognized type %s", d.Name)
		}
	})
}

func TestString(t *testing.T) {
	RunTest(t, "testdata/string", func(t *testing.T, d *TestCase) string {
		return d.String()
	})
}

func BenchmarkInput(b *testing.B) {
	RunTestFromStringAny(b, `
> foo
----
unknown command
`, func(t testing.TB, d *TestCase) string {
		return "unknown command"
	})
}
