// Package commitref parses commit messages for task references (bare #N,
// keyword refs, and slug#N cross-project refs) and maps action keywords to
// task statuses. Parsing is split into a chain of internal parsers sharing
// per-message dedup state; the package is provided as a service so every
// consumer shares the same configured keyword table.
package commitref
