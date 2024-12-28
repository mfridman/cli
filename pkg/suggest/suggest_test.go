package suggest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindSimilar(t *testing.T) {
	tests := []struct {
		name       string
		target     string
		candidates []string
		maxResults int
		expected   []string
	}{
		{
			name:       "exact match",
			target:     "hello",
			candidates: []string{"hello", "world", "help"},
			maxResults: 2,
			expected:   []string{"hello", "help"},
		},
		{
			name:       "empty target",
			target:     "",
			candidates: []string{"hello", "world"},
			maxResults: 2,
			expected:   []string{},
		},
		{
			name:       "no matches",
			target:     "xyz",
			candidates: []string{"hello", "world"},
			maxResults: 2,
			expected:   []string{},
		},
		{
			name:       "invalid max results",
			target:     "hello",
			candidates: []string{"hello", "world"},
			maxResults: -1,
			expected:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindSimilar(tt.target, tt.candidates, tt.maxResults)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateSimilarity(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
