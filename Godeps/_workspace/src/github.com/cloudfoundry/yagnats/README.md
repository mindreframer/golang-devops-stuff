[![Build Status](https://travis-ci.org/cloudfoundry/yagnats.png)](https://travis-ci.org/cloudfoundry/yagnats)

Yet Another Go NATS Client
==========================

(or: You Ain't Gonna Need Another TIBCO System)

A simple client for NATS written in Go.

Basic usage:

```go
client := yagnats.NewClient()

err := client.Connect(&yagnats.ConnectionInfo{
		Addr:     "127.0.0.1:4222",
		Username: "user",
		Password: "pass",
})
if err != nil {
  panic("Wrong auth or something.")
}

client.Subscribe("some.subject", func(msg *Message) {
  fmt.Printf("Got message: %s\n", msg.Payload)
})

client.Publish("some.subject", []byte("Sup son?"))
```

