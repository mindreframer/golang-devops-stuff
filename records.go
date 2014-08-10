package main

type SrvRecord struct {
  Priority uint16
  Weight   uint16
  Port     uint16
  Target   string
}
