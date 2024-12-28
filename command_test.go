package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected []string
	}{
		{
			name:     "simple wrap",
			text:     "hello world",
			width:    5,
			expected: []string{"hello", "world"},
		},
		{
			name:     "no wrap needed",
			text:     "hello",
			width:    10,
			expected: []string{"hello"},
		},
		{
			name:     "multiple wraps",
			text:     "this is a long text that needs wrapping",
			width:    10,
			expected: []string{"this is a", "long text", "that needs", "wrapping"},
		},
		{
			name:     "empty string",
			text:     "",
			width:    10,
			expected: nil,
		},
		{
			name:     "single word longer than width",
			text:     "supercalifragilistic",
			width:    10,
			expected: []string{"supercalifragilistic"},
		},
		{
			name:     "multiple spaces",
			text:     "hello    world",
			width:    20,
			expected: []string{"hello world"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapText(tt.text, tt.width)
			assert.EqualValues(t, tt.expected, result, "wrapped text mismatch for input %q with width %d", tt.text, tt.width)
		})
	}
}
