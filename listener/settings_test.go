package listener

import (
	"testing"
)

func TestReplayAddressWithoutLimit(t *testing.T) {
	settings := &ListenerSettings{
		ReplayAddress: "replay:1",
	}

	settings.Parse()

	if settings.ReplayAddress != "replay:1" {
		t.Error("Address not match")
	}

	if settings.ReplayLimit != 0 {
		t.Error("Replay limit should be 0")
	}
}

func TestReplayAddressWithLimit(t *testing.T) {
	settings := &ListenerSettings{
		ReplayAddress: "replay:1|10",
	}

	settings.Parse()

	if settings.ReplayAddress != "replay:1" {
		t.Error("Address not match")
	}

	if settings.ReplayLimit != 10 {
		t.Error("Replay limit should be 10")
	}
}
