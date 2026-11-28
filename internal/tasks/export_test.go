package tasks

// NewTaskImportForTest exposes newTaskImport to the external test package.
var NewTaskImportForTest = newTaskImport

// SanitizeSearchForTest exposes sanitizeSearch to the external test package.
// This file is compiled only during testing.
func SanitizeSearchForTest(search string) string {
	return sanitizeSearch(search)
}
