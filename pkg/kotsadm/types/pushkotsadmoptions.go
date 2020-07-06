package types

import "io"

type PushKotsadmImagesOptions struct {
	AirgapArchive  string
	Registry       string
	Username       string
	Password       string
	ProgressWriter io.Writer // using a writer because of skopeo
}
