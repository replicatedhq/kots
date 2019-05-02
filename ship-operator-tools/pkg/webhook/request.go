package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mholt/archiver"
	"github.com/replicatedhq/ship-operator-tools/pkg/logger"
	"github.com/spf13/viper"
)

type Request struct {
	req    *http.Request
	logger log.Logger
}

type StateLocation struct {
	Namespace string
	Name      string
	Key       string
}

func NewRequest(v *viper.Viper, uri string) (*Request, error) {
	jsonPayload := v.GetString("json-payload")
	dirPayload := v.GetString("directory-payload")
	secretNamespace := v.GetString("secret-namespace")
	secretName := v.GetString("secret-name")
	secretKey := v.GetString("secret-key")

	logger := logger.New(v)
	errs := level.Error(log.With(logger))

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	// write tar part
	w, err := mw.CreateFormFile(tarFormName, "output.tar.gz")
	if err != nil {
		errs.Log("event", "create.multipart.tar", "error", err)
		return nil, err
	}
	if err := archiver.TarGz.Write(w, []string{dirPayload}); err != nil {
		errs.Log("event", "create.tar.archive", "directory", dirPayload, "error", err)
		return nil, err
	}

	// write json part
	w, err = mw.CreateFormFile(jsonFormName, "payload.json")
	if err != nil {
		errs.Log("event", "create.multipart.json", "error", err)
		return nil, err
	}
	if _, err := w.Write([]byte(jsonPayload)); err != nil {
		errs.Log("event", "write.json.multipart", "error", err)
		return nil, err
	}

	// Write the secret (state) part
	stateLocation := StateLocation{
		Namespace: secretNamespace,
		Name:      secretName,
		Key:       secretKey,
	}
	b, err := json.Marshal(stateLocation)
	if err != nil {
		errs.Log("event", "json.marshal", "error", err)
		return nil, err
	}
	w, err = mw.CreateFormFile(stateFormName, "statelocation.json")
	if err != nil {
		errs.Log("event", "create.multipart.statelocation", "error", err)
		return nil, err
	}
	if _, err := w.Write(b); err != nil {
		errs.Log("event", "write.statelocation.multipart", "error", err)
		return nil, err
	}

	if err := mw.Close(); err != nil {
		errs.Log("event", "close.multipartwriter", "error", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, &buf)
	if err != nil {
		errs.Log("event", "http.NewRequest", "error", err)
		return nil, err
	}

	req.Header.Set("Content-Type", mw.FormDataContentType())

	return &Request{
		req:    req,
		logger: logger,
	}, nil
}

func (r Request) Do() error {
	debug := level.Debug(log.With(r.logger, "method", "Do"))

	client := http.Client{}
	resp, err := client.Do(r.req)
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		debug.Log("event", "read.body", "error", err) // continue
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%d: %s", resp.StatusCode, body)
	}

	return nil
}
