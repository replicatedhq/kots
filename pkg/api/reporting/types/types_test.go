package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func Test_ReportingInfoRestoredFromInstanceIDSerialization(t *testing.T) {
	t.Run("omitted when empty", func(t *testing.T) {
		req := require.New(t)

		b, err := json.Marshal(ReportingInfo{InstanceID: "instance-1"})
		req.NoError(err)

		decoded := map[string]interface{}{}
		req.NoError(json.Unmarshal(b, &decoded))
		req.NotContains(decoded, "restored_from_instance_id")
	})

	t.Run("present when set", func(t *testing.T) {
		req := require.New(t)

		b, err := json.Marshal(ReportingInfo{InstanceID: "instance-1", RestoredFromInstanceID: "instance-0"})
		req.NoError(err)

		decoded := map[string]interface{}{}
		req.NoError(json.Unmarshal(b, &decoded))
		req.Equal("instance-0", decoded["restored_from_instance_id"])
	})

	t.Run("yaml: omitted when empty", func(t *testing.T) {
		req := require.New(t)

		b, err := yaml.Marshal(ReportingInfo{InstanceID: "instance-1"})
		req.NoError(err)
		req.NotContains(string(b), "restored_from_instance_id")
	})

	t.Run("yaml: present when set", func(t *testing.T) {
		req := require.New(t)

		b, err := yaml.Marshal(ReportingInfo{InstanceID: "instance-1", RestoredFromInstanceID: "instance-0"})
		req.NoError(err)
		req.Contains(string(b), "restored_from_instance_id: instance-0")
	})
}
