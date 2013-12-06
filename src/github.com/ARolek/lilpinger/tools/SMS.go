package tools

import (
	"log"

	"github.com/sfreiberg/gotwilio"

	"lilpinger/config"
)

func SendSMS(msg string) {
	if config.Params.Twilio.AccountSid == "" || config.Params.Twilio.AuthToken == "" {
		log.Println("twilio creds not set")
		return
	}

	twilio := gotwilio.NewTwilioClient(config.Params.Twilio.AccountSid, config.Params.Twilio.AuthToken)

	from := config.Params.Twilio.Number

	for i := 0; i < len(config.Params.Notify.Phones); i++ {
		twilio.SendSMS(from, config.Params.Notify.Phones[i], msg, "", "")
	}
}
