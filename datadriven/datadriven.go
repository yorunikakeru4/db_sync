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
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/pmezard/go-difflib/difflib"
)

var (
	rewriteTestFiles = flag.Bool(
		"rewrite", false,
		"ignore the expected results and rewrite the test files with the actual results from this "+
			"run. Used to update tests when a change affects many cases; please verify the testfile "+
			"diffs carefully!",
	)

	quietLog = flag.Bool(
		"datadriven-quiet", false,
		"avoid echoing the directives and responses from test files.",
	)

	filenameFilter = flag.String("filename", "", "run only the given testdata/<type>/<name> file (without extension)")
)

// Verbose returns true iff -datadriven-quiet was not passed.
func Verbose() bool {
	return testing.Verbose() && !*quietLog
}

// init reads DATADRIVEN_QUIET_LOG so the quiet flag can be set via environment
// in addition to the -datadriven-quiet command-line flag. This allows silencing
// log output in packages that register their own flags and would otherwise error
// on unrecognised flags.
func init() {
	const quietEnvVar = "DATADRIVEN_QUIET_LOG"
	if str, ok := os.LookupEnv(quietEnvVar); ok {
		v, err := strconv.ParseBool(str)
		if err != nil {
			panic(fmt.Sprintf("error parsing %s: %s", quietEnvVar, err))
		}
		*quietLog = v
	}
}

// RunTest invokes a data-driven test. The test cases are contained in a
// separate test file and are dynamically loaded, parsed, and executed by this
// testing framework. By convention, test files are typically located in a
// sub-directory called "testdata". Each test file has the following format:
//
//	<command>[,<command>...] [arg | arg=val | arg=[val1, val2, ...]]...
//	<input to the command>
//	----
//	<expected results>
//
// The command input can contain blank lines. However, by default, the expected
// results cannot contain blank lines. This alternate syntax allows the use of
// blank lines:
//
//	<command>[,<command>...] [arg | arg=val | arg=[val1, val2, ...]]...
//	<input to the command>
//	----
//	----
//	<expected results>
//
//	<more expected results>
//	----
//	----
//
// To execute data-driven tests, pass the path of the test file as well as a
// function which can interpret and execute whatever commands are present in
// the test file. The framework invokes the function, passing it information
// about the test case in a TestCase struct.
//
// The function must return the actual results of the case, which
// RunTest() compares with the expected results. If the two are not
// equal, the test is marked to fail.
//
// Note that RunTest() creates a sub-instance of testing.T for each
// directive in the input file. It is thus unsafe/invalid to call
// e.g. Fatal() or Skip() on the parent testing.T from inside the
// callback function. Use the provided testing.T instance instead.
//
// It is possible for a test to test for an "expected error" as follows:
//   - run the code to test
//   - if an error occurs, report the detail of the error as actual
//     output.
//   - place the expected error details in the expected results
//     in the input file.
//
// It is also possible for a test to report an _unexpected_ test
// error by calling t.Error().
func RunTest(t *testing.T, path string, f func(t *testing.T, tc *TestCase) string) {
	t.Helper()

	RunTestAny(t, path, func(t testing.TB, tc *TestCase) string {
		return f(t.(*testing.T), tc)
	})
}

// RunTestAny is like RunTest but works over a testing.TB.
func RunTestAny(t testing.TB, path string, f func(t testing.TB, tc *TestCase) string) {
	t.Helper()

	mode := os.O_RDONLY
	if *rewriteTestFiles {
		// We only open read-write if rewriting, so as to enable running
		// tests on read-only copies of the source tree.
		mode = os.O_RDWR
	}
	file, err := os.OpenFile(path, mode, 0644 /* irrelevant */)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = file.Close()
	}()
	finfo, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	} else if finfo.IsDir() {
		t.Fatalf("%s is a directory, not a file; consider using datadriven.Walk", path)
	}

	rewriteData := runTestInternal(t, path, file, f, *rewriteTestFiles)
	if *rewriteTestFiles {
		if _, err := file.WriteAt(rewriteData, 0); err != nil {
			t.Fatal(err)
		}
		if err := file.Truncate(int64(len(rewriteData))); err != nil {
			t.Fatal(err)
		}
		if err := file.Sync(); err != nil {
			t.Fatal(err)
		}
	}
}

