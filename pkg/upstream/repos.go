package upstream

var KnownRepos = map[string]string{
	"stable":  "https://kubernetes-charts.storage.googleapis.com",
	"local":   "http://127.0.0.1:8879",
	"elastic": "https://helm.elastic.co",
	"gomods":  "https://athens.blob.core.windows.net/charts",
	"harbor":  "https://helm.goharbor.io",
}
