package netutils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type HttpError struct {
	StatusCode int
	Body       []byte
}

func (r *HttpError) Error() string {
	return fmt.Sprintf(
		"HttpError(code=%d, body=%s)", r.StatusCode, r.Body)
}

func NewHttpError(statusCode int) *HttpError {
	encodedError, err := json.Marshal(map[string]interface{}{
		"error": http.StatusText(statusCode),
	})

	if err != nil {
		panic(err)
	}

	return &HttpError{
		StatusCode: statusCode,
		Body:       encodedError}
}
