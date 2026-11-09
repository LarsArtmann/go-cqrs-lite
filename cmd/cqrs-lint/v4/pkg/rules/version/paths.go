package version

import "strings"

// isInThirdParty returns true if the file path is inside a third_party/ directory.
// Handles both absolute ("/project/third_party/pkg/file.go") and relative
// ("third_party/pkg/file.go") paths.
func isInThirdParty(path string) bool {
	return strings.Contains(path, "/third_party/") ||
		strings.HasPrefix(path, "third_party/")
}

// isInVendor returns true if the file path is inside a vendor/ directory.
func isInVendor(path string) bool {
	return strings.Contains(path, "/vendor/") ||
		strings.HasPrefix(path, "vendor/")
}
