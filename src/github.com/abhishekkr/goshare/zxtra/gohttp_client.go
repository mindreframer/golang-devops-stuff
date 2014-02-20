package main

import (
  "fmt"
  "flag"
  "io/ioutil"
  "net/http"
)

var (
  httphost   = flag.String("host", "127.0.0.1", "what Host to run at")
  httpport   = flag.Int("port", 9999, "what Socket PORT to connect")
)

/* KeyVal Support */

func GetURL(host string, port int, key string) string{
  return fmt.Sprintf("http://%s:%d/get?key=%s", host, port, key)
}

func PutURL(host string, port int, key string, val string) string{
  return fmt.Sprintf("http://%s:%d/put?key=%s&val=%s",
                     host, port, key, val)
}

func DelURL(host string, port int, key string) string{
  return fmt.Sprintf("http://%s:%d/del?key=%s", host, port, key)
}


/* KeyVal NS Support */

func NSGetURL(host string, port int, key string) string{
  return fmt.Sprintf("http://%s:%d/get?key=%s&type=ns", host, port, key)
}

func NSPutURL(host string, port int, key string, val string) string{
  return fmt.Sprintf("http://%s:%d/put?key=%s&val=%s&type=ns",
                     host, port, key, val)
}

func NSDelURL(host string, port int, key string) string{
  return fmt.Sprintf("http://%s:%d/del?key=%s&type=ns", host, port, key)
}


/* KeyVal TSDS Support */

func TSDSGetURL(host string, port int, key string) string{
  return fmt.Sprintf("http://%s:%d/get?key=%s&type=tsds", host, port, key)
}

func TSDSPutURL(host string, port int, key, val, year, month, day, hr, min, sec string) string{
  return fmt.Sprintf("http://%s:%d/put?key=%s&val=%s&type=tsds&year=%s&month=%s&day=%s&hour=%s&min=%s&sec=%s",
                     host, port, key, val, year, month, day, hr, min, sec)
}

func NowTSDSPutURL(host string, port int, key string, val string) string{
  return fmt.Sprintf("http://%s:%d/put?key=%s&val=%s&type=tsds-now",
                     host, port, key, val)
}

func TSDSDelURL(host string, port int, key string) string{
  return fmt.Sprintf("http://%s:%d/del?key=%s&type=tsds", host, port, key)
}

func HttpGet(url string) string{
  resp, err := http.Get(url)
  if err != nil {
    return "Error: " + url + " failed for HTTP GET"
  }
  defer resp.Body.Close()

  body, _ := ioutil.ReadAll(resp.Body)
  fmt.Printf("Url: %s; with\n result:\n%s\n", url, string(body))
  fmt.Println("///////////////////////////////////////////////")
  return string(body)
}

func main(){
  flag.Parse()

  HttpGet(PutURL(*httphost, *httpport, "myname", "anon"))
  HttpGet(GetURL(*httphost, *httpport, "myname"))
  HttpGet(PutURL(*httphost, *httpport, "myname", "anonymous"))
  HttpGet(GetURL(*httphost, *httpport, "myname"))
  HttpGet(DelURL(*httphost, *httpport, "myname"))
  HttpGet(GetURL(*httphost, *httpport, "myname"))

  HttpGet(NSPutURL(*httphost, *httpport, "myname:last:first", "anon"))
  HttpGet(NSGetURL(*httphost, *httpport, "myname:last:first"))
  HttpGet(NSPutURL(*httphost, *httpport, "myname:last", "ymous"))
  HttpGet(NSPutURL(*httphost, *httpport, "myname", "anonymous"))
  HttpGet(NSGetURL(*httphost, *httpport, "myname:last"))
  HttpGet(NSDelURL(*httphost, *httpport, "myname"))
  HttpGet(NSGetURL(*httphost, *httpport, "myname:last"))

  HttpGet(NowTSDSPutURL(*httphost, *httpport, "myname:last:first", "anon"))
  HttpGet(TSDSGetURL(*httphost, *httpport, "myname:last:first"))
  HttpGet(TSDSPutURL(*httphost, *httpport, "myname:last", "ymous", "2014", "2", "10", "1", "1", "1"))
  HttpGet(TSDSPutURL(*httphost, *httpport, "myname", "anonymous", "2014", "2", "10", "9", "8", "7"))
  HttpGet(TSDSPutURL(*httphost, *httpport, "myname", "untitled", "2014", "2", "10", "0", "1", "7"))
  HttpGet(TSDSGetURL(*httphost, *httpport, "myname:last"))
  HttpGet(TSDSGetURL(*httphost, *httpport, "myname"))
  HttpGet(TSDSGetURL(*httphost, *httpport, "myname:2014:February:10"))
  HttpGet(TSDSDelURL(*httphost, *httpport, "myname"))
  HttpGet(TSDSGetURL(*httphost, *httpport, "myname"))

  HttpGet(NowTSDSPutURL(*httphost, *httpport, "myname:last:first", "anon"))
  HttpGet(TSDSGetURL(*httphost, *httpport, "myname:last:first:2014:February"))
  HttpGet(TSDSDelURL(*httphost, *httpport, "myname"))
}
