package email

import (
	"bytes"
	"fmt"
	"mime/multipart"
	stdmail "net/mail"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-mail/mail"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
	"github.com/replicatedhq/ship-cluster/worker/pkg/util"
	"github.com/replicatedhq/ship/pkg/state"
)

type EmailRequest struct {
	subject          string
	body             string
	recipient        string
	renderedFileName string

	file multipart.File
}

type Mailer struct {
	logger log.Logger
	config *config.Config
}

func NewMailer(
	logger log.Logger,
	config *config.Config,
) (*Mailer, error) {
	return &Mailer{
		logger: logger,
		config: config,
	}, nil
}

func NewEmailRequest(watch *types.Watch, notification *types.EmailNotification, watchState state.State, file multipart.File, title string) *EmailRequest {
	newVersionString := ""
	if watchState.V1 != nil && watchState.V1.Metadata != nil && watchState.V1.Metadata.Version != "" {
		newVersionString = watchState.V1.Metadata.Version
	}

	if len(title) == 0 {
		title = fmt.Sprintf("Update %s to version %s from Replicated Ship Cloud", watch.Title, newVersionString)
	}

	message := title
	renderedFilename := "rendered.yaml"
	if watchState.V1 != nil && watchState.V1.Metadata != nil {
		if watchState.V1.Metadata.ReleaseNotes != "" {
			message = fmt.Sprintf("Release notes:\n\n%s", watchState.V1.Metadata.ReleaseNotes)
		}
		if watchState.V1.Metadata.Name != "" {
			renderedFilename = fmt.Sprintf("%s.yaml", watchState.V1.Metadata.Name)
		}
	}

	return &EmailRequest{
		subject:          title,
		body:             message,
		recipient:        notification.Address,
		file:             file,
		renderedFileName: renderedFilename,
	}
}

func (m *Mailer) SendEmail(emailRequest *EmailRequest) error {
	_, err := stdmail.ParseAddress(emailRequest.recipient)
	if err != nil {
		level.Error(m.logger).Log("event", "invalid receipient address", "err", err)
		return nil
	}

	fileName, fileContents, err := util.FindRendered(emailRequest.file)
	if err != nil {
		return errors.Wrap(err, "find rendered")
	}

	if len(fileContents) == 0 {
		return errors.Wrap(err, "empty file contents")
	}

	renderedReader := bytes.NewReader([]byte(fileContents))

	msg := mail.NewMessage()
	msg.SetAddressHeader("From", m.config.SMTPFrom, m.config.SMTPFromName)
	msg.SetHeader("To", emailRequest.recipient)
	msg.SetHeader("Subject", emailRequest.subject)
	msg.SetBody("text/html", emailRequest.body)
	msg.AttachReader(fileName, renderedReader)

	d := mail.NewDialer(m.config.SMTPHost, m.config.SMTPPort, m.config.SMTPUser, m.config.SMTPPassword)

	return d.DialAndSend(msg)
}