// RunTestFromString is a version of RunTest which takes the contents of a test
// directly.
func RunTestFromString(t *testing.T, input string, f func(t *testing.T, tc *TestCase) string) {
	t.Helper()
	RunTestFromStringAny(t, input, func(t testing.TB, tc *TestCase) string {
		return f(t.(*testing.T), tc)
	})
}

// RunTestFromStringAny is like RunTestFromString but works with a testing.TB.
func RunTestFromStringAny(t testing.TB, input string, f func(t testing.TB, tc *TestCase) string) {
	t.Helper()
	runTestInternal(t, "<string>" /* sourceName */, strings.NewReader(input), f, *rewriteTestFiles)
}

func runTestInternal(
	t testing.TB,
	sourceName string,
	reader io.Reader,
	f func(t testing.TB, tc *TestCase) string,
	rewrite bool,
) (rewriteOutput []byte) {
	t.Helper()

	r := newTestDataReader(t, sourceName, reader, rewrite)
	for r.Next(t) {
		runDirectiveOrSubTest(t, r, "" /*mandatorySubTestPrefix*/, f)
	}

	if r.rewrite != nil {
		data := r.rewrite.Bytes()
		// Remove any trailing blank line.
		if l := len(data); l > 2 && data[l-1] == '\n' && data[l-2] == '\n' {
			data = data[:l-1]
		}
		return data
	}
	return nil
}

// runDirectiveOrSubTest runs either a "subtest" directive or an
// actual test directive. The "mandatorySubTestPrefix" argument indicates
// a mandatory prefix required from all sub-test names at this point.
func runDirectiveOrSubTest(
	t testing.TB,
	r *testDataReader,
	mandatorySubTestPrefix string,
	f func(testing.TB, *TestCase) string,
) {
	t.Helper()
	if subTestName, ok := isSubTestStart(t, r, mandatorySubTestPrefix); ok {
		runSubTest(subTestName, t, r, f)
	} else {
		runDirective(t, r, f)
	}
	if t.Failed() {
		// If a test has failed with .Error(), we can't expect any
		// subsequent test to be even able to start. Stop processing the
		// file in that case.
		t.FailNow()
	}
}

