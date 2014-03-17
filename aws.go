package main

import (
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/ec2"
	"log"
)

func getEC2Instances(config map[string]string) (instances Instances) {
	instances = make(Instances)

	if _, ok := config["access_key"]; !ok {
		log.Fatal("Missing access_key for ", config["name"], " AWS cloud")
	}

	if _, ok := config["secret_key"]; !ok {
		log.Fatal("Missing secret_key for ", config["name"], " AWS cloud")
	}

	if _, ok := config["region"]; !ok {
		config["region"] = "us-east-1"
	}

	auth := aws.Auth{config["access_key"], config["secret_key"]}

	e := ec2.New(auth, aws.Regions[config["region"]])
	resp, err := e.Instances(nil, nil)

	if err != nil {
		log.Println(err)
		return
	}

	for _, res := range resp.Reservations {
		for _, inst := range res.Instances {

			if inst.DNSName != "" {
				var tags []Tag

				for _, tag := range inst.Tags {
					tags = append(tags, Tag{tag.Key, tag.Value})
				}

				for _, sg := range inst.SecurityGroups {
					tags = append(tags, Tag{"Security group", sg.Name})
				}

				instances[inst.DNSName] = tags
			}
		}
	}

	return
}
