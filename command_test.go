package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected float64
	}{
		{
			name:     "perfect match",
			a:        "hello",
			b:        "hello",
			expected: 1.0,
		},
		{
			name:     "perfect match with different case",
			a:        "Hello",
			b:        "hello",
			expected: 1.0,
		},
		{
			name:     "prefix match",
			a:        "hel",
			b:        "hello",
			expected: 0.9,
		},
		{
			name:     "one character difference",
			a:        "hello",
			b:        "hello1",
			expected: 0.9, // prefix match case
		},
		{
			name:     "completely different strings",
			a:        "hello",
			b:        "world",
			expected: 0.2, // Based on Levenshtein distance of 4 with max length 5
		},
		{
			name:     "empty strings",
			a:        "",
			b:        "",
			expected: 1.0,
		},
		{
			name:     "one empty string",
			a:        "hello",
			b:        "",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSimilarity(tt.a, tt.b)
			assert.InDelta(t, tt.expected, result, 0.001, "similarity mismatch for %q and %q", tt.a, tt.b)
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{
			name:     "identical strings",
			a:        "hello",
			b:        "hello",
			expected: 0,
		},
		{
			name:     "one character difference",
			a:        "hello",
			b:        "hallo",
			expected: 1,
		},
		{
			name:     "addition",
			a:        "hello",
			b:        "hello1",
			expected: 1,
		},
		{
			name:     "deletion",
			a:        "hello",
			b:        "hell",
			expected: 1,
		},
		{
			name:     "empty first string",
			a:        "",
			b:        "hello",
			expected: 5,
		},
		{
			name:     "empty second string",
			a:        "hello",
			b:        "",
			expected: 5,
		},
		{
			name:     "both empty strings",
			a:        "",
			b:        "",
			expected: 0,
		},
		{
			name:     "completely different strings",
			a:        "hello",
			b:        "world",
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := levenshteinDistance(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "distance mismatch for %q and %q", tt.a, tt.b)
		})
	}
}

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
