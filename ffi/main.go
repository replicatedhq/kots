package main

import "C"

import (
	"github.com/replicatedhq/kots/pkg/pull"
)


//export Pull
func Pull(upstreamURI string) error { 
	pullOptions := pull.PullOptions{}

	if err := pull.Pull(upstreamURI, pullOptions); err != nil {
		return err
	}

	return nil
}

func main() {}
