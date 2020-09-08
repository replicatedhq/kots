package main

import (
	"math/rand"
	"time"

	"github.com/replicatedhq/kots/kotsadm/cmd/kotsadm/cli"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	cli.InitAndExecute()
}
