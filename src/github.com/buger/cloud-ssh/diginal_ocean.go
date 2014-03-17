package main

import (
	"encoding/json"
	"github.com/jmoiron/jsonq"
	"log"
	"net/http"
)

func getDigitalOceanInstances(config map[string]string) (instances Instances) {
	instances = make(Instances)

	if _, ok := config["client_id"]; !ok {
		log.Fatal("Missing client_id for ", config["name"], " DigitalOcean cloud")
	}

	if _, ok := config["api_key"]; !ok {
		log.Fatal("Missing api_key for ", config["name"], " DigitalOcean cloud")
	}

	resp, err := http.Get("https://api.digitalocean.com/droplets/?client_id=" + config["client_id"] + "&api_key=" + config["api_key"])

	if err != nil {
		log.Println("DigitalOcean API error:", err)
		return
	}

	defer resp.Body.Close()

	data := map[string]interface{}{}
	dec := json.NewDecoder(resp.Body)
	dec.Decode(&data)
	jq := jsonq.NewQuery(data)

	status, err := jq.String("status")

	if status == "ERROR" {
		err_msg, _ := jq.String("error_message")

		log.Println("DigitalOcean API error: ", err_msg)
		return
	}

	droplets, err := jq.ArrayOfObjects("droplets")

	if err != nil {
		log.Println(err)
		return
	}

	for _, droplet := range droplets {
		instances[droplet["ip_address"].(string)] = []Tag{
			Tag{"Name", droplet["name"].(string)},
		}
	}

	return
}
