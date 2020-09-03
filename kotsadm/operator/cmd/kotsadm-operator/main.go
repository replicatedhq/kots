package main

import (
	"math/rand"
	"time"

	"github.com/replicatedhq/kots/kotsadm/operator/cmd/kotsadm-operator/cli"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	cli.InitAndExecute()
}
