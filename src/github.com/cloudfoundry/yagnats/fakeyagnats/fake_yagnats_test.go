package fakeyagnats

import (
	"testing"

	"github.com/cloudfoundry/yagnats"
)

func FunctionTakingNATSClient(yagnats.NATSClient) {

}

func TestCanPassFakeYagnatsAsNATSClient(t *testing.T) {
	FunctionTakingNATSClient(New())
}
