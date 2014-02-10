# Gordon

A simple Warden client. Automatically handles reconnecting and dynamically
opening new connections to support concurrent requests.

## Installation

```bash
cd $GOPATH
go get github.com/vito/gordon
```

## Usage

```go
package main

import (
  "fmt"
  "github.com/vito/gordon"
  "os"
)

func main() {
  client := warden.NewClient(
    &warden.ConnectionInfo{
      SocketPath: "/tmp/warden.sock"
    }
  )

  err := client.Connect()
  if err != nil {
    fmt.Println("Failed to connect to Warden: ", err)
    os.Exit(1)
    return
  }

  createResponse, err := client.Create()
  if err != nil {
    fmt.Println("Failed to create container: ", err)
    os.Exit(1)
    return
  }

  handle := createResponse.GetHandle()
  defer client.Destroy(handle)

  fmt.Printf("Container: %s\n", handle)

  spawnResponse, err := client.Spawn(handle, `
    for i in $(seq 10); do
      echo out $i;
      echo err $i 1>&2;
      sleep 1;
    done
  `)

  if err != nil {
    fmt.Println("Failed to spawn process: ", err)
    os.Exit(1)
    return
  }

  fmt.Println("Spawned!", spawnResponse)

  responses, err := client.Stream(handle, spawnResponse.GetJobId())
  if err != nil {
    fmt.Println("Failed to stream output: ", err)
    os.Exit(1)
    return
  }

  fmt.Println("Streaming output...")

  for {
    res, ok := <-responses
    if !ok {
      break
    }

    if res.ExitStatus == nil {
      fmt.Printf("%s: %s", res.GetName(), res.GetData())
    } else {
      fmt.Printf("exited: %d\n", res.GetExitStatus())
    }
  }
}
```
