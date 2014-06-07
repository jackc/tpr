package main

type Mailer interface {
	SendPasswordResetMail(to, token string) error
}
