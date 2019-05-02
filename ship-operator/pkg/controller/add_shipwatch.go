package controller

import (
	"github.com/replicatedhq/ship-operator/pkg/controller/shipwatch"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, shipwatch.Add)
}
