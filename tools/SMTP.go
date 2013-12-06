package tools

import (
	"fmt"
	"log"
	"net/smtp"

	"lilpinger/config"
)

func SendMail(msg string) {
	if config.Params.SMTP.Email == "" || config.Params.SMTP.Password == "" || config.Params.SMTP.Server == "" || config.Params.SMTP.Port == "" {
		log.Println("SMTP creds not set")
	}

	auth := smtp.PlainAuth(
		"",
		config.Params.SMTP.Email,
		config.Params.SMTP.Password,
		config.Params.SMTP.Server,
	)

	server := fmt.Sprintf("%v:%v", config.Params.SMTP.Server, config.Params.SMTP.Port)
	err := smtp.SendMail(
		server,
		auth,
		config.Params.SMTP.Email,
		config.Params.Notify.Emails,
		[]byte("Subject: "+msg+"\r\n\r\n"+msg),
	)

	if err != nil {
		fmt.Println("Error sending mail - %v - message: ", err, msg)
	}
}
