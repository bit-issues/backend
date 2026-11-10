package tasks

import "github.com/bit-issues/backend/internal/history"

// DiffTaskForTest exposes diffTask to the external test package.
func DiffTaskForTest(current *Task, update TaskUpdate) []history.FieldChange {
	return diffTask(current, update)
}

// NewTaskImportForTest exposes newTaskImport to the external test package.
var NewTaskImportForTest = newTaskImport
