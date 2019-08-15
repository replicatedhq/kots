package main

import (
	"github.com/replicatedhq/kots/cmd/kots/cli"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func main() {
	cli.InitAndExecute()
}
