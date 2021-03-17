package main

import (
	"fmt"
	"testing"
)

func TestSplitByLines(t *testing.T) {
	tests := []struct {
		input   string
		limit   int
		wantLen int
	}{
		{"abc", 3, 1},
		{"abc", 2, 2},
	}

	for _, test := range tests {
		fmt.Println("\n", test.input)
		splits := splitByLines(test.input, test.limit)
		for i, v := range splits {
			fmt.Println(i, v)
		}
		length := len(splits)
		if test.wantLen != length {
			t.Error("wantLen:", test.wantLen, ", got:", length)
		}
	}
}
