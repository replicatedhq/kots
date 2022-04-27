package main

import (
	"github.com/replicatedhq/kots/cmd/kots/cli"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	cli.InitAndExecute()
}
