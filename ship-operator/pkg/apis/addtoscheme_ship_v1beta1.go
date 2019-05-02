package apis

import (
	"math/rand"
	"time"

	"github.com/replicatedhq/ship-operator/pkg/apis/ship/v1beta1"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1beta1.SchemeBuilder.AddToScheme)
}
