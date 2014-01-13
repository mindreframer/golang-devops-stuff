package control

import (
	"github.com/mailgun/vulcan/netutils"
	"net/http"
)

type Controller interface {
	GetInstructions(*http.Request) (interface{}, error)
	ConvertError(*http.Request, error) (*netutils.HttpError, error)
}
