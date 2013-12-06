package config

import (
	"log"

	"github.com/BurntSushi/toml"
)

var Params struct {
	LagThreshold int
	PingInterval int
	URLsFile     string
	Notify       struct {
		Phones []string
		Emails []string
	}
	SMTP struct {
		Email    string
		Password string
		Server   string
		Port     string
	}
	Twilio struct {
		AccountSid string
		AuthToken  string
		Number     string
	}
}

var tomlFile = "lilpinger.toml"

func init() {
	if _, err := toml.DecodeFile(tomlFile, &Params); err != nil {
		log.Fatal(err)
	}
}