// runSubTest runs a subtest up to and including the final `subtest
// end`. The opening `subtest` directive has been consumed already.
// The first parameter `subTestName` is the full path to the subtest,
// including the parent subtest names as prefix. This is used to
// validate the nesting and thus prevent mistakes.
func runSubTest(
	subTestName string, t testing.TB, r *testDataReader, f func(testing.TB, *TestCase) string,
) {
	// Remember the current reader position in case we need to spell out
	// an error message below.
	subTestStartPos := r.data.Pos
	// seenSubTestEnd is used below to verify that a "subtest end" directive
	// has been detected (as opposed to EOF).
	seenSubTestEnd := false
	// seenSkip is used below to verify that "Skip" has not been used
	// inside a subtest. See below for details.
	seenSkip := false

	// The name passed to t.Run is the last component in the subtest
	// name, because all components before that are already prefixed by
	// t.Run from the names of the parent sub-tests.
	testingSubTestName := subTestName[strings.LastIndex(subTestName, "/")+1:]

	// Begin the sub-test.
	called := false
	subTest(t, testingSubTestName, func(t testing.TB) {
		called = true
		defer func() {
			// Skips are signalled using Goexit() so we must catch it /
			// remember it here.
			if t.Skipped() {
				seenSkip = true
			}
		}()

		for r.Next(t) {
			if isSubTestEnd(t, r) {
				seenSubTestEnd = true
				return
			}
			runDirectiveOrSubTest(t, r, subTestName+"/" /*mandatorySubTestPrefix*/, f)
		}
	})

	if !called {
		// The subtest was filtered by -run; its function was never called.
		// Drain the reader past this subtest's directives so the parent can
		// continue. In rewrite mode, preserve the existing expected output
		// unchanged so other subtests are not zeroed out.
		for r.Next(t) {
			if isSubTestEnd(t, r) {
				return
			}
			if r.rewrite != nil && r.data.Separator != "" {
				r.rewrite.WriteString(r.data.Separator + "\n")
				r.rewrite.WriteString(r.data.Expected)
				r.rewrite.WriteString("\n")
			}
		}
		return
	}

	if seenSkip {
		// t.Skip() is not yet supported inside a subtest. To add
		// this functionality the following extra complexity is needed:
		// - the test reader must continue to read after the skip
		//   until the end of the subtest, and ignore all the directives in-between.
		// - the rewrite logic must be careful to keep the input as-is
		//   for the skipped sub-test, while proceeding to rewrite for
		//   non-skipped tests.
		r.data.Fatalf(t,
			"cannot use t.Skip inside subtest\n%s: subtest started here", subTestStartPos)
	}

	if seenSubTestEnd && len(r.data.Args) == 2 && r.data.Args[1].Name != subTestName {
		// If a subtest name was provided after "subtest end", ensure that it matches.
		r.data.Fatalf(t,
			"mismatched subtest end directive: expected %q, got %q", r.data.Args[1].Name, subTestName)
	}

	if !seenSubTestEnd && !t.Failed() {
		// We only report missing "subtest end" if there was no error otherwise;
		// for if there was an error, the reading would have stopped.
		r.data.Fatalf(t,
			"EOF encountered without subtest end directive\n%s: subtest started here", subTestStartPos)
	}

}

func isSubTestStart(t testing.TB, r *testDataReader, mandatorySubTestPrefix string) (string, bool) {
	if r.data.Name != "subtest" {
		return "", false
	}
	if len(r.data.Args) != 1 {
		r.data.Fatalf(t, "invalid syntax for subtest")
	}
	subTestName := r.data.Args[0].Name
	if subTestName == "end" {
		r.data.Fatalf(t, "subtest end without corresponding start")
	}
	if !strings.HasPrefix(subTestName, mandatorySubTestPrefix) {
		r.data.Fatalf(t, "name of nested subtest must begin with %q", mandatorySubTestPrefix)
	}
	return subTestName, true
}

func isSubTestEnd(t testing.TB, r *testDataReader) bool {
	if r.data.Name != "subtest" {
		return false
	}
	if len(r.data.Args) == 0 || r.data.Args[0].Name != "end" {
		return false
	}
	if len(r.data.Args) > 2 {
		r.data.Fatalf(t, "invalid syntax for subtest end")
	}
	return true
}

