package cluster

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/k3s-io/kine/pkg/endpoint"
	"github.com/pkg/errors"
	"github.com/rancher/wrangler/pkg/signals"
)

func setupEtcd(ctx context.Context, dataDir string, slug string) ([]string, context.Context, error) {
	dataFile := filepath.Join(dataDir, "kubernetes", "etcd.sqlite")
	listenerValue := "tcp://0.0.0.0:2379"
	endpointValue := fmt.Sprintf("sqlite://%s", dataFile)

	ctx = signals.SetupSignalHandler(ctx)

	config := endpoint.Config{
		Endpoint: endpointValue,
		Listener: listenerValue,
	}

	_, err := endpoint.Listen(ctx, config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "endpoint listen")
	}

	return []string{"http://localhost:2379"}, ctx, nil
}
