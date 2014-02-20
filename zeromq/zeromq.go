package abkzeromq

import (
  "fmt"
  "strings"
  zmq "github.com/alecthomas/gozmq"
)


func ZmqRep(req_port int, rep_port int) *zmq.Socket {
  context, _ := zmq.NewContext()
  socket, _ := context.NewSocket(zmq.REP)
  socket.Bind(fmt.Sprintf("tcp://127.0.0.1:%d", req_port))
  socket.Bind(fmt.Sprintf("tcp://127.0.0.1:%d", rep_port))

  fmt.Printf("ZMQ REQ/REP Daemon at port %d and %d\n", req_port, rep_port)
  return socket
}

func ZmqReq(req_port int, rep_port int, dat ...string) []byte{
  fmt.Printf("ZMQ REQ/REP Client at port %d and %d\n", req_port, rep_port)
  context, _ := zmq.NewContext()
  socket, _ := context.NewSocket(zmq.REQ)
  socket.Connect(fmt.Sprintf("tcp://127.0.0.1:%d", req_port))
  socket.Connect(fmt.Sprintf("tcp://127.0.0.1:%d", rep_port))

  var msg string
  msg = strings.Join(dat, " ")
  socket.Send([]byte(msg), 0)
  response, _ := socket.Recv(0)
  fmt.Printf("msg: %s\nresponse: %s\n\n", msg, response)
  return response
}
