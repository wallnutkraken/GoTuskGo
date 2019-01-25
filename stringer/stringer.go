// Package stringer contains various string utilities not in strings/strconv
package stringer

// SplitMultiple splits on a multi-character list (splitChars)
func SplitMultiple(v string, splitChars string) []string {
	// Create a rune slice from splitChars
	splitRunes := []rune(splitChars)
	result := []string{}
	// Create the current string, this string will be added to every
	// character until we get to a split character
	currentString := []rune{}
	for _, currentRune := range v {
		if !inRuneSlice(currentRune, splitRunes) {
			currentString = append(currentString, currentRune)
			continue
		}
		// Split here, add currentString to the result, then clear it
		// But first, if currentString is empty, omit it
		if len(currentString) > 0 {
			result = append(result, string(currentString))
		}
		currentString = []rune{}
	}
	// Append the final currentString, if it has any length
	if len(currentString) != 0 {
		result = append(result, string(currentString))
	}
	return result
}

func inRuneSlice(v rune, s []rune) bool {
	for _, spl := range s {
		if spl == v {
			return true
		}
	}
	return false
}
