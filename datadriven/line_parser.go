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
	"fmt"
	"strings"
	"unicode/utf8"
)

// parseQuotedValue returns the raw YAML scalar bytes for a single-quoted or
// double-quoted argument value and the unparsed remainder of the directive
// line. It preserves the surrounding quotes so Arg.Scan can decode the scalar
// using YAML rules.
func parseQuotedValue(line string) ([]byte, string) {
	quote := line[0]
	pos := 1

	for pos < len(line) {
		switch quote {
		case '"':
			if line[pos] == '\\' {
				// A trailing backslash deliberately runs pos past len(line), so the
				// loop exits and we report an unterminated quoted scalar below.
				pos += 2
				continue
			}
			if line[pos] == '"' {
				return []byte(line[:pos+1]), strings.TrimSpace(line[pos+1:])
			}
		case '\'':
			if line[pos] == '\'' {
				if pos+1 < len(line) && line[pos+1] == '\'' {
					pos += 2
					continue
				}
				return []byte(line[:pos+1]), strings.TrimSpace(line[pos+1:])
			}
		}
		pos++
	}

	panic(parseError{})
}

// ParseLine parses a datadriven directive line and returns the parsed command
// and Args.
//
// An input directive line starts with "> " followed by a command and an
// optional list of arguments:
//
//	> cmd arg=val
//
// Arguments may or may not have values and are specified with one of
// the forms:
//   - <argname>                            # No values.
//   - <argname>=<value>                    # Single value.
//   - <argname>=[<value1>, <value2>, ...]  # Multiple values.
//
// Note that in the last case, we allow the values to contain brackets; the
// parsing will take nesting into account. For example:
//
//	cmd exprs=[a + [b + c], d + f]
//
// is valid and produces the expected values for the argument.
func ParseLine(line string) (name string, args []Arg, err error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", nil, nil
	}
	// Command lines must start with "> ".
	after, ok := strings.CutPrefix(line, "> ")
	if !ok {
		return "", nil, nil
	}
	origLine := line
	line = after

	defer func() {
		if r := recover(); r != nil {
			if r == (parseError{}) {
				column := len(origLine) - len(line) + 1
				name = ""
				args = nil
				err = fmt.Errorf("cannot parse directive at column %d: %s", column, origLine)
				// Note: to debug an unexpected parsing error, this is a good place to
				// add a debug.PrintStack().
			} else {
				panic(r)
			}
		}
	}()

	// until removes the prefix up to one of the given characters from line and
	// returns the prefix.
	until := func(chars string) string {
		idx := strings.IndexAny(line, chars)
		if idx == -1 {
			idx = len(line)
		}
		res := line[:idx]
		line = line[idx:]
		return res
	}

	name = until(" ")
	if name == "" {
		panic(parseError{})
	}
	line = strings.TrimSpace(line)

	for line != "" {
		var arg Arg
		arg.Name = until(" =")
		if arg.Name == "" {
			panic(parseError{})
		}
		if line != "" && line[0] == '=' {
			// Skip the '='.
			line = line[1:]

			if line == "" || line[0] == ' ' {
				// Empty value.
				arg.Value = []byte{}
			} else if line[0] == '"' || line[0] == '\'' {
				arg.Value, line = parseQuotedValue(line)
			} else if line[0] != '[' {
				// Single value: capture raw bytes up to the next space.
				val := until(" ")
				arg.Value = []byte(val)
			} else {
				// Bracket value: capture raw bytes including the brackets.
				// Walk forward tracking nesting to find the matching ']'.
				pos := 1
				nestLevel := 1
				for nestLevel > 0 {
					if pos == len(line) {
						// The string ended before we found the final ']'.
						panic(parseError{})
					}
					r, runeSize := utf8.DecodeRuneInString(line[pos:])
					pos += runeSize
					switch r {
					case '[':
						nestLevel++
					case ']':
						nestLevel--
					}
				}
				arg.Value = []byte(line[:pos])
				line = strings.TrimSpace(line[pos:])
			}
		}
		args = append(args, arg)
		line = strings.TrimSpace(line)
	}
	return name, args, nil
}

type parseError struct{}
