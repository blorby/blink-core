package implementation

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/blinkops/blink-sdk/plugin"
	log "github.com/sirupsen/logrus"
	gomail "gopkg.in/mail.v2"
	"strconv"
	"strings"
)

func executeCoreMailAction(context *plugin.ActionContext, request *plugin.ExecuteActionRequest) ([]byte, error) {
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
	if fromEmail, ok = fromEmail.(string); !ok {
		return nil, errors.New("mail connection contain an invalid email address")
	}

	password, ok := mailCredentials["password"]
	if !ok {
		return nil, errors.New("mail connection does not contain a password")
	}
	if password, ok = password.(string); !ok {
		return nil, errors.New("mail connection contain an invalid password")
	}

	smtpHost, ok := mailCredentials["smtpHost"]
	if !ok {
		return nil, errors.New("mail connection does not contain smtp host server")
	}
	if smtpHost, ok = smtpHost.(string); !ok {
		return nil, errors.New("mail connection contain an invalid smtp host server")
	}

	smtpPort, ok := mailCredentials["smtpPort"]
	if !ok {
		return nil, errors.New("mail connection does not contain smtp host port")
	}

	smtpPortString := fmt.Sprintf("%v", smtpPort)
	port, err := strconv.Atoi(smtpPortString)

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

	m := gomail.NewMessage()

	// Set E-Mail sender
	m.SetHeader("From", fromEmail.(string))

	// Set E-Mail receivers
	m.SetHeader("To", receivers...)

	// Set E-Mail subject
	m.SetHeader("Subject", subject)

	// Set E-Mail body. You can set plain text or html with text/html
	m.SetBody("text/plain", content)

	// Settings for SMTP server
	d := gomail.NewDialer(smtpHost.(string), port, fromEmail.(string), password.(string))

	// This is only needed when SSL/TLS certificate is not valid on server.
	// In production this should be set to false.
	d.TLSConfig = &tls.Config{InsecureSkipVerify: false, ServerName: smtpHost.(string)}

	// Now send E-Mail
	if err := d.DialAndSend(m); err != nil {
		return nil, err
	}

	return []byte("Message sent successfully"), nil
}