// runDirective runs just one directive in the input.
//
// The stopNow and subTestSkipped booleans are modified by-reference
// instead of returned because the testing module implements t.Skip
// and t.Fatal using panics, and we're not guaranteed to get back to
// the caller via a return in those cases.
func runDirective(t testing.TB, r *testDataReader, f func(testing.TB, *TestCase) string) {
	t.Helper()

	d := &r.data
	actual := func() string {
		fReturned := false
		defer func() {
			if r := recover(); r != nil {
				t.Logf("\npanic during %s:\n%s\n", d.Pos, d.Input)
				panic(r)
			} else if !fReturned {
				// t.Fatal calls runtime.Goexit() which runs the defers but there is no
				// panic to recover.
				t.Logf("\nfatal during %s:\n%s\n", d.Pos, d.Input)
			}
		}()

		// Set up a goroutine to log periodically if the function is taking a long
		// time. This is useful to pinpoint the cause of a test timeout.
		done := make(chan struct{})
		go func() {
			startTime := time.Now()
			for {
				select {
				case <-done:
					return
				case <-time.After(10 * time.Second):
					t.Logf("%s: still running after %s\n", d.Pos, time.Since(startTime))
				}
			}
		}()
		defer func() {
			// Because the channel is unbuffered, we wait here until the goroutine is
			// exiting.
			done <- struct{}{}
		}()

		actual := f(t, d)
		fReturned = true
		if actual != "" && !strings.HasSuffix(actual, "\n") {
			actual += "\n"
		}
		return actual
	}()

	if t.Failed() {
		// If the test has failed with .Error(), then we can't hope it
		// will have produced a useful actual output. Trying to do
		// something with it here would risk corrupting the expected
		// output.
		//
		// Moreover, we can't expect any subsequent test to be even
		// able to start. Stop processing the file in that case.
		t.FailNow()
	}

	// The test has not failed, we can analyze the expected
	// output.
	if r.rewrite != nil {
		if hasBlankLine(actual) {
			r.emit("---- BEGIN")
			r.rewrite.WriteString(actual)
			r.emit("---- END")
			r.emit("")
		} else {
			r.emit("----")
			// Here actual already ends in \n so emit adds a blank line.
			r.emit(actual)
		}
	} else if d.Expected != actual {
		expectedLines := difflib.SplitLines(d.Expected)
		actualLines := difflib.SplitLines(actual)
		if len(expectedLines) > 5 {
			// Print a unified diff if there is a lot of output to compare.
			diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
				Context: 5,
				A:       expectedLines,
				B:       actualLines,
			})
			if err == nil {
				t.Fatalf("\n%s:\n%s\noutput didn't match expected:\n%s", indentLines(d.Pos), indentLines(d.String()), indentLines(diff))
				return
			}
			t.Logf("Failed to produce diff %v", err)
		}
		t.Fatalf("\n%s:\n%s\nexpected:\n%s\nfound:\n%s", d.Pos, indentLines(d.String()), indentLines(d.Expected), indentLines(actual))
	} else if Verbose() {
		t.Logf("\n%s:\n%s  ----\n%s", d.Pos, indentLines(d.String()), indentLines(actual))
	}
	return
}

// Walk goes through all the files in a subdirectory, creating subtests to match
// the file hierarchy; for each "leaf" file, the given function is called.
//
// This can be used in conjunction with RunTest. For example:
//
//	 datadriven.Walk(t, path, func (t *testing.T, path string) {
//	   // initialize per-test state
//	   datadriven.RunTest(t, path, func (t *testing.T, d *datadriven.TestCase) string {
//	    // ...
//	   }
//	 }
//
//	Files:
//	  testdata/typing
//	  testdata/logprops/scan
//	  testdata/logprops/select
//
// If path is "testdata/typing", the function is called once and no subtests
// are created.
//
// If path is "testdata/logprops", the function is called two times, in
// separate subtests /scan, /select.
//
// If path is "testdata", the function is called three times, in subtest
// hierarchy /typing, /logprops/scan, /logprops/select.
func Walk(t *testing.T, path string, f func(t *testing.T, path string)) {
	t.Helper()
	root := path
	WalkAny(t, path, func(t testing.TB, filePath string) {
		if filter := strings.TrimSpace(*filenameFilter); filter != "" {
			if filepath.Clean(filePath) != filepath.Clean(filepath.Join(root, filter)) {
				t.Skip()
				return
			}
		}
		f(t.(*testing.T), filePath)
	})
}

// WalkAny is like Walk but works over a testing.TB.
func WalkAny(t testing.TB, path string, f func(t testing.TB, path string)) {
	finfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !finfo.IsDir() {
		f(t, path)
		return
	}
	files, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if tempFileRe.MatchString(file.Name()) {
			// Temp or hidden file, don't even try processing.
			continue
		}
		subTest(t, cutExt(file.Name()), func(t testing.TB) {
			WalkAny(t, filepath.Join(path, file.Name()), f)
		})
	}
}

