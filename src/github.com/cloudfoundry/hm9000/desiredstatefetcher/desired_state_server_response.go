package desiredstatefetcher

import (
	"encoding/json"
	"github.com/cloudfoundry/hm9000/models"
)

type DesiredStateServerResponse struct {
	Results   map[string]models.DesiredAppState `json:"results"`
	BulkToken BulkToken                         `json:"bulk_token"`
}

type BulkToken struct {
	Id int `json:"id"`
}

func NewDesiredStateServerResponse(jsonMessage []byte) (DesiredStateServerResponse, error) {
	response := DesiredStateServerResponse{}
	err := json.Unmarshal(jsonMessage, &response)
	return response, err
}

func (response DesiredStateServerResponse) BulkTokenRepresentation() string {
	bulkTokenRepresentation, _ := json.Marshal(response.BulkToken)
	return string(bulkTokenRepresentation)
}

func (response DesiredStateServerResponse) ToJSON() []byte {
	encoded, _ := json.Marshal(response)
	return encoded
}
