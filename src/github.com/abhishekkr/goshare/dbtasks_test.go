package goshare

import (
	"testing"

	golassert "github.com/abhishekkr/gol/golassert"
	golhashmap "github.com/abhishekkr/gol/golhashmap"
	goltime "github.com/abhishekkr/gol/goltime"
	abklevigoNS "github.com/abhishekkr/levigoNS"
	abkleveldb "github.com/abhishekkr/levigoNS/leveldb"
)

var (
	test_dbpath = "/tmp/delete-this-goshare"
)

func setupTestData() {
	db = abkleveldb.CreateDB(test_dbpath)
	abkleveldb.PushKeyVal("upstate:2014:January:2:12:1:20", "down", db)
	abklevigoNS.PushNS("upstate:2014:January:2:12:1:20", "down", db)
	abklevigoNS.PushNS("upstate:2014:January:2:12:11:20", "up", db)
}

func TestReadKey(t *testing.T) {
	setupTestData()

	expected_val := "upstate:2014:January:2:12:1:20,down"
	result_val := golhashmap.HashMapToCSV(ReadKey("upstate:2014:January:2:12:1:20"))
	golassert.AssertEqual(expected_val, result_val)

	abkleveldb.CloseAndDeleteDB(test_dbpath, db)
}

func TestPushKeyVal(t *testing.T) {
	setupTestData()

	expected_val := "yeah"
	status := PushKeyVal("oh", expected_val)
	result_val := abkleveldb.GetVal("oh", db)
	if expected_val != result_val {
		t.Error("Fail: Get", result_val, "instead of", expected_val)
	}
	if !status {
		t.Error("Fail: Wrong status returned by PushKeyVal")
	}

	abkleveldb.CloseAndDeleteDB(test_dbpath, db)
}

func TestDelKey(t *testing.T) {
	setupTestData()

	status := DelKey("upstate:2014:January:2:12:1:20")
	expected_val := ""
	result_val := abkleveldb.GetVal("oh", db)
	if expected_val != result_val {
		t.Error("Fail: Get", result_val, "instead of", expected_val)
	}
	if !status {
		t.Error("Fail: Wrong status returned by DelKey")
	}
	status = DelKey("oh")
	if !status {
		t.Error("Fail: Wrong status returned by DelKey")
	}

	abkleveldb.CloseAndDeleteDB(test_dbpath, db)
}

func TestReadKeyNS(t *testing.T) {
	setupTestData()

	expected_val := "upstate:2014:January:2:12:1:20,down\n"
	expected_val += "upstate:2014:January:2:12:11:20,up"
	result_val := golhashmap.HashMapToCSV(ReadKeyNS("upstate:2014:January"))
	golassert.AssertEqual(expected_val, result_val)

	abkleveldb.CloseAndDeleteDB(test_dbpath, db)
}

func TestPushKeyValNS(t *testing.T) {
	setupTestData()

	expected_val := "right"
	status := PushKeyValNS("oh:yeah", expected_val)
	result_val := abkleveldb.GetVal("val::oh:yeah", db)
	if expected_val != result_val {
		t.Error("Fail: Get", result_val, "instead of", expected_val)
	}
	if !status {
		t.Error("Fail: Wrong status returned by PushKeyValNS")
	}

	abkleveldb.CloseAndDeleteDB(test_dbpath, db)
}

func TestDelKeyNS(t *testing.T) {
	setupTestData()

	status := DelKeyNS("upstate:2014")
	expected_val := ""
	result_val := abkleveldb.GetVal("val::upstate:2014:January:2:12:1:20", db)
	if expected_val != result_val {
		t.Error("Fail: Get", result_val, "instead of", expected_val)
	}
	if !status {
		t.Error("Fail: Wrong status returned by DelKeyNS")
	}

	abkleveldb.CloseAndDeleteDB(test_dbpath, db)
}

func TestReadKeyTSDS(t *testing.T) {
	setupTestData()

	expected_val := "upstate:2014:January:2:12:1:20,down\n"
	expected_val += "upstate:2014:January:2:12:11:20,up"
	result_val := golhashmap.HashMapToCSV(ReadKeyTSDS("upstate:2014:January:2"))
	golassert.AssertEqual(expected_val, result_val)

	abkleveldb.CloseAndDeleteDB(test_dbpath, db)
}

func TestPushKeyValTSDS(t *testing.T) {
	setupTestData()
	packet := Packet{}
	packet.HashMap = make(golhashmap.HashMap)
	packet.HashMap["oh"] = "yeah"
	packet.ParentNS = "phrase"
	packet.TimeDot = goltime.Timestamp{
		Year: 2014, Month: 1, Day: 2, Hour: 12, Min: 1, Sec: 20,
	}
	packet.KeyType = "tsds"

	status := PushFromPacket(packet)
	expected_val := "yeah"
	result_val := abkleveldb.GetVal("val::oh:2014:January:2:12:1:20", db)
	golassert.AssertEqual(expected_val, result_val)
	if !status {
		t.Error("Fail: Wrong status returned by PushKeyValTSDS")
	}

	abkleveldb.CloseAndDeleteDB(test_dbpath, db)
}
