package utils

import (
	"strings"
)

// LevenshteinDistance calculates the edit distance between two strings
func LevenshteinDistance(s1, s2 string) int {
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))

	r1, r2 := []rune(s1), []rune(s2)
	n, m := len(r1), len(r2)

	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}

	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
		dp[i][0] = i
	}
	for j := 0; j <= m; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			dp[i][j] = minThree(
				dp[i-1][j]+1,      // deletion
				dp[i][j-1]+1,      // insertion
				dp[i-1][j-1]+cost, // substitution
			)
		}
	}
	return dp[n][m]
}

func minThree(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// FindBestFuzzyMatch finds the closest candidate string if within maxDistance
func FindBestFuzzyMatch(target string, candidates []string, maxDistance int) (string, bool) {
	if target == "" || len(candidates) == 0 {
		return "", false
	}
	if maxDistance <= 0 {
		maxDistance = 3 // default edit distance threshold
	}

	bestCandidate := ""
	bestDist := maxDistance + 1

	normTarget := strings.ToLower(strings.TrimSpace(target))

	for _, cand := range candidates {
		normCand := strings.ToLower(strings.TrimSpace(cand))
		if normCand == "" {
			continue
		}

		// Exact match
		if normTarget == normCand {
			return cand, true
		}

		dist := LevenshteinDistance(normTarget, normCand)
		if dist < bestDist {
			bestDist = dist
			bestCandidate = cand
		}
	}

	if bestDist <= maxDistance && bestCandidate != "" {
		return bestCandidate, true
	}
	return "", false
}
