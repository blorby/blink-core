package implementation

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/blinkops/blink-core/implementation/execution"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	gomail "gopkg.in/mail.v2"
	"strconv"
	"strings"
)

func addAttachmentIfNeeded(request *plugin.ExecuteActionRequest, message *gomail.Message) {

	attachmentName, nOk := request.Parameters["Attachment Name"]
	attachmentBody, bOk := request.Parameters["Attachment Body"]

	if !nOk || !bOk {
		return
	}

	bodyReader := strings.NewReader(attachmentBody)
	message.AttachReader(attachmentName, bodyReader)
}

func executeCoreMailAction(_ *execution.PrivateExecutionEnvironment, context *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
	mailCredentials, err := context.GetCredentials("core-mail")
	if err != nil {
		err = fmt.Errorf("mail connection was not provided")
		log.Error(err)
		return nil, err
	}

	fromEmail, ok := mailCredentials["email"]
	if !ok {
		return nil, errors.New("mail connection does not contain an email address")
	}

	password, ok := mailCredentials["password"]
	if !ok {
		return nil, errors.New("mail connection does not contain a password")
	}

	smtpHost, ok := mailCredentials["smtpHost"]
	if !ok {
		return nil, errors.New("mail connection does not contain smtp host server")
	}
	smtpPort, ok := mailCredentials["smtpPort"]
	if !ok {
		return nil, errors.New("mail connection does not contain smtp host port")
	}

	port, err := strconv.Atoi(smtpPort)

	if err != nil {
		err = fmt.Errorf("provided smtp port is invalid, error: %v", err)
		log.Error(err)
		return nil, err
	}

	receiver, ok := request.Parameters[mailToKey]
	if !ok {
		return nil, errors.New("no receiver provided for execution")
	}

	receivers := strings.Split(receiver, "\n")

	subject, ok := request.Parameters[mailSubjectKey]
	if !ok {
		return nil, errors.New("no subject provided for execution")
	}

	content, ok := request.Parameters[mailContentKey]
	if !ok {
		return nil, errors.New("no content provided for execution")
	}

	fromEmailDomain := fromEmail
	if strings.Contains(fromEmailDomain, "@") && strings.Contains(fromEmailDomain, ".") {
		fromEmailDomain = strings.Split(strings.Split(fromEmailDomain, "@")[1], ".")[0]
	}

	m := gomail.NewMessage()

	// Set E-Mail sender
	m.SetHeader("From", fmt.Sprintf("%s <%v>", fromEmailDomain, fromEmail))

	// Set E-Mail receivers
	m.SetHeader("To", receivers...)

	// Set E-Mail subject
	m.SetHeader("Subject", subject)

	// Set E-Mail body. You can set plain text or html with text/html
	m.SetBody("text/html", content)

	addAttachmentIfNeeded(request, m)

	// Settings for SMTP server
	d := gomail.NewDialer(smtpHost, port, fromEmail, password)

	// This is only needed when SSL/TLS certificate is not valid on server.
	// In production this should be set to false.
	d.TLSConfig = &tls.Config{InsecureSkipVerify: false, ServerName: smtpHost}

	// Now send E-Mail
	if err := d.DialAndSend(m); err != nil {
		return nil, err
	}

	return []byte("Message sent successfully"), nil
}
