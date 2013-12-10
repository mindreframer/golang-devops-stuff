package control

import (
	"github.com/mailgun/vulcan/instructions"
	"net/http"
)

type Controller interface {
	GetInstructions(*http.Request) (*instructions.ProxyInstructions, error)
}
