package utils

import "strings"

func InSlice(what rune, where []rune) bool {
	for _, r := range where {
		if r == what {
			return true
		}
	}
	return false
}

func CountLeadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}
