package deploy

import (
	"encoding/json"
	"os"

	"github.com/dexidp/dex/server"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	dextypes "github.com/replicatedhq/kots/pkg/identity/types/dex"
)

func DexConnectorsToDexTypeConnectors(conns []kotsv1beta1.DexConnector) ([]dextypes.Connector, error) {
	dexConnectors := []dextypes.Connector{}
	for _, conn := range conns {
		f, ok := server.ConnectorsConfig[conn.Type]
		if !ok {
			return nil, errors.Errorf("unknown connector type %q", conn.Type)
		}

		connConfig := f()
		if len(conn.Config.Raw) != 0 {
			data := []byte(os.ExpandEnv(string(conn.Config.Raw)))
			if err := json.Unmarshal(data, connConfig); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal connector config")
			}
		}

		dexConnectors = append(dexConnectors, dextypes.Connector{
			Type:   conn.Type,
			Name:   conn.Name,
			ID:     conn.ID,
			Config: connConfig,
		})
	}
	return dexConnectors, nil
}
