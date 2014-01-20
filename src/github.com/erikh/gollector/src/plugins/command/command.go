package command

import (
	"encoding/json"
	"fmt"
	"logger"
	"os/exec"
)

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	array_params := params.([]interface{})
	command := make([]string, len(array_params))

	var json_out interface{}

	for i, param := range array_params {
		command[i] = param.(string)
	}

	log.Log("debug", fmt.Sprintf("Command executing: %v", command))

	cmd := exec.Command(command[0], command[1:]...)

	out, err := cmd.Output()

	if err != nil {
		log.Log("crit", fmt.Sprintf("Error while gathering output for command `%s`: %s", command, err))
		return nil
	}

	err = json.Unmarshal(out, &json_out)

	if err != nil {
		log.Log("crit", fmt.Sprintf("Error while marshalling content: %s", string(out)))
		return nil
	}

	return json_out
}
