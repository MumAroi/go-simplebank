package mail

import (
	"testing"

	"github.com/MumAroi/go-simplebank/util"
	"github.com/stretchr/testify/require"
)

func TestSendEmailWithGmail(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	config, err := util.LoadConfig("..")
	require.NoError(t, err)

	sender := NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)

	subject := "A test email"
	content := `
	<h1>Hello</h1>
	<p>You have successfully sent an email using Go. <a href="https://go.dev">Learn more</a></p>
	`
	to := []string{"for.len.games@gmail.com"}
	attachFiles := []string{"../learn.md"}

	err = sender.SendEmail(
		subject,
		content,
		to,
		nil,
		nil,
		attachFiles,
	)
	require.NoError(t, err)

}
