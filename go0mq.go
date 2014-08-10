package goshare

import (
	"fmt"
	"runtime"
	"strings"

	zmq "github.com/alecthomas/gozmq"

	golzmq "github.com/abhishekkr/gol/golzmq"
)

/* handling Read/Push/Delete tasks diversion based on task-type */
func goShareZmqRep(socket *zmq.Socket) {
	var err_response string
	for {
		msg, _ := socket.Recv(0)
		message_array := strings.Fields(string(msg))
		response_bytes, axn_status := DBTasks(message_array)

		if axn_status {
			socket.Send([]byte(response_bytes), 0)
		} else {
			err_response = fmt.Sprintf("Error for request sent: %s", msg)
			socket.Send([]byte(err_response), 0)
		}
	}
}

/* start a Daemon communicating over 2 ports over ZMQ Rep/Req */
func GoShareZMQ(ip string, reply_ports []int) {
	fmt.Printf("starting ZeroMQ REP/REQ at %v\n", reply_ports)
	runtime.GOMAXPROCS(runtime.NumCPU())

	socket := golzmq.ZmqReplySocket(ip, reply_ports)
	goShareZmqRep(socket)
}