// cutExt returns the given file name with the extension removed, if there is
// one.
func cutExt(fileName string) string {
	extStart := len(fileName) - len(filepath.Ext(fileName))
	return fileName[:extStart]
}

// ClearResults strips expected output from a test file while keeping commands
// and inputs intact. Use this when -rewrite would produce garbage — for example
// after a serialization bug that makes every result malformed. Clear first, fix
// the bug, then rewrite cleanly.
func ClearResults(path string) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0644 /* irrelevant */)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	finfo, err := file.Stat()
	if err != nil {
		return err
	}

	if finfo.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", path)
	}

	runTestInternal(
		&testing.T{}, path, file,
		func(testing.TB, *TestCase) string { return "" },
		true, /* rewrite */
	)

	return nil
}

// Ignore files named .XXXX, XXX~ or #XXX#.
var tempFileRe = regexp.MustCompile(`(^\..*)|(.*~$)|(^#.*#$)`)

// TestCase contains information about one data-driven test case that was
// parsed from the test file.
type TestCase struct {
	// Path is the file path of the test file containing this case.
	Path string

	// Pos is a file:line prefix for the input test file, suitable for
	// inclusion in logs and error messages.
	Pos string

	// Name is the first string on the directive line (up to the first whitespace).
	Name string

	// Args contains the k/v arguments to the command.
	Args []Arg

	// NamedArgs indexes Args by key for O(1) named lookup via NamedArg.
	NamedArgs map[string]Arg

	// Input is the text between the first directive line and the ---- separator.
	Input string

	// Separator is the separator line that precedes Expected ("----",
	// "---- BEGIN", or "" if there is no expected section).
	Separator string

	// Expected is the value below the ---- separator. In most cases,
	// tests need not check this, and instead return their own actual
	// output.
	// This field is provided so that a test can perform an early return
	// with "return d.Expected" to signal that nothing has changed.
	Expected string

	// Rewrite is set if the test is being run with the -rewrite flag.
	Rewrite bool
}

// String renders the entire testcase. The string ends in a newline.
func (tc *TestCase) String() string {
	fields := make([]string, 0, len(tc.Args)+1)
	fields = append(fields, tc.Name)
	for i := range tc.Args {
		fields = append(fields, tc.Args[i].String())
	}
	s := "> " + strings.Join(fields, " ")
	if tc.Input == "" {
		return s + "\n"
	}
	return fmt.Sprintf("%s\n%s", s, tc.Input)
}

// Arg returns the argument at idx. Returns a zero Arg if idx is
// out of range — check with IsZero before use.
func (tc *TestCase) Arg(idx int) (arg Arg) {
	if idx < 0 || idx > len(tc.Args)-1 {
		return arg
	}
	return tc.Args[idx]
}

// NamedArg returns the argument with the given key. Returns a zero Arg if
// no such argument exists — check with IsZero before use.
func (tc *TestCase) NamedArg(key string) (arg Arg) {
	return tc.NamedArgs[key]
}

// Retry is used for tests that depend on background goroutines to finish work.
// It takes a function that produces the output of the testcase and calls it
// repeatedly until it matches the expected output (for at most 1 second).
//
// Returns the last value returned by f (which can be directly returned from the
// function passed to RunTest).
//
// If --rewrite is used, just sleeps for 100ms.
func (tc *TestCase) Retry(tb testing.TB, f func() string) string {
	return tc.RetryFor(tb, time.Second, f)
}

