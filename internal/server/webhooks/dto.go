package webhooks

import "github.com/bit-issues/backend/internal/webhooks"

// --- Push event payload (BitBucket webhook) ---

type PushEvent struct {
	Push       PushData   `json:"push"`
	Repository Repository `json:"repository"`
	Actor      Actor      `json:"actor"`
}

type PushData struct {
	Changes []Change `json:"changes"`
}

type Change struct {
	Commits []Commit `json:"commits"`
	New     *Ref     `json:"new,omitempty"`
	Old     *Ref     `json:"old,omitempty"`
}

type Commit struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Author  Actor  `json:"author"`
	Date    string `json:"date"`
}

type Ref struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Target Commit `json:"target"`
}

type Repository struct {
	FullName string    `json:"full_name"`
	Name     string    `json:"name"`
	UUID     string    `json:"uuid"`
	Scm      string    `json:"scm"`
	Links    RepoLinks `json:"links"`
}

type RepoLinks struct {
	Self Link `json:"self"`
}

type Link struct {
	Href string `json:"href"`
}

type Actor struct {
	Nickname    string `json:"nickname"`
	DisplayName string `json:"display_name"`
	UUID        string `json:"uuid"`
}

func toPushCommits(event PushEvent) []webhooks.PushCommit {
	var result []webhooks.PushCommit
	for _, change := range event.Push.Changes {
		for _, c := range change.Commits {
			result = append(result, webhooks.PushCommit{
				Hash:    c.Hash,
				Message: c.Message,
				Author:  c.Author.DisplayName,
			})
		}
	}
	return result
}
