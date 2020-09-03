package registry

import (
	"encoding/json"
	"fmt"
	"strings"
)

type registryError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

type dockerIOError struct {
	Details string `json:"details,omitempty"`
}

type registryErrors struct {
	Errors []registryError `json:"errors"`
}

func errorResponseToString(statusCode int, response []byte) string {
	if len(response) == 0 {
		return fmt.Sprintf("unexpected status code %d", statusCode)
	}

	e := &registryErrors{}
	_ = json.Unmarshal(response, e)

	messages := make([]string, 0)
	for _, err := range e.Errors {
		if err.Message != "" && err.Detail != "" {
			messages = append(messages, fmt.Sprintf("%s: %s", err.Message, err.Detail))
		} else if err.Message != "" {
			messages = append(messages, err.Message)
		} else if err.Detail != "" {
			messages = append(messages, err.Detail)
		} else {
			messages = append(messages, err.Code)
		}
	}

	ee := &dockerIOError{}
	_ = json.Unmarshal(response, ee)
	if ee.Details != "" {
		messages = append(messages, ee.Details)
	}

	errResponse := strings.Join(messages, "\n")
	if errResponse == "" {
		return string(response)
	}

	return errResponse
}
