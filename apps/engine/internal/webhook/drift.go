package webhook

import (
	"fmt"
	"strings"
)

// DriftNote compares the payload keys of a dead letter webhook against recent
// healthy ones and describes what changed in plain English. It returns an
// empty string when nothing looks different.
func DriftNote(failedKeys []string, healthy [][]string) string {
	baseline := map[string]bool{}
	samples := 0
	for _, keys := range healthy {
		if len(keys) == 0 {
			continue
		}
		samples++
		for _, k := range keys {
			baseline[k] = true
		}
	}
	if samples == 0 {
		return ""
	}

	failed := map[string]bool{}
	for _, k := range failedKeys {
		failed[k] = true
	}

	var missing, extra []string
	for k := range baseline {
		if !failed[k] {
			missing = append(missing, k)
		}
	}
	for k := range failed {
		if !baseline[k] {
			extra = append(extra, k)
		}
	}
	sortStrings(missing)
	sortStrings(extra)

	usedExtra := map[string]bool{}
	var notes []string

	for _, m := range missing {
		renamed := ""
		for _, e := range extra {
			if usedExtra[e] {
				continue
			}
			if looksRenamed(m, e) {
				renamed = e
				usedExtra[e] = true
				break
			}
		}
		if renamed != "" {
			notes = append(notes, fmt.Sprintf("Field '%s' looks renamed to '%s'", m, renamed))
		} else {
			notes = append(notes, fmt.Sprintf("Field '%s' is missing compared with recent healthy webhooks", m))
		}
	}
	for _, e := range extra {
		if usedExtra[e] {
			continue
		}
		notes = append(notes, fmt.Sprintf("New field '%s' appeared that healthy webhooks did not have", e))
	}

	const maxNotes = 3
	if len(notes) > maxNotes {
		rest := len(notes) - maxNotes
		notes = notes[:maxNotes]
		notes = append(notes, fmt.Sprintf("and %d more field changes", rest))
	}
	out := strings.Join(notes, ". ")
	if out != "" {
		out += "."
	}
	return out
}

// looksRenamed reports whether two field paths could be the same field after a
// rename: same parent and similar leaf names, or one leaf contains the other.
func looksRenamed(a, b string) bool {
	parentA, leafA := splitPath(a)
	parentB, leafB := splitPath(b)
	if parentA != parentB {
		return false
	}
	la, lb := strings.ToLower(leafA), strings.ToLower(leafB)
	if la == lb {
		return true
	}
	if strings.Contains(la, lb) || strings.Contains(lb, la) {
		return true
	}
	return editDistance(la, lb) <= 2 && len(la) > 2 && len(lb) > 2
}

func splitPath(p string) (string, string) {
	if i := strings.LastIndex(p, "."); i >= 0 {
		return p[:i], p[i+1:]
	}
	return "", p
}

// editDistance is the classic Levenshtein distance, fine for short names.
func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			m := prev[j] + 1
			if cur[j-1]+1 < m {
				m = cur[j-1] + 1
			}
			if prev[j-1]+cost < m {
				m = prev[j-1] + cost
			}
			cur[j] = m
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
