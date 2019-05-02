package initworker

import (
	"io/ioutil"
	"testing"

	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/stretchr/testify/require"
)

func Test_buildInjector(t *testing.T) {
	type args struct {
		c *config.Config
	}
	tests := []struct {
		name string
		c    *config.Config
	}{
		{
			name: "basic",
			c:    &config.Config{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			_, err := buildInjector(tt.c, ioutil.Discard)
			req.NoError(err)

			// NOTE: this will fail without a k8s connection
			// err = container.Invoke(func(s *Worker) error {
			// 	// don't do anything with it, just make sure we can get one
			// 	return nil
			// })
			req.NoError(err)
		})
	}
}
