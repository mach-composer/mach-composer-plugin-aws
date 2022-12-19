package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripProtocol(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"http://example.com", "example.com"},
		{"https://example.com", "example.com"},
		{"example.com", "example.com"},
		{"http://", ""},
		{"https://", ""},
		{"", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			output := StripProtocol(tc.input)
			assert.Equal(t, tc.expected, output)
		})
	}
}
