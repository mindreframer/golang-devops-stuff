package command

import (
	"fmt"
)

func NewCommandFromObj(in interface{}) (interface{}, error) {
	obj, ok := in.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Command: expected dictionary, got %T", in)
	}
	_, exists := obj["code"]
	if exists {
		return NewReplyFromDict(obj)
	} else {
		return NewForwardFromDict(obj)
	}
}
