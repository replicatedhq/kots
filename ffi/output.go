package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/pkg/errors"
)

type statusMessage struct {
	Status         string `json:"status,omitempty"`
	DisplayMessage string `json:"display_message,omitempty"`
	ExitCode       *int   `json:"exit_code,omitempty"`
}

type StatusClient struct {
	Chan chan interface{}
}

func connectToStatusServer(socket string) (*StatusClient, error) {
	client, err := net.Dial("unix", socket)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to server")
	}
	ch := make(chan interface{}, 0)

	go func() {
		for {
			msg, ok := <-ch
			if !ok {
				client.Close()
				return
			}
			switch msg := msg.(type) {
			case statusMessage, *statusMessage:
				b, err := json.Marshal(msg)
				if err != nil {
					// silent fail
					continue
				}
				buff := bytes.NewBuffer(b)
				if _, err := buff.Write([]byte("\n")); err != nil {
					// silent fail
					continue
				}
				if _, err := buff.WriteTo(client); err != nil {
					fmt.Printf("failed to send status %s: %v\n", msg, err)
					continue
				}
			default:
				// silent fail
			}
		}
	}()

	return &StatusClient{Chan: ch}, nil
}

func (c *StatusClient) getOutputWriter() io.WriteCloser {
	pipeReader, pipeWriter := io.Pipe()
	go func() {
		scanner := bufio.NewScanner(pipeReader)
		for scanner.Scan() {
			c.Chan <- statusMessage{
				Status:         "running",
				DisplayMessage: scanner.Text(),
			}
		}
		pipeReader.CloseWithError(scanner.Err())
	}()
	return pipeWriter
}

func (c *StatusClient) end(err error) {
	exitCode := 0
	message := ""
	if err != nil {
		exitCode = 1
		message = err.Error()
	}

	c.Chan <- statusMessage{
		Status:         "terminated",
		ExitCode:       &exitCode,
		DisplayMessage: message,
	}

	close(c.Chan)
}
