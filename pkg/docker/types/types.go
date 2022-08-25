package types

const (
	FormatDockerRegistry = "docker"
	FormatDockerArchive  = "docker-archive"
)

type Layer struct {
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}