// RetryFor is like Retry but with a custom timeout.
func (tc *TestCase) RetryFor(tb testing.TB, d time.Duration, f func() string) string {
	if tc.Rewrite {
		// For rewrite mode, we have nothing to compare the output to. Just sleep a
		// reasonable amount, under the assumption that --rewrite won't be used
		// under stress or a loaded system.
		time.Sleep(d / 10)
		return f()
	}
	runtime.Gosched()
	// We are going to evaluate f until it produces the correct answer numStable
	// times in a row.
	const numAttempts = 100
	const numStable = 3
	// numOk is the number of consecutive calls of f() that have returned the
	// correct answer.
	numOk := 0
	expected := strings.TrimSpace(tc.Expected)
	for i := 0; ; i++ {
		s := f()
		if strings.TrimSpace(s) == expected {
			numOk++
		} else {
			numOk = 0
		}
		if numOk == numStable || i == numAttempts {
			if i >= numStable {
				tc.Logf(tb, "retried for %s (%d times)", time.Duration(i-numStable+1)*d/numAttempts, i-numStable+1)
			}
			return s
		}
		time.Sleep(d/numAttempts + 1)
	}
}

// Arg contains information about an argument on the directive line. An
// argument is specified in one of the following forms:
//   - argument
//   - argument=value
//   - argument=[values, ...]
type Arg struct {
	Name  string
	Value []byte
}

// IsZero reports whether the argument is a zero value (empty Key).
// A zero Arg is returned by Arg and NamedArg when the requested argument does not exist.
func (arg Arg) IsZero() bool {
	return arg.Name == ""
}

func (arg Arg) String() string {
	if arg.Value == nil {
		return arg.Name
	}
	return arg.Name + "=" + string(arg.Value)
}

// Scan unmarshals the argument value into dest via YAML.
// If the argument is zero (key not found), dest is left unchanged and nil is returned.
func (arg Arg) Scan(dest any) error {
	if arg.IsZero() {
		return nil
	}
	value := arg.Value
	if arg.Value == nil {
		value = []byte(arg.Name)
	}
	return yaml.Unmarshal(value, dest)
}

// Logf is a wrapper for tb.Logf which adds file position information, so
// that it's easy to locate the source of the log.
func (tc TestCase) Logf(tb testing.TB, format string, args ...any) {
	tb.Helper()
	tb.Logf("%s: %s", tc.Pos, fmt.Sprintf(format, args...))
}

// Fatalf wraps a fatal testing error with test file position information, so
// that it's easy to locate the source of the error.
//
// Note: tb.Fatalf can also be used directly; the test file position information
// will be logged.
func (tc TestCase) Fatalf(tb testing.TB, format string, args ...any) {
	tb.Helper()
	tb.Fatalf("%s: %s", tc.Pos, fmt.Sprintf(format, args...))
}

// hasBlankLine returns true iff `s` contains at least one line that's
// empty or contains only whitespace.
func hasBlankLine(s string) bool {
	return blankLineRe.MatchString(s)
}

// blankLineRe matches lines that contain only whitespaces (or
// entirely empty/blank lines).  We use the "m" flag for "multiline"
// mode so that "^" can match the beginning of individual lines inside
// the input, not just the beginning of the input.  In multiline mode,
// "$" also matches the end of lines. However, note how the regexp
// uses "\n" to match the end of lines instead of "$". This is
// because of an oddity in the Go regexp engine: at the very end of
// the input, *after the final \n in the input*, Go estimates there is
// still one more line containing no characters but that matches the
// "^.*$" regexp. The result of this oddity is that an input text like
// "foo\n" will match as "foo\n" (no match) + "" (yes match). We don't
// want that final match to be included, so we force the end-of-line
// match using "\n" specifically.
var blankLineRe = regexp.MustCompile(`(?m)^[\t ]*\n`)

func indentLines(str string) string {
	var b strings.Builder
	if str == "" {
		return ""
	}
	for i, l := range strings.Split(str, "\n") {
		if i > 0 {
			b.WriteString("\n")
		}
		if l != "" {
			b.WriteString("  ")
			b.WriteString(l)
		}
	}
	return b.String()
}
