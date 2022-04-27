package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/replicatedhq/kots/cmd/kotsadm/cli"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	fmt.Println("test")
	cli.InitAndExecute()
}
