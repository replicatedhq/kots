package pull

var p PullerInterface

func init() {
	SetPuller(&Puller{})
}

func SetPuller(_p PullerInterface) {
	p = _p
}

type PullerInterface interface {
	Pull(upstreamURI string, pullOptions PullOptions) (string, error)
}

// Convenience functions

func Pull(upstreamURI string, pullOptions PullOptions) (string, error) {
	return p.Pull(upstreamURI, pullOptions)
}
