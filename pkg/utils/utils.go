package utils

import (
	"path/filepath"
	"strings"
)

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

func DefaultString(s, defaultValue string) string {
	if s == "" {
		return defaultValue
	}
	return s
}

func DefaultInt(i, defaultValue int) int {
	if i == 0 {
		return defaultValue
	}
	return i
}

// Function to check if one path is a subdirectory of another
func IsSubdirectory(parent, child string) (bool, error) {
    // Clean and convert the parent path to an absolute path
    parentPath, err := filepath.Abs(filepath.Clean(parent))
    if err != nil {
        return false, err
    }
    childPath, err := filepath.Abs(child)
    if err != nil {
        return false, err
    }
    // Ensure proper boundary matching by adding a trailing separator to the parent path
    parentPathWithSep := parentPath + string(filepath.Separator)

    // Use strings.HasPrefix to check if the child path is a subdirectory of the parent path
    isSubdir := strings.HasPrefix(childPath, parentPathWithSep)
    return isSubdir, nil
}