// Package commitref parses commit messages for task references (bare #N,
// keyword refs, and slug#N cross-project refs) and maps action keywords to
// task statuses. It is provided as a service so every consumer shares the
// same configured keyword table.
package commitref
