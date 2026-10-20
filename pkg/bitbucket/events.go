package bitbucket

// Event is a Bitbucket Cloud webhook event key, as used in the events array
// of a repository webhook subscription and in delivered event payloads.
type Event string

// Repository webhook event keys. Source: GET /2.0/hook_events/repository.

// Repository events.
const (
	EventRepoPush                 Event = "repo:push"
	EventRepoFork                 Event = "repo:fork"
	EventRepoUpdated              Event = "repo:updated"
	EventRepoCommitCommentCreated Event = "repo:commit_comment_created"
	EventRepoCommitStatusCreated  Event = "repo:commit_status_created"
	EventRepoCommitStatusUpdated  Event = "repo:commit_status_updated"
)

// Pull request events.
const (
	EventPullRequestCreated               Event = "pullrequest:created"
	EventPullRequestPush                  Event = "pullrequest:push"
	EventPullRequestUpdated               Event = "pullrequest:updated"
	EventPullRequestApproved              Event = "pullrequest:approved"
	EventPullRequestUnapproved            Event = "pullrequest:unapproved"
	EventPullRequestChangesRequestCreated Event = "pullrequest:changes_request_created"
	EventPullRequestChangesRequestRemoved Event = "pullrequest:changes_request_removed"
	EventPullRequestFulfilled             Event = "pullrequest:fulfilled"
	EventPullRequestRejected              Event = "pullrequest:rejected"
	EventPullRequestCommentCreated        Event = "pullrequest:comment_created"
	EventPullRequestCommentUpdated        Event = "pullrequest:comment_updated"
	EventPullRequestCommentDeleted        Event = "pullrequest:comment_deleted"
	EventPullRequestCommentResolved       Event = "pullrequest:comment_resolved"
	EventPullRequestCommentReopened       Event = "pullrequest:comment_reopened"
)

// Issue events.
const (
	EventIssueCreated        Event = "issue:created"
	EventIssueUpdated        Event = "issue:updated"
	EventIssueCommentCreated Event = "issue:comment_created"
)

// Pipeline events.
const (
	EventPipelineSpanCreated Event = "pipeline:span_created"
)
