package fakeyagnats

import (
	"github.com/cloudfoundry/yagnats"
)

type FakeYagnats struct {
	Subscriptions        map[string][]yagnats.Subscription
	PublishedMessages    map[string][]yagnats.Message
	Unsubscriptions      []int
	UnsubscribedSubjects []string

	ConnectedConnectionProvider yagnats.ConnectionProvider

	ConnectError     error
	PublishError     error
	SubscribeError   error
	UnsubscribeError error

	PingResponse bool

	counter int
}

func New() *FakeYagnats {
	fake := &FakeYagnats{}
	fake.Reset()
	return fake
}

func (f *FakeYagnats) Reset() {
	f.PublishedMessages = map[string][]yagnats.Message{}
	f.Subscriptions = map[string][]yagnats.Subscription{}
	f.Unsubscriptions = []int{}
	f.UnsubscribedSubjects = []string{}

	f.ConnectedConnectionProvider = nil

	f.ConnectError = nil
	f.PublishError = nil
	f.SubscribeError = nil
	f.UnsubscribeError = nil

	f.PingResponse = true

	f.counter = 0
}

func (f *FakeYagnats) Ping() bool {
	return f.PingResponse
}

func (f *FakeYagnats) Connect(connectionProvider yagnats.ConnectionProvider) error {
	f.ConnectedConnectionProvider = connectionProvider
	return f.ConnectError
}

func (f *FakeYagnats) Disconnect() {
	f.ConnectedConnectionProvider = nil
	return
}

func (f *FakeYagnats) Publish(subject, payload string) error {
	return f.PublishWithReplyTo(subject, payload, "")
}

func (f *FakeYagnats) PublishWithReplyTo(subject, payload, reply string) error {
	message := yagnats.Message{
		Subject: subject,
		Payload: payload,
		ReplyTo: reply,
	}

	f.PublishedMessages[subject] = append(f.PublishedMessages[subject], message)

	return f.PublishError
}

func (f *FakeYagnats) Subscribe(subject string, callback yagnats.Callback) (int, error) {
	return f.SubscribeWithQueue(subject, "", callback)
}

func (f *FakeYagnats) SubscribeWithQueue(subject, queue string, callback yagnats.Callback) (int, error) {
	f.counter++
	subscription := yagnats.Subscription{
		Subject:  subject,
		Queue:    queue,
		ID:       f.counter,
		Callback: callback,
	}

	f.Subscriptions[subject] = append(f.Subscriptions[subject], subscription)

	return subscription.ID, f.SubscribeError
}

func (f *FakeYagnats) Unsubscribe(subscription int) error {
	f.Unsubscriptions = append(f.Unsubscriptions, subscription)
	return f.UnsubscribeError
}

func (f *FakeYagnats) UnsubscribeAll(subject string) {
	f.UnsubscribedSubjects = append(f.UnsubscribedSubjects, subject)
}
