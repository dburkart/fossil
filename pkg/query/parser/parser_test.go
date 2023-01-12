/*
 * Copyright (c) 2022-2023, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package parser

import (
	"bufio"
	"fmt"
	"github.com/andreyvit/diff"
	"github.com/dburkart/fossil/pkg/query/ast"
	"github.com/dburkart/fossil/pkg/query/scanner"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTimeWhence(t *testing.T) {
	p := Parser{
		Scanner: scanner.Scanner{
			Input: "~(1996-12-19T16:39:57-08:00)",
		},
	}

	root := p.timeWhence()
	if fmt.Sprint(reflect.TypeOf(root)) != "*ast.TimeWhenceNode" {
		t.Errorf("wanted first child to be *ast.TimeWhenceNode, found %s", reflect.TypeOf(root))
	}

	want, _ := time.Parse(time.RFC3339, "1996-12-19T16:39:57-08:00")

	tm := root.(*ast.TimeWhenceNode).Time()
	if !tm.Equal(want) {
		t.Errorf("wanted time-whence to parse to %s, got %s", want, tm)
	}
}

func TestParse(t *testing.T) {
	testDirectory, err := filepath.Abs("../../../test/parsing/query")
	if err != nil {
		panic(err)
	}

	inputDirectory := path.Join(testDirectory, "input")
	expectationDirectory := path.Join(testDirectory, "expectations")

	tests, err := filepath.Glob(fmt.Sprintf("%s/*.txt", inputDirectory))

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			var expected string
			expectation := path.Join(expectationDirectory, filepath.Base(test))
			expectedBytes, err := os.ReadFile(expectation)
			if err == nil {
				expected = string(expectedBytes)
			}

			file, err := os.Open(test)
			if err != nil {
				t.Errorf("Error opening test: %s", test)
			}

			s := bufio.NewScanner(file)

			shouldPass := false
			s.Scan()
			if strings.ToUpper(s.Text()) == "PASS" {
				shouldPass = true
			}

			actual := ""
			for s.Scan() {
				var d ast.Dumper
				p := Parser{
					scanner.Scanner{
						Input: s.Text(),
					},
				}

				root, err := p.Parse()
				if shouldPass && err != nil {
					t.Error(err)
					continue
				}
				if !shouldPass && err == nil {
					t.Errorf("Expected query to fail: %s", s.Text())
					continue
				}

				if shouldPass {
					ast.Walk(&d, root)
					actual += d.Output
				}
			}

			if os.Getenv("SHOULD_REBASE") != "" {
				err := os.WriteFile(expectation, []byte(actual), 0666)
				if err != nil {
					t.Error(err)
				}
				expected = actual
			}

			if a, e := strings.TrimSpace(actual), strings.TrimSpace(expected); a != e {
				t.Errorf("Expectation not met:\n%s", diff.LineDiff(e, a))
			}
		})
	}

	fmt.Print(tests)
}
