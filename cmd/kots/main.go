package main

import (
	"github.com/replicatedhq/kots/cmd/kots/cli"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

func main() {
	cli.InitAndExecute()
}
