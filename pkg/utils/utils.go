package utils

// Check if a slice contains a string
func ContainsString(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

// RemoveString removes all occurrences of a string from a slice and returns the new slice.
func RemoveString(slice []string, s string) []string {
    var result []string
    for _, item := range slice {
        if item != s {
            result = append(result, item)
        }
    }
    return result
}
