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
		"hogepiyo",
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
				{line: "hogepiyo"},
			},
		},
		{
			input: "pi",
			lines: defaultLines,
			expect: []matched{
				{line: "piyo"},
				{line: "pipipihoa"},
				{line: "hogepiyo"},
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
