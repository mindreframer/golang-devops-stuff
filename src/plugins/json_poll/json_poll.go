package json_poll

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"logger"
	"net"
	"net/http"
	"os"
	"strings"
)

func doRequest(req *http.Request, client *http.Client, log *logger.Logger) interface{} {
	var json_out interface{}
	resp, err := client.Do(req)

	if err != nil {
		log.Log("crit", fmt.Sprintf("Could not contact resource at URL '%s': %s", req.URL.String(), err.Error()))
		return nil
	}

	defer resp.Body.Close()
	out, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Log("crit", fmt.Sprintf("Error while gathering output from URL '%s': %s", req.URL.String(), err.Error()))
		return nil
	}

	err = json.Unmarshal(out, &json_out)

	if err != nil {
		log.Log("crit", fmt.Sprintf("Error while marshalling content: %s: %s", string(out), err.Error()))
		return nil
	}

	return json_out
}

func readURL(url string, log *logger.Logger) interface{} {
	req, err := http.NewRequest("GET", url, nil)
	req.Close = true

	if err != nil {
		log.Log("crit", fmt.Sprintf("URL '%s' is malformed: %s", url, err.Error()))
		return nil
	}

	return doRequest(req, &http.Client{}, log)
}

func readUnixSocket(path string, log *logger.Logger) interface{} {
	fi, err := os.Stat(path)

	if err != nil && os.IsNotExist(err) {
		log.Log("crit", "Could not locate unix socket for json_poll at path: "+path)
		return nil
	}

	if fi.Mode()&os.ModeSocket != os.ModeSocket {
		log.Log("crit", path+" is not a unix socket for json_poll")
		return nil
	}

	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.Dial("unix", path)
			},
		},
	}

	req, _ := http.NewRequest("GET", "http://localhost", nil)
	req.Close = true
	return doRequest(req, client, log)
}

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	url := params.(string)

	if strings.HasPrefix(url, "http") {
		return readURL(url, log)
	} else {
		return readUnixSocket(url, log)
	}
}
