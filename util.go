package tachyon

import (
	"io/ioutil"
	"launchpad.net/goyaml"
)

func yamlFile(path string, v interface{}) error {
	data, err := ioutil.ReadFile(path)

	if err != nil {
		return err
	}

	return goyaml.Unmarshal(data, v)
}
