package backend

import (
	"bytes"
	"net/smtp"
	"text/template"

	log "gopkg.in/inconshreveable/log15.v2"
)

var passwordResetMailTmpl = template.Must(template.New("passwordResetMailTemplate").Parse("To: {{.To}}\r\nSubject: The Pithy Reader Password Reset\r\n\r\nClick the following link to reset password: {{.RootURL}}/#resetPassword?token={{.Token}}"))

type SMTPMailer struct {
	ServerAddr string
	Auth       smtp.Auth
	From       string
	RootURL    string
	Logger     log.Logger
}

func (m *SMTPMailer) SendPasswordResetMail(to, token string) error {
	var data = struct {
		RootURL string
		To      string
		Token   string
	}{
		RootURL: m.RootURL,
		To:      to,
		Token:   token,
	}

	buf := &bytes.Buffer{}
	err := passwordResetMailTmpl.Execute(buf, data)
	if err != nil {
		return err
	}

	err = smtp.SendMail(m.ServerAddr, m.Auth, m.From, []string{to}, buf.Bytes())
	if err != nil {
		m.Logger.Error("SendPasswordResetEmail failed", "to", to, "error", err)
		return err
	}

	m.Logger.Info("SendPasswordResetEmail", "to", to)
	return nil
}
