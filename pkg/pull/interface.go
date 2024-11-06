package pull

var _p Puller

func init() {
	_p = &puller{}
}

func Mock(p Puller) {
	_p = p
}

type Puller interface {
	Pull(upstreamURI string, pullOptions PullOptions) (string, error)
}

// Convenience functions

func Pull(upstreamURI string, pullOptions PullOptions) (string, error) {
	return _p.Pull(upstreamURI, pullOptions)
}
