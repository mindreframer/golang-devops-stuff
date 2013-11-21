package models

type DropletUpdatedMessage struct {
	AppGuid string `json:"droplet"`
}

//Freshness Timestamp

type FreshnessTimestamp struct {
	Timestamp int64 `json:"timestamp"`
}
