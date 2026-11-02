package tasks

// DiffTaskForTest exposes diffTask to the external test package.
func DiffTaskForTest(current *Task, update TaskUpdate) []FieldChange {
	return diffTask(current, update)
}
