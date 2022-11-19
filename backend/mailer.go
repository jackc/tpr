package backend

type Mailer interface {
	SendPasswordResetMail(to, token string) error
}
