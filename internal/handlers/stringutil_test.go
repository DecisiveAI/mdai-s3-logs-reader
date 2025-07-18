package handlers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpperFirst(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{input: "hello", output: "Hello"},
		{input: "Hello", output: "Hello"},
		{input: "hELLO", output: "HELLO"},
		{input: "", output: ""},
		{input: "x", output: "X"},
		{input: "X", output: "X"},
		{input: " hello", output: " hello"},
		{input: "élan", output: "Élan"},
		{input: "😊yay", output: "😊yay"},
		{input: "1abc", output: "1abc"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("upperFirst(%q)", tt.input), func(t *testing.T) {
			assert.Equal(t, tt.output, upperFirst(tt.input))
		})
	}
}
