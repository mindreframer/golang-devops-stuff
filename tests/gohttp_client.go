package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	golassert "github.com/abhishekkr/gol/golassert"
	golhashmap "github.com/abhishekkr/gol/golhashmap"
)

var (
	httphost = flag.String("host", "127.0.0.1", "what Host to run at")
	httpport = flag.Int("port", 9999, "what Socket PORT to connect")
)

// return Get URL for task_type, key
func GetURL(host string, port int, key_type, key string) string {
	return fmt.Sprintf("http://%s:%d/get?type=%s&key=%s", host, port, key_type, key)
}

// return Push URL for task_type, key, val
func PutURL(host string, port int, key_type, key, val string) string {
	return fmt.Sprintf("http://%s:%d/put?type=%s&key=%s&val=%s",
		host, port, key_type, key, val)
}

// return Delete URL for task_type, key
func DelURL(host string, port int, key_type, key string) string {
	return fmt.Sprintf("http://%s:%d/del?type=%s&key=%s", host, port, key_type, key)
}

// return Push URL for TSDS type key, val, time-elements
func TSDSPutURL(host string, port int, key, val, year, month, day, hr, min, sec string) string {
	return fmt.Sprintf("http://%s:%d/put?key=%s&val=%s&type=tsds&year=%s&month=%s&day=%s&hour=%s&min=%s&sec=%s",
		host, port, key, val, year, month, day, hr, min, sec)
}

// return Push URL for multi-val-type on task-type and multi-val
func MultiValPutURL(host string, port int, task_type, multi_value string) string {
	return fmt.Sprintf("http://%s:%d/put?dbdata=%s&type=%s", host, port, multi_value, task_type)
}

// return Push URL for TSDS multi-val-type on task-type, val-type and multi-val
func MultiTSDSPutURL(host string, port int, val_type, multi_value, year, month, day, hr, min, sec string) string {
	return fmt.Sprintf("http://%s:%d/put?dbdata=%s&type=tsds-%s&year=%s&month=%s&day=%s&hour=%s&min=%s&sec=%s",
		host, port, multi_value, val_type, year, month, day, hr, min, sec)
}

// append url values for parentNS and return URL
func URLAppendParentNS(url string, parentNS string) string {
	return fmt.Sprintf("%s&parentNS=%s", url, parentNS)
}

// makes HTTP call for given URL and returns response body
func HttpGet(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		return "Error: " + url + " failed for HTTP GET"
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	//fmt.Printf("Url: %s; with\n result:\n%s\n", url, string(body))
	return string(body)
}

// for default key-type
func TestDefaultKeyType() {
	golassert.AssertEqual(HttpGet(PutURL(*httphost, *httpport, "default", "myname", "anon")), "Success")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "default", "myname")), "myname,anon")
	golassert.AssertEqual(HttpGet(PutURL(*httphost, *httpport, "default", "myname", "anonymous")), "Success")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "default", "myname")), "myname,anonymous")
	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "default", "myname")), "Success")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "default", "myname")), "FATAL Error: (DBTasks) map[\"type\":[\"default\"] \"key\":[\"myname\"]]\n")
}

// for ns key-type
func TestNamespaceKeyType() {
	golassert.AssertEqual(HttpGet(PutURL(*httphost, *httpport, "ns", "myname:last:first", "anon")), "Success")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "ns", "myname:last:first")), "myname:last:first,anon")
	golassert.AssertEqual(HttpGet(PutURL(*httphost, *httpport, "ns", "myname:last", "ymous")), "Success")
	golassert.AssertEqual(HttpGet(PutURL(*httphost, *httpport, "ns", "myname", "anonymous")), "Success")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "ns", "myname:last")), "myname:last,ymous\nmyname:last:first,anon")
	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "ns", "myname")), "Success")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "ns", "myname:last")), "FATAL Error: (DBTasks) map[\"type\":[\"ns\"] \"key\":[\"myname:last\"]]\n")
}

// for tsds key-type
func TestTSDSKeyType() {
	golassert.AssertEqual(HttpGet(TSDSPutURL(*httphost, *httpport, "myname:last", "ymous", "2014", "2", "10", "1", "1", "1")), "Success")
	golassert.AssertEqual(HttpGet(TSDSPutURL(*httphost, *httpport, "myname", "anonymous", "2014", "2", "10", "9", "8", "7")), "Success")
	golassert.AssertEqual(HttpGet(TSDSPutURL(*httphost, *httpport, "myname", "untitled", "2014", "2", "10", "0", "1", "7")), "Success")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "tsds", "myname:last")), "myname:last:2014:February:10:1:1:1,ymous")

	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "tsds", "myname")), "myname:last:2014:February:10:1:1:1,ymous\nmyname:2014:February:10:9:8:7,anonymous\nmyname:2014:February:10:0:1:7,untitled")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "tsds", "myname:2014:February:10")), "myname:2014:February:10:9:8:7,anonymous\nmyname:2014:February:10:0:1:7,untitled")
	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "tsds", "myname")), "Success")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "tsds", "myname")), "FATAL Error: (DBTasks) map[\"type\":[\"tsds\"] \"key\":[\"myname\"]]\n")
}

