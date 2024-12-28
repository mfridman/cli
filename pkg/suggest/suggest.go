package suggest

import (
	"sort"
	"strings"
)

// threshold is the minimum similarity score required for a string to be considered similar.
const threshold = 0.5

// FindSimilar returns a list of similar strings to the target string from a list of candidates.
func FindSimilar(target string, candidates []string, maxResults int) []string {
	// Early returns for invalid inputs
	if target == "" || maxResults <= 0 {
		return []string{}
	}

	suggestions := make([]struct {
		name  string
		score float64
	}, 0, len(candidates))

	// Calculate similarity scores
	for _, name := range candidates {
		score := calculateSimilarity(target, name)
		if score > threshold { // Only include reasonably similar commands
			suggestions = append(suggestions, struct {
				name  string
				score float64
			}{name, score})
		}
	}

	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].score == suggestions[j].score {
			return suggestions[i].name < suggestions[j].name
		}
		return suggestions[i].score > suggestions[j].score
	})

	// Get top N suggestions
	result := make([]string, 0, maxResults)
	for i := 0; i < len(suggestions) && i < maxResults; i++ {
		result = append(result, suggestions[i].name)
	}

	return result
}

func calculateSimilarity(a, b string) float64 {
	a = strings.ToLower(a)
	b = strings.ToLower(b)

	// Perfect match
	if a == b {
		return 1.0
	}
	// Prefix match bonus
	if strings.HasPrefix(b, a) {
		return 0.9
	}
	// Calculate Levenshtein distance
	distance := levenshteinDistance(a, b)
	maxLen := float64(max(len(a), len(b)))

	// Convert distance to similarity score (0 to 1)
	similarity := 1.0 - float64(distance)/maxLen

	return similarity
}

func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
	}

	for i := 0; i <= len(a); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(b); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1, // deletion
				min(matrix[i][j-1]+1, // insertion
					matrix[i-1][j-1]+cost)) // substitution
		}
	}

	return matrix[len(a)][len(b)]
}
