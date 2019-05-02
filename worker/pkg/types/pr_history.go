package types

type PRHistoryItem struct {
	Org            string
	Repo           string
	Branch         string
	RootPath       string
	Sequence       int
	GithubStatus   string
	SourceBranch   string
	NotificationID string
}
