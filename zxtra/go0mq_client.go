package main

import (
  "fmt"
  "flag"

  "../zeromq"
)

var (
  req_port    = flag.Int("req-port", 9797, "what Socket PORT to run at")
  rep_port    = flag.Int("rep-port", 9898, "what Socket PORT to run at")
)

func main(){
  flag.Parse()
  fmt.Printf("client ZeroMQ REP/REQ... at %d, %d", req_port, rep_port)

  fmt.Println("Checking out levigo based storage...")
  abkzeromq.ZmqReq(*req_port, *rep_port, "push", "default", "myname", "anon")
  abkzeromq.ZmqReq(*req_port, *rep_port, "read", "default", "myname")
  abkzeromq.ZmqReq(*req_port, *rep_port, "push", "default", "myname", "anonymous")
  abkzeromq.ZmqReq(*req_port, *rep_port, "read", "default", "myname")
  abkzeromq.ZmqReq(*req_port, *rep_port, "delete", "default", "myname")
  abkzeromq.ZmqReq(*req_port, *rep_port, "read", "default", "myname")

  fmt.Println("Checking out levigoNS based storage...")
  abkzeromq.ZmqReq(*req_port, *rep_port, "push", "ns", "myname:last:first", "anon")
  abkzeromq.ZmqReq(*req_port, *rep_port, "read", "ns", "myname")
  abkzeromq.ZmqReq(*req_port, *rep_port, "push", "ns", "myname:last", "ymous")
  abkzeromq.ZmqReq(*req_port, *rep_port, "push", "ns", "myname", "anonymous")
  abkzeromq.ZmqReq(*req_port, *rep_port, "read", "ns", "myname")
  abkzeromq.ZmqReq(*req_port, *rep_port, "delete", "ns", "myname")
  abkzeromq.ZmqReq(*req_port, *rep_port, "read", "ns", "myname")

  fmt.Println("Checking out levigoTSDS based storage...")
  abkzeromq.ZmqReq(*req_port, *rep_port, "push", "tsds-now", "myname:last:first", "anon")
  abkzeromq.ZmqReq(*req_port, *rep_port, "push", "tsds", "myname:last:first", "2014", "2", "10", "9", "8", "7", "anon")
  abkzeromq.ZmqReq(*req_port, *rep_port, "read", "tsds", "myname")
  abkzeromq.ZmqReq(*req_port, *rep_port, "push", "tsds-now", "myname:last", "ymous")
  abkzeromq.ZmqReq(*req_port, *rep_port, "push", "tsds", "myname", "2014", "2", "10", "9", "18", "37", "anonymous")
  abkzeromq.ZmqReq(*req_port, *rep_port, "push", "tsds", "myname", "2014", "2", "10", "5", "28", "57", "untitles")
  abkzeromq.ZmqReq(*req_port, *rep_port, "read", "tsds", "myname")
  abkzeromq.ZmqReq(*req_port, *rep_port, "read", "tsds", "myname:2014:February:10")
  abkzeromq.ZmqReq(*req_port, *rep_port, "delete", "tsds", "myname")
  abkzeromq.ZmqReq(*req_port, *rep_port, "read", "tsds", "myname")
}
