package main

import (
	"fmt"
	"github.com/replicatedhq/kots/cmd/kots/cli"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	fmt.Println("test")
	cli.InitAndExecute()
}
