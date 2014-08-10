loggregatorlib/emitter
==================

This is a GO library to emit messages to the loggregator.

Create an emitter with NewLogMessageEmitter with the loggregator trafficcontroller hostname and port, a source name, the loggregator shared secret, and a gosteno logger.

Call Emit on the emitter with the application GUID and message strings.

##### A valid source name is any 3 character string.   Some common component sources are:

 	API (Cloud Controller)
 	RTR (Go Router)
 	UAA
 	DEA
 	APP (Warden container)
 	LGR (Loggregator)

###Sample Workflow

    import "github.com/cloudfoundry/loggregatorlib/emitter"

    func main() {
        appGuid := "a8977cb6-3365-4be1-907e-0c878b3a4c6b" // The GUID(UUID) for the user's application
        emitter, err := emitter.NewLogMessageEmitter("10.10.10.16:38452", "RTR", "shared secret", gosteno.NewLogger("LoggregatorEmitter"))
        emitter.Emit(appGuid, message)
    }

###TODO

* All messages are annotated with a message type of OUT. At this time, we don't support emitting messages with a message type of ERR.

