package models

import "github.com/nu7hatch/gouuid"

func Guid() string {
	u, _ := uuid.NewV4()
	return u.String()
}
