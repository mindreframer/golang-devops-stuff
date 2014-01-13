package js

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/mailgun/vulcan/client"
	"github.com/mailgun/vulcan/netutils"
	"github.com/robertkrimen/otto"
	"net/url"
)

func newError(o *otto.Otto, inError error) otto.Value {
	obj := errorToJs(inError)
	jsObj, err := o.ToValue(obj)
	if err != nil {
		glog.Errorf("Error: %s", err)
		return otto.NullValue()
	}
	return jsObj
}

func toString(in interface{}) (string, error) {
	str, ok := in.(string)
	if !ok {
		return "", fmt.Errorf("Expected string")
	}
	return str, nil
}

func toStringArray(in interface{}) ([]string, error) {
	switch converted := in.(type) {
	case string:
		return []string{converted}, nil
	case []string:
		return converted, nil
	case []interface{}:
		values := make([]string, len(converted))
		for i, valI := range converted {
			val, err := toString(valI)
			if err != nil {
				return nil, err
			}
			values[i] = val
		}
		return values, nil
	}
	return nil, fmt.Errorf("Unsupported type: %T", in)
}

func toMultiDict(in interface{}) (map[string][]string, error) {
	switch value := in.(type) {
	case map[string][]string:
		return value, nil
	case url.Values:
		return value, nil
	case map[string]interface{}:
		return toMultiDictFromInterface(in)
	}
	return nil, fmt.Errorf("Unsupported type: %T", in)
}

func toMultiDictFromInterface(in interface{}) (map[string][]string, error) {
	value, ok := in.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Expected dictionary, got %T", in)
	}
	query := make(client.MultiDict)
	for key, valuesI := range value {
		switch values := valuesI.(type) {
		case string:
			query.Add(key, values)
		case []string:
			for _, val := range values {
				query.Add(key, val)
			}
		case []interface{}:
			for _, subValueI := range values {
				val, ok := subValueI.(string)
				if !ok {
					return nil, fmt.Errorf("Expected string as query value")
				}
				query.Add(key, val)
			}
		default:
			return nil, fmt.Errorf("Unsupported type: %T", values)
		}
	}
	return query, nil
}

func toBasicAuth(in interface{}) (*netutils.BasicAuth, error) {
	value, ok := in.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Expected dictionary, got %T")
	}
	usernameI, ok := value["username"]
	if !ok {
		return nil, fmt.Errorf("Expected username")
	}
	username, err := toString(usernameI)
	if err != nil {
		return nil, err
	}

	passwordI, ok := value["password"]
	if !ok {
		return nil, fmt.Errorf("Expected password")
	}
	password, err := toString(passwordI)
	if err != nil {
		return nil, err
	}
	return &netutils.BasicAuth{Username: username, Password: password}, nil
}
