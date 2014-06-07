package main

type testPasswordResetMail struct {
	to    string
	token string
}

type testMailer struct {
	sentPasswordResetMails []testPasswordResetMail
}

func (m *testMailer) SendPasswordResetMail(to, token string) error {
	e := testPasswordResetMail{to: to, token: token}
	m.sentPasswordResetMails = append(m.sentPasswordResetMails, e)
	return nil
}
