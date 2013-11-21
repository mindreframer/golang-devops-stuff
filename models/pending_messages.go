package models

import (
	"encoding/json"
	"sort"
	"strconv"
	"time"
)

type PendingStartMessageReason string

const (
	PendingStartMessageReasonInvalid    PendingStartMessageReason = ""
	PendingStartMessageReasonCrashed    PendingStartMessageReason = "CRASHED"
	PendingStartMessageReasonMissing    PendingStartMessageReason = "MISSING"
	PendingStartMessageReasonEvacuating PendingStartMessageReason = "EVACUATING"
)

type PendingStopMessageReason string

const (
	PendingStopMessageReasonInvalid            PendingStopMessageReason = ""
	PendingStopMessageReasonExtra              PendingStopMessageReason = "EXTRA"
	PendingStopMessageReasonDuplicate          PendingStopMessageReason = "DUPLICATE"
	PendingStopMessageReasonEvacuationComplete PendingStopMessageReason = "EVACUATION_COMPLETE"
)

type PendingMessage struct {
	MessageId  string `json:"message_id"`
	SendOn     int64  `json:"send_on"`
	SentOn     int64  `json:"sent_on"`
	KeepAlive  int    `json:"keep_alive"`
	AppGuid    string `json:"droplet"`
	AppVersion string `json:"version"`
}

type PendingStartMessage struct {
	PendingMessage
	IndexToStart     int                       `json:"index"`
	Priority         float64                   `json:"priority"`
	SkipVerification bool                      `json:"skip_verification"` //This only exists to allow the evacuator to specify that a message *must* be sent, regardless of verification status
	StartReason      PendingStartMessageReason `json:"start_reason"`
}

type PendingStopMessage struct {
	PendingMessage
	InstanceGuid string                   `json:"instance"`
	StopReason   PendingStopMessageReason `json:"stop_reason"`
}

func newPendingMessage(now time.Time, delayInSeconds int, keepAliveInSeconds int, appGuid string, appVersion string) PendingMessage {
	return PendingMessage{
		SendOn:     now.Add(time.Duration(delayInSeconds) * time.Second).Unix(),
		SentOn:     0,
		KeepAlive:  keepAliveInSeconds,
		AppGuid:    appGuid,
		AppVersion: appVersion,
		MessageId:  Guid(),
	}
}

func (message PendingMessage) pendingLogDescription() map[string]string {
	return map[string]string{
		"SendOn":     time.Unix(message.SendOn, 0).String(),
		"SentOn":     time.Unix(message.SentOn, 0).String(),
		"KeepAlive":  strconv.Itoa(int(message.KeepAlive)),
		"MessageId":  message.MessageId,
		"AppGuid":    message.AppGuid,
		"AppVersion": message.AppVersion,
	}
}

func (message PendingMessage) pendingEqual(another PendingMessage) bool {
	return message.SendOn == another.SendOn &&
		message.SentOn == another.SentOn &&
		message.KeepAlive == another.KeepAlive &&
		message.AppGuid == another.AppGuid &&
		message.AppVersion == another.AppVersion
}

func (message PendingMessage) HasBeenSent() bool {
	return message.SentOn != 0
}

func (message PendingMessage) IsTimeToSend(currentTime time.Time) bool {
	return !message.HasBeenSent() && message.SendOn <= currentTime.Unix()
}

func (message PendingMessage) IsExpired(currentTime time.Time) bool {
	return message.HasBeenSent() && message.SentOn+int64(message.KeepAlive) <= currentTime.Unix()
}

func NewPendingStartMessage(now time.Time, delayInSeconds int, keepAliveInSeconds int, appGuid string, appVersion string, indexToStart int, priority float64, startReason PendingStartMessageReason) PendingStartMessage {
	return PendingStartMessage{
		PendingMessage: newPendingMessage(now, delayInSeconds, keepAliveInSeconds, appGuid, appVersion),
		IndexToStart:   indexToStart,
		Priority:       priority,
		StartReason:    startReason,
	}
}

func NewPendingStartMessageFromJSON(encoded []byte) (PendingStartMessage, error) {
	message := PendingStartMessage{}
	err := json.Unmarshal(encoded, &message)
	if err != nil {
		return PendingStartMessage{}, err
	}
	return message, nil
}

type sortablePendingStartMessagesByPriority []PendingStartMessage

func (s sortablePendingStartMessagesByPriority) Len() int      { return len(s) }
func (s sortablePendingStartMessagesByPriority) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortablePendingStartMessagesByPriority) Less(i, j int) bool {
	return s[i].Priority < s[j].Priority
}

func SortStartMessagesByPriority(messages map[string]PendingStartMessage) []PendingStartMessage {
	sortedStartMessages := make(sortablePendingStartMessagesByPriority, len(messages))
	i := 0
	for _, message := range messages {
		sortedStartMessages[i] = message
		i++
	}
	sort.Sort(sort.Reverse(sortedStartMessages))
	return sortedStartMessages
}

func (message PendingStartMessage) StoreKey() string {
	return message.AppGuid + "-" + message.AppVersion + "-" + strconv.Itoa(message.IndexToStart)
}

func (message PendingStartMessage) ToJSON() []byte {
	encoded, _ := json.Marshal(message)
	return encoded
}

func (message PendingStartMessage) LogDescription() map[string]string {
	base := message.pendingLogDescription()
	base["IndexToStart"] = strconv.Itoa(message.IndexToStart)
	base["SkipVerification"] = strconv.FormatBool(message.SkipVerification)
	base["StartReason"] = string(message.StartReason)
	return base
}

func (message PendingStartMessage) Equal(another PendingStartMessage) bool {
	return message.pendingEqual(another.PendingMessage) &&
		message.IndexToStart == another.IndexToStart &&
		message.Priority == another.Priority &&
		message.SkipVerification == another.SkipVerification &&
		message.StartReason == another.StartReason
}

func NewPendingStopMessage(now time.Time, delayInSeconds int, keepAliveInSeconds int, appGuid string, appVersion string, instanceGuid string, stopReason PendingStopMessageReason) PendingStopMessage {
	return PendingStopMessage{
		PendingMessage: newPendingMessage(now, delayInSeconds, keepAliveInSeconds, appGuid, appVersion),
		InstanceGuid:   instanceGuid,
		StopReason:     stopReason,
	}
}

func NewPendingStopMessageFromJSON(encoded []byte) (PendingStopMessage, error) {
	message := PendingStopMessage{}
	err := json.Unmarshal(encoded, &message)
	if err != nil {
		return PendingStopMessage{}, err
	}
	return message, nil
}

func (message PendingStopMessage) ToJSON() []byte {
	encoded, _ := json.Marshal(message)
	return encoded
}

func (message PendingStopMessage) StoreKey() string {
	return message.InstanceGuid
}

func (message PendingStopMessage) LogDescription() map[string]string {
	base := message.pendingLogDescription()
	base["InstanceGuid"] = message.InstanceGuid
	base["StopReason"] = string(message.StopReason)
	return base
}

func (message PendingStopMessage) Equal(another PendingStopMessage) bool {
	return message.pendingEqual(another.PendingMessage) &&
		message.InstanceGuid == another.InstanceGuid &&
		message.StopReason == another.StopReason
}
