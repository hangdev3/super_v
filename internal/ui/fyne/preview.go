package fyneui

import "strings"

func previewText(text string, max int) string {
	compact := strings.Join(strings.Fields(text), " ")
	if max <= 0 {
		return ""
	}
	if len([]rune(compact)) <= max {
		return compact
	}
	if max <= 3 {
		return string([]rune("...")[:max])
	}

	words := strings.Fields(compact)
	prefix := ""
	for _, word := range words {
		candidate := word
		if prefix != "" {
			candidate = prefix + " " + word
		}
		if len([]rune(candidate))+3 > max {
			break
		}
		prefix = candidate
	}
	if prefix != "" {
		return prefix + "..."
	}

	runes := []rune(compact)
	return string(runes[:max-3]) + "..."
}
