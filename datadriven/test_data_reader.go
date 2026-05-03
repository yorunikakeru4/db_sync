// Copyright 2018 The Cockroach Authors.
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
	"io"
	"strings"
	"testing"
)

// headingEntry tracks an open heading-based subtest (# SUBTEST: / ## SUBTEST: / ...).
type headingEntry struct {
	depth int    // number of '#' characters (1 for # SUBTEST:, 2 for ## SUBTEST:, etc.)
	name  string // full subtest path (e.g. "foo/bar")
}

type testDataReader struct {
	sourceName   string
	reader       io.Reader
	scanner      *lineScanner
	data         TestCase
	rewrite      *bytes.Buffer
	headingStack []headingEntry // open heading-based subtests
	pending      []TestCase     // synthetic directives waiting to be yielded
}

func newTestDataReader(
	t testing.TB, sourceName string, file io.Reader, record bool,
) *testDataReader {
	t.Helper()

	var rewrite *bytes.Buffer
	if record {
		rewrite = &bytes.Buffer{}
	}
	return &testDataReader{
		sourceName: sourceName,
		reader:     file,
		scanner:    newLineScanner(file),
		rewrite:    rewrite,
	}
}

func (r *testDataReader) Next(t testing.TB) bool {
	t.Helper()

	// Yield pending synthetic directives (heading-based subtest open/close).
	if len(r.pending) > 0 {
		r.data = r.pending[0]
		r.pending = r.pending[1:]
		return true
	}

	for r.scanner.Scan() {
		// Ensure to not re-initialize r.data unless a line is read
		// successfully. The reason is that we want to keep the last
		// stored value of `Pos` after encountering EOF, to produce useful
		// error messages.
		r.data = TestCase{Path: r.sourceName}
		line := r.scanner.Text()
		r.emit(line)

		// Update Pos early so that a late error message has an updated
		// position.
		pos := fmt.Sprintf("%s:%d", r.sourceName, r.scanner.line)
		r.data.Pos = pos

		line = strings.TrimSpace(line)

		// Hash-prefixed lines are either comments or heading-based subtests.
		if strings.HasPrefix(line, "#") {
			depth, name, ok, err := parseHeading(line)
			if err != nil {
				t.Fatalf("%s: %v", pos, err)
			}
			if !ok {
				continue
			}

			r.closeHeadings(depth, pos)

			// Build full path from parent heading.
			fullName := name
			if n := len(r.headingStack); n > 0 {
				fullName = r.headingStack[n-1].name + "/" + name
			}
			r.headingStack = append(r.headingStack, headingEntry{depth: depth, name: fullName})

			// Queue a synthetic subtest-start directive.
			r.pending = append(r.pending, TestCase{
				Path: r.sourceName,
				Pos:  pos,
				Name: "subtest",
				Args: []Arg{{Name: fullName}},
			})

			r.data = r.pending[0]
			r.pending = r.pending[1:]
			return true
		}

		if strings.HasPrefix(line, "//") {
			// Skip comment lines.
			continue
		}

		// Support wrapping directive lines using \, for example:
		//   > build-scalar \
		//   > vars(int)
		for strings.HasSuffix(line, `\`) && r.scanner.Scan() {
			nextLine := r.scanner.Text()
			r.emit(nextLine)
			line = strings.TrimSuffix(line, `\`) + " " + strings.TrimSpace(nextLine)
		}

		cmd, args, err := ParseLine(line)
		if err != nil {
			t.Fatalf("%s: %v", pos, err)
		}
		if cmd == "" {
			if line != "" {
				// Non-empty, non-comment line that does not start with "> ".
				// This is almost certainly a missing prefix — fail loudly rather
				// than silently dropping the directive and corrupting the scanner.
				t.Fatalf("%s: directive line must start with \"> \", got: %s", pos, line)
			}
			// Blank line — nothing to do.
			continue
		}

		r.data.Name = cmd
		r.data.Args = args

		r.data.NamedArgs = make(map[string]Arg, len(args))
		for _, arg := range args {
			r.data.NamedArgs[arg.Name] = arg
		}

		if cmd == "subtest" {
			t.Fatalf("%s: use \"# SUBTEST: name\" heading syntax for subtests instead of \"> subtest\"", pos)
		}

		var buf bytes.Buffer
		var separator string
		for r.scanner.Scan() {
			line := r.scanner.Text()
			if line == "----" || line == "---- BEGIN" {
				separator = line
				break
			}

			r.emit(line)
			fmt.Fprintln(&buf, line)
		}

		r.data.Input = strings.TrimSuffix(buf.String(), "\n")
		r.data.Separator = separator

		if separator != "" {
			r.readExpected(t, separator == "---- BEGIN")
		}

		r.data.Rewrite = *rewriteTestFiles
		return true
	}

	// EOF: close all remaining heading-based subtests.
	if len(r.headingStack) > 0 {
		r.closeHeadings(0, r.data.Pos)
		if len(r.pending) > 0 {
			r.data = r.pending[0]
			r.pending = r.pending[1:]
			return true
		}
	}

	return false
}

// closeHeadings synthesizes "subtest end" directives for all open headings
// whose depth is >= toDepth, closing innermost first.
func (r *testDataReader) closeHeadings(toDepth int, pos string) {
	for len(r.headingStack) > 0 {
		top := r.headingStack[len(r.headingStack)-1]
		if top.depth < toDepth {
			break
		}
		r.headingStack = r.headingStack[:len(r.headingStack)-1]
		r.pending = append(r.pending, TestCase{
			Path: r.sourceName,
			Pos:  pos,
			Name: "subtest",
			Args: []Arg{{Name: "end"}},
		})
	}
}

// parseHeading checks whether a hash-prefixed line is a comment or heading-based
// subtest marker. Plain `# comment` lines are comments; subtests use the form
// `# SUBTEST: name`, `## SUBTEST: name`, and so on. Returns the depth, name,
// whether the line opened a subtest, and any syntax error for malformed
// multi-hash headings.
func parseHeading(line string) (depth int, name string, ok bool, err error) {
	if !strings.HasPrefix(line, "#") {
		return 0, "", false, nil
	}

	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}

	rest := strings.TrimSpace(line[i:])
	if !strings.HasPrefix(rest, "SUBTEST:") {
		if i == 1 {
			return 0, "", false, nil
		}
		return 0, "", false, fmt.Errorf(
			"invalid subtest heading: expected %q",
			strings.Repeat("#", i)+" SUBTEST: name",
		)
	}

	name = strings.TrimSpace(strings.TrimPrefix(rest, "SUBTEST:"))
	if name == "" {
		return 0, "", false, fmt.Errorf(
			"invalid subtest heading: expected %q",
			strings.Repeat("#", i)+" SUBTEST: name",
		)
	}

	return i, name, true, nil
}

func (r *testDataReader) readExpected(t testing.TB, allowBlankLines bool) {
	var buf bytes.Buffer

	if allowBlankLines {
		// Read until "---- END" marker.
		for r.scanner.Scan() {
			line := r.scanner.Text()
			if line == "---- END" {
				// Consume the following blank line (if we don't do this, we will
				// emit an extra blank line when rewriting).
				if r.scanner.Scan() && r.scanner.Text() != "" {
					t.Fatal("non-blank line after ---- END")
				}
				break
			}
			fmt.Fprintln(&buf, line)
		}
	} else {
		// Terminate on first blank line.
		if !r.scanner.Scan() {
			r.data.Expected = ""
			return
		}
		for line := r.scanner.Text(); ; {
			if strings.TrimSpace(line) == "" {
				break
			}

			fmt.Fprintln(&buf, line)

			if !r.scanner.Scan() {
				break
			}

			line = r.scanner.Text()
		}
	}

	r.data.Expected = buf.String()
}

func (r *testDataReader) emit(s string) {
	if r.rewrite != nil {
		r.rewrite.WriteString(s)
		r.rewrite.WriteString("\n")
	}
}