// for now key-type
func TestNowKeyType() {
	csvmap := golhashmap.GetHashMapEngine("csv")
	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "tsds", "yname")), "Success")
	golassert.AssertEqual(HttpGet(PutURL(*httphost, *httpport, "now", "yname:last:first", "zodiac")), "Success")
	golassert.AssertEqual(len(csvmap.ToHashMap(HttpGet(GetURL(*httphost, *httpport, "tsds", "yname:last:first")))), 1)
	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "tsds", "yname")), "Success")

	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "tsds", "myname")), "Success")
	golassert.AssertEqual(HttpGet(PutURL(*httphost, *httpport, "now", "myname:last:first", "ripper")), "Success")
	golassert.AssertEqual(len(csvmap.ToHashMap(HttpGet(GetURL(*httphost, *httpport, "tsds", "myname:last:first")))), 1)
	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "tsds", "myname")), "Success")
}

// for csv val-type
func TestForCSV() {
	golassert.AssertEqual(HttpGet(MultiTSDSPutURL(*httphost, *httpport, "csv", "yourname:last:first,trudy", "2014", "2", "10", "1", "1", "1")), "Success")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "tsds", "yourname:last:first:2014:February")), "yourname:last:first:2014:February:10:1:1:1,trudy")
	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "tsds", "yourname")), "Success")

	golassert.AssertEqual(HttpGet(MultiValPutURL(*httphost, *httpport, "ns-csv", "yname:frend:first,monica")), "Success")
	golassert.AssertEqual(HttpGet(MultiValPutURL(*httphost, *httpport, "ns-csv", "yname:frend:second,lolita")), "Success")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "tsds", "yname:frend")), "yname:frend:first,monica\nyname:frend:second,lolita")
	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "tsds", "yname")), "Success")

	golassert.AssertEqual(HttpGet(MultiValPutURL(*httphost, *httpport, "ns-csv", "yname:frend:first,monica%0D%0Ayname:frend:second,lolita%0D%0Auname:frend:second,juno%0D%0A")), "Success")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "tsds", "yname:frend")), "yname:frend:first,monica\nyname:frend:second,lolita")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "tsds", "uname:frend")), "uname:frend:second,juno")
	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "tsds", "yname")), "Success")
	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "tsds", "uname")), "Success")
}

// for JSON val-type
func TestForJSON() {
	golassert.AssertEqual(HttpGet(MultiValPutURL(*httphost, *httpport, "ns-json", "{\"power:first\":\"yay\",\"power:second\":\"way\",\"rower:second\":\"kay\"}")), "Success")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "tsds", "power")), "power:first,yay\npower:second,way")
	golassert.AssertEqual(HttpGet(GetURL(*httphost, *httpport, "tsds", "rower")), "rower:second,kay")
	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "tsds", "power")), "Success")
	golassert.AssertEqual(HttpGet(DelURL(*httphost, *httpport, "tsds", "rower")), "Success")
}

// for &parentNS=parent:namespace
func TestWithParentNS() {
	var url string

	url = MultiValPutURL(*httphost, *httpport, "ns-csv-parent", "yname:frend:first,monica")
	golassert.AssertEqual(HttpGet(URLAppendParentNS(url, "animal:people")), "Success")

	url = MultiValPutURL(*httphost, *httpport, "ns-csv-parent", "yname:frend:second,lolita")
	golassert.AssertEqual(HttpGet(URLAppendParentNS(url, "animal:people")), "Success")

	url = URLAppendParentNS(GetURL(*httphost, *httpport, "tsds-csv-parent", "yname:frend"), "animal:people")
	golassert.AssertEqual(HttpGet(url), "animal:people:yname:frend:first,monica\nanimal:people:yname:frend:second,lolita")

	url = URLAppendParentNS(GetURL(*httphost, *httpport, "tsds-csv-parent", "people:yname:frend"), "animal")
	golassert.AssertEqual(HttpGet(url), "animal:people:yname:frend:first,monica\nanimal:people:yname:frend:second,lolita")

	url = GetURL(*httphost, *httpport, "tsds-csv-parent", "animal:people:yname:frend")
	golassert.AssertEqual(HttpGet(url), "animal:people:yname:frend:first,monica\nanimal:people:yname:frend:second,lolita")

	url = URLAppendParentNS(DelURL(*httphost, *httpport, "tsds-csv-parent", "yname"), "animal:people")
	golassert.AssertEqual(HttpGet(url), "Success")
}

func main() {
	flag.Parse()

	TestDefaultKeyType()
	TestNamespaceKeyType()
	TestTSDSKeyType()
	TestNowKeyType()
	TestForCSV()
	TestForJSON()
	TestWithParentNS()
	fmt.Println("passed not panic")
}
