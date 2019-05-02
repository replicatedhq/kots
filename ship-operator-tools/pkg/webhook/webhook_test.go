package webhook

import (
	"io/ioutil"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/replicatedhq/ship-operator/pkg/logger"
	"github.com/spf13/viper"
)

func TestRequest(t *testing.T) {
	serverDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{
		dir:    serverDir,
		logger: logger.FromEnv(),
	}
	ts := httptest.NewServer(h)
	defer ts.Close()

	// make an output directory with some content to test the tar part
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile := filepath.Join(tmpDir, "state.json")
	if err := ioutil.WriteFile(tmpFile, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	tmpFile2 := filepath.Join(tmpDir, "pod.yaml")
	if err := ioutil.WriteFile(tmpFile2, []byte("kind: Pod"), 0644); err != nil {
		t.Fatal(err)
	}

	v := viper.New()
	v.Set("json-payload", `{"foo": "bar"}`)
	v.Set("directory-payload", tmpDir)
	v.Set("secret-namespace", "ns")
	v.Set("secret-name", "s")
	v.Set("secret-key", "k")
	req, err := NewRequest(v, ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := req.Do(); err != nil {
		t.Fatal(err)
	}
}
