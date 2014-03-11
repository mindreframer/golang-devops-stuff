GoSSHa: Go SSH agent
====================

Ssh client that supports command execution and file upload on multiple servers (designed to handle thousands of parallel SSH connections). GoSSHa supports SSH authentication using private keys (encrypted keys are supported using external call to *ssh-keygen*) and ssh-agent, implemented using go.crypto/ssh.

Installation
============

1. Install go (programming language) at http://golang.org/
2. Install GoSSHa: `$ go get github.com/YuriyNasretdinov/GoSSHa`

Usage
=====

GoSSHa is not designed to be used directly by end users, but rather serve as a lightweight proxy between your application (GUI or CLI) and thousands of SSH connections to remote servers.

## Basic protocol

You send commands and receive response by writing and reading JSON lines, for example:

```
$ GoSSHa
{"InitializeComplete":true}
{"Action":"ssh","Cmd":"uptime","Hosts":["localhost"]}   # your input
{"ConnectedHost":"localhost"}
{"Hostname":"localhost","Stdout":" 1:07  up 1 day,  1:32, 2 users, load averages: 0.90 0.99 1.08\n","Stderr":"","Success":true,"ErrMsg":""}
{"TotalTime":0.082024023,"TimedOutHosts":{}}
```

GoSSHa continiously reads stdin and writes response to stdout. The protocol can be split into 2 major phases: initialization and execute loop.

**Note:** When stdin is closed (EOF), then the program exits even if pending operations are not completed.

## Initialization

To be able to run commands GoSSHa examines `~/.ssh/id_rsa` and `~/.ssh/id_dsa` if present and asks for their passwords if they are encrypted. If ssh-agent auth socket is present (identified by presence of `SSH_AUTH_SOCK` environment variable) then it is used as a primary authentication method with fallback to private keys. Password or keyboard-interactive authentication methods are not currently supported, but there are no technical difficulties for adding them.

During initialization, GoSSHa will ask for password for all encrypted private keys it finds, printing message in the following format:

```
{"PasswordFor":"<path-to-private-key>"}
```

You can respond with empty object (`{}`) or provide the passphrase:

```
{"Password":"<passphrase>"}
```

In case of any non-critical errors (e.g. you did not provide a passphrase or the passphase is invalid) you will receive message in the following format:

```
{"IsCritical":false,"ErrorMsg":"<error-message>"}
```

If critical error occurs then all pending operations will be aborted and you will be presented with the same response but "IsCritical":true, for example:

```
{"IsCritical":true,"ErrorMsg":"Cannot parse JSON: unexpected end of JSON input"}
```

When GoSSHa finishes initialization and is ready to accept commands, the following line will be printed:

```
{"InitializeComplete":true}
```

## Commands execution

In order to execute a certain `<command>` on remote servers (e.g. `<server1>` and `<server2>`):

```
{"Action":"ssh","Cmd":"<command>","Hosts":["<server1>","<server2>"]}
```

You can also set `"Timeout": <timeout>` in milliseconds (default is 30000 ms)

While connections to hosts are estabilished and command results are ready you will receive one of the following messages:

1. Error messages: `{"IsCritical":false,"ErrorMsg":"<error-message>"}`
2. Connection progress: `{"ConnectedHost":"<hostname>"}`
3. Command result:

```
{"Hostname":"<hostname>","Stdout":"<command-stdout>","Stderr":"<command-stderr>","Success":true|false,"ErrMsg":"<error message>"}
```

After all commands have done executing or when timeout comes you will receive the following response:

```
{"TotalTime":<total-request-time>,"TimedOutHosts":{"<server1>":true,...,"<serverN>": true}}
```

For your convenience all hosts that timed out are listed in "TimedOutHosts" property, although you could deduce these hosts by subtracting the sets of hostnames that were present in request and the ones present in response.

**Note:** If you send requests to hosts that previously timed out then GoSSHa may not send `{"ConnectedHost":"<hostname>"}` for it and only send the command result.

## File upload

You can also upload file using the following command:

```
{"Action":"scp","Source":"<source-file-path>","Target":"<target-file-path>","Hosts":[...]}
```

You can also set `"Timeout": <timeout>` in milliseconds (default is 30000 ms)

You will receive progress and results in exactly the same format as for command execution.

**Note:** Source file contents are fully read in memory, so you should not try to upload very large files using this command. If you really need to upload huge file to a lot of hosts, try using bittorrent or UFTP, as they provide much higher network effeciency than SSH.

Source code modification
========================

GoSSHa is pretty simple (all it's code is contained in a single file with 500 SLOC) and it should be pretty easy to add new functionality or alter some of it's behaviour. We are always open for pull requests and feature requests as well.
