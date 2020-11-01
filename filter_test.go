package fy

import (
	"reflect"
	"testing"
)

func TestFindMatches(t *testing.T) {
	defaultLines := []string{
		"こんにちわ",
		"fuga",
		"piyo",
		"pipipihoa",
		"123",
		"hogefupiyo",
	}

	type testcase struct {
		input  string
		lines  []string
		expect []matched
	}

	tc := []testcase{
		{
			input: "hoge",
			lines: defaultLines,
			expect: []matched{
				{line: "hogefupiyo", index: 0, pos1: 0, pos2: 4},
			},
		},
		{
			input: "pi",
			lines: defaultLines,
			expect: []matched{
				{line: "piyo", index: 0, pos1: 0, pos2: 2},
				{line: "pipipihoa", index: 1, pos1: 4, pos2: 6},
				{line: "hogefupiyo", index: 2, pos1: 6, pos2: 8},
			},
		},
	}

	for _, tt := range tc {
		tt := tt

		ma, err := findMatches(tt.input, tt.lines)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(ma, tt.expect) {
			t.Errorf("expect %v, but got %v", tt.expect, ma)
		}
	}
}
