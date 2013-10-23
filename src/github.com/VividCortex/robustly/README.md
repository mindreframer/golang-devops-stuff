## Robustly

Robustly runs code resiliently, recovering from occasional errors.
It also gives you the ability to probabilistically inject panics into
your application, configuring them at runtime at crash sites of your
choosing. We use it at [VividCortex](https://vividcortex.com/blog/2013/07/30/writing-resilient-programs-with-go-and-robustly-run/)
to ensure that unexpected problems don't disable our agent programs.

![Build Status](https://circleci.com/gh/VividCortex/robustly.png?circle-token=75e143a154914d6ecf50376b0d93b5401739c52e)

### Getting Started

```
go get github.com/VividCortex/robustly
```

Now import the following in your code:

```go
import (
	"github.com/VividCortex/robustly"
)

func main() {
	go robustly.Run(func() { somefunc() })
}

func somefunc() {
	for {
		// do something here that may panic
	}
}
```

### API Documentation

View the GoDoc generated documentation [here](http://godoc.org/github.com/VividCortex/robustly).

### Robustly's Purpose

Robustly is designed to help make Go programs more resilient to errors
you don't discover until they're in the field. It is not a general-purpose
approach and shouldn't be overused, but in specific conditions it can be valuable.

![cat](http://eventingnation.com/eventingnation.com/images/2012/04/cat-helmet.jpg)

Imagine, for example, that you are writing a program designed to process events
at a high rate, such as 50,000 per second. The program is stateful, and its
value comes from observing the event stream for relatively long periods, such
as several minutes, to learn its behavior. Now imagine that you introduce a
subtle bug into the program, which will happen extremely rarely -- once in a
million. Although rare, this bug will cause a panic and crash the program.

Your program will be completely useless for its intended purpose, because
you're likely to hit a once-in-a-million error every 20 seconds.
Handling such errors, especially when the program will take some time and effort
to fix and redeploy, can make the program 99.9999% useful again.

Robustly is targeted towards this type of use case. Its design is inspired by
the `net/http` server's code, where each HTTP request is handled in a goroutine
that can crash without crashing the entire server.

When Robustly handles a crash, it immediately restarts the offending code. It keeps
track of how fast the code crashes, and if it crashes too quickly for too long, it
gives up and crashes the whole program. This way once-in-a-million errors can be
restarted without getting into infinite loops.

### Using Run

To use Run, simply wrap around the function call that represents
the entry point to the code you wish to catch and restart:

```go
robustly.Run(func() { /* your code here */ }, 1, 1, 1)
```

The function takes three options: a crash rate threshold, a crash timeout, and whether
to print stack traces to STDOUT when there's a crash. All three have reasonable defaults.

### Using Crash

Robustly also includes `Crash()`, a way to inject panics into your code at runtime.
To use it, select places where you'd like to cause crashes, and add the following
line of code:

```go
robustly.Crash()
```

Configure crash sites with `CrashSetup()`. Pass it a comma-separated string of crash
sites, which are colon-separated `file:line:probability` specifications. Probability
should range between 0 and 1. If you pass the special spec `"VERBOSE"`, it will enable
printouts of all crash sites that are located in your code.

The idea is to match the crash sites configured in the setup with those actually
present in your code. For example, if you have added a crash site in the code at
line 53 of client.go, and you'd like to crash there, as well as at line 18 of server.go:

    client.go:53:.003,server.go:18:.02

That will cause a crash .003 of the time at client.go line 53, and .02 of the time
at server.go line 18.

If you are using `robustly.Run()` to make your code resilient to errors, it is a very
good idea to deliberately inject errors and make sure they are indeed handled. You can
easily miss a detail such as a potentially crashing function that is called as a goroutine.

## Contributing

Contributions are welcome. Please send a pull request!

## License

This program is (c) VividCortex 2013, and is licensed under the MIT license. Please see the LICENSE file.
