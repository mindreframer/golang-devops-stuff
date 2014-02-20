package goshare

import (
  "time"

  abkleveldb "github.com/abhishekkr/levigoNS/leveldb"
  levigoNS "github.com/abhishekkr/levigoNS"
  levigoTSDS "github.com/abhishekkr/levigoTSDS"
)


type GetValFunc func(key string) string
type PushKeyValFunc func(key string, val string) bool
type DelKeyFunc func(key string) bool


func GetVal(key string) string{
  return abkleveldb.GetVal(key, db)
}


func PushKeyVal(key string, val string) bool{
  return abkleveldb.PushKeyVal(key, val, db)
}


func DelKey(key string) bool{
  return abkleveldb.DelKey(key, db)
}


func hmap_to_csv(hmap levigoNS.HashMap) string{
  csv := ""
  for  key, value := range hmap {
    csv += key + "," + value + "\n"
  }
  return csv
}


func GetValNS(key string) string{
  return hmap_to_csv(levigoNS.ReadNSRecursive(key, db))
}


func PushKeyValNS(key string, val string) bool{
  return levigoNS.PushNS(key, val, db)
}


func DelKeyNS(key string) bool{
  levigoNS.DeleteNSRecursive(key, db)
  return true
}


func GetValTSDS(key string) string{
  return hmap_to_csv(levigoTSDS.ReadTSDS(key, db))
}


func PushKeyValTSDS(key string, val string,
                    year int, month int, day int,
                    hour int, min int, sec int) bool{
  key_time := time.Date(year, time.Month(month),
                        day, hour, min, sec, 0, time.UTC)
  levigoTSDS.PushTSDS(key, val, key_time, db)
  return true
}


func PushKeyValNowTSDS(key string, val string) bool{
  levigoTSDS.PushNowTSDS(key, val, db)
  return true
}


func DelKeyTSDS(key string) bool{
  levigoTSDS.DeleteTSDS(key, db)
  return true
}


func GetValTask(task_type string) GetValFunc {
  if task_type == "tsds" {
    return GetValTSDS
  } else if task_type == "ns" {
    return GetValNS
  }
  return GetVal
}


func PushKeyValTask(task_type string) PushKeyValFunc {
  if task_type == "tsds-now" {
    return PushKeyValNowTSDS
  } else if task_type == "ns" {
    return PushKeyValNS
  }
  return PushKeyVal
}


func DelKeyTask(task_type string) DelKeyFunc {
  if task_type == "tsds" {
    return DelKeyTSDS
  } else if task_type == "ns" {
    return DelKeyNS
  }
  return DelKey
}
