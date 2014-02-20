package goshare

import (
  "fmt"
  "runtime"
  "strings"
  "strconv"

  abkzeromq "github.com/abhishekkr/goshare/zeromq"
)


func goShareZmqRep(req_port int, rep_port int) {
  socket := abkzeromq.ZmqRep(req_port, rep_port)
  for {
    msg, _ := socket.Recv(0)
    msg_arr := strings.Fields(string(msg))
    _axn, _type, _key := msg_arr[0], msg_arr[1], msg_arr[2]
    return_value := ""

    if _axn == "read" {
      _get_val := GetValTask(_type)
      return_value = _get_val(_key)

    } else if _axn == "push" {

      if _type == "tsds" {
        year, _ := strconv.Atoi(msg_arr[3])
        month, _ := strconv.Atoi(msg_arr[4])
        day, _ := strconv.Atoi(msg_arr[5])
        hour, _ := strconv.Atoi(msg_arr[6])
        min, _ := strconv.Atoi(msg_arr[7])
        sec, _ := strconv.Atoi(msg_arr[8])
        _value := strings.Join(msg_arr[9:], " ")
        if PushKeyValTSDS(_key, _value, year, month, day, hour, min, sec){
          return_value = _value
        }
      } else {
        _push_keyval := PushKeyValTask(_type)
        _value := strings.Join(msg_arr[3:], " ")
        if _push_keyval(_key, _value) {
          return_value = _value
        }
      }

    } else if _axn == "delete" {
      _del_key := DelKeyTask(_type)
      if _del_key(_key) {
        return_value = _key
      }

    } else {
      fmt.Printf("unhandled request sent: %s", msg)

    }
    socket.Send([]byte(return_value), 0)
    fmt.Println("Got: [ ", string(msg), " ]; Sent: [ ", return_value, " ]")
  }
}

func GoShareZMQ(req_port int, rep_port int){
  fmt.Printf("starting ZeroMQ REP/REQ at %d/%d\n", req_port, rep_port)
  runtime.GOMAXPROCS(runtime.NumCPU())

  goShareZmqRep(req_port, rep_port)
}
