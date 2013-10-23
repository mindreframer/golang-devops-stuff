package models

import (
	"encoding/base64"
	"fmt"
	"strings"
)

type BasicAuthInfo struct {
	User     string
	Password string
}

func DecodeBasicAuthInfo(encoded string) (BasicAuthInfo, error) {
	parts := strings.Split(encoded, " ")
	if len(parts) != 2 || parts[0] != "Basic" {
		return BasicAuthInfo{}, fmt.Errorf("Invalid Authorization")
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return BasicAuthInfo{}, fmt.Errorf("Invalid Authorization")
	}

	userPassword := strings.Split(string(decoded), ":")
	if len(userPassword) != 2 {
		return BasicAuthInfo{}, fmt.Errorf("Invalid Authorization")
	}

	return BasicAuthInfo{User: userPassword[0], Password: userPassword[1]}, nil
}

func (info BasicAuthInfo) Encode() string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(info.User+":"+info.Password))
}
