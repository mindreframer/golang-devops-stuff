package command

import (
	"encoding/json"
	"fmt"
)

// Reply command tells proxy to reply with a special
// code and body and do not forward the request.
type Reply struct {
	// HTTP status code
	Code int
	// Body that should be json serializable
	Body interface{}
}

func NewReplyFromDict(in map[string]interface{}) (interface{}, error) {
	codeI, exists := in["code"]
	if !exists {
		return nil, fmt.Errorf("Expected code")
	}
	code := 0
	switch codeC := codeI.(type) {
	case int:
		code = codeC
	case float64:
		if codeC != float64(int(codeC)) {
			return nil, fmt.Errorf("HTTP code should be an integer, got %v", code)
		}
		code = int(codeC)
	default:
		return nil, fmt.Errorf("HTTP code should be an integer, got %v", code)
	}

	if code < 0 {
		return nil, fmt.Errorf("HTTP code should be a positive integer, got %v", code)
	}

	bodyI, exists := in["body"]
	if !exists {
		return nil, fmt.Errorf("Expected body")
	}
	_, err := json.Marshal(bodyI)
	if err != nil {
		return nil, fmt.Errorf("Property 'body' should be json encodeable")
	}
	return &Reply{Code: code, Body: bodyI}, nil
}
