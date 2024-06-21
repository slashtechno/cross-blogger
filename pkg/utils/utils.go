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
