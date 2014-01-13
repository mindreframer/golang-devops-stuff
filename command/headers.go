package command

import (
	"fmt"
	"net/http"
)

func NewHeadersFromObj(in interface{}) (http.Header, error) {
	inHeaders, ok := in.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Headers should be a dictionary")
	}
	headers := make(http.Header)
	for key, inHeader := range inHeaders {
		switch values := inHeader.(type) {
		case string:
			headers.Add(key, values)
		case []interface{}:
			for _, valueI := range values {
				value, ok := valueI.(string)
				if !ok {
					return nil, fmt.Errorf("Header value should be a string got unknown type: %#v in %v", valueI, in)
				}
				headers.Add(key, value)
			}
		default:
			return nil, fmt.Errorf("Unsupported header type: %T in %#v", values, in)
		}
	}
	return headers, nil
}

func NewHeadersListFromObj(in interface{}) ([]string, error) {
	inHeaders, ok := in.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Headers should be an array")
	}
	headers := make([]string, len(inHeaders))
	for i, inHeader := range inHeaders {
		switch values := inHeader.(type) {
		case string:
			headers[i] = values
		default:
			return nil, fmt.Errorf("Unsupported header type: %T in %#v", values, in)
		}
	}
	return headers, nil
}

func AddRemoveHeadersFromDict(in map[string]interface{}) (http.Header, []string, error) {
	addHeadersI, exists := in["add_headers"]
	var addHeaders http.Header
	var err error
	if exists {
		addHeaders, err = NewHeadersFromObj(addHeadersI)
		if err != nil {
			return nil, nil, err
		}
	}

	removeHeadersI, exists := in["remove_headers"]
	var removeHeaders []string
	if exists {
		removeHeaders, err = NewHeadersListFromObj(removeHeadersI)
		if err != nil {
			return nil, nil, err
		}
	}
	return addHeaders, removeHeaders, nil
}
