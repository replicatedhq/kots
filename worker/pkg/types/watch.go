package types

import (
	"fmt"
	"time"
)

type Watch struct {
	ID            string
	Title         string
	StateJSON     string
	Slug          string
	Notifications []*WatchNotification
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type WatchNotification struct {
	ID          string
	Enabled     bool
	PullRequest *PullRequestNotification
	Webhook     *WebhookNotification
	Email       *EmailNotification
}

type PullRequestNotification struct {
	Org                  string
	Repo                 string
	Branch               string
	RootPath             string
	GithubInstallationID string
}

type WebhookNotification struct {
	URI    string
	Secret string
}

type EmailNotification struct {
	Address string
}

func (w Watch) Namespace() string {
	return fmt.Sprintf("shipwatch-%s", w.ID)
}
