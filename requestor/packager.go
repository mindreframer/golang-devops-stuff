package goshare_requestor

import (
	"fmt"

	golhashmap "github.com/abhishekkr/gol/golhashmap"
	gollist "github.com/abhishekkr/gol/gollist"
	goshare "github.com/abhishekkr/goshare"
)

/*
Create request bytes from Packet
[DB-Action] [Task-Type] ([Time-Dot]) ([Parent-NameSpace]) {[Key ([Val...])] OR [DB-Data...]}
*/
func RequestPacketBytes(packet *goshare.Packet) []byte {
	var request string
	task_type := taskTypeFromPacket(packet)
	dbdata := dbDataFromPacket(packet)

	request = fmt.Sprintf("%s %s", packet.DBAction, task_type)
	if packet.KeyType == "tsds" {
		request = fmt.Sprintf("%s %s", request, packet.TimeDot)
	}
	if packet.ParentNS != "" {
		request = fmt.Sprintf("%s %s", request, packet.ParentNS)
	}
	request = fmt.Sprintf("%s %s", request, dbdata)

	return []byte(request)
}

/* formulate Task-Type from other info in Packet */
func taskTypeFromPacket(packet *goshare.Packet) (task_type string) {
	if packet.ValType == "" {
		packet.ValType = "default"
	}

	task_type = fmt.Sprintf("%s-%s", packet.KeyType, packet.ValType)

	if packet.ParentNS != "" {
		task_type = fmt.Sprintf("%s-parentNS", task_type)
	}
	return task_type
}

/* formulate DBData from other info in Packet */
func dbDataFromPacket(packet *goshare.Packet) string {
	if packet.ValType == "default" {
		return dbDataForDefaultVal(packet)
	}
	return dbDataForEncodedVal(packet)
}

/* formulate dbdata for default val-type FOR dbDataFromPacket */
func dbDataForDefaultVal(packet *goshare.Packet) (dbdata string) {
	switch packet.DBAction {
	case "push":
		for _key, _val := range packet.HashMap {
			dbdata = fmt.Sprintf("%s %s", _key, _val)
			break
		}
	case "read", "delete":
		for _, _val := range packet.KeyList {
			dbdata = _val
			break
		}
	default:
		//to-be-log
	}
	return dbdata
}

/* formulate dbdata for non-default val-type FOR dbDataFromPacket */
func dbDataForEncodedVal(packet *goshare.Packet) (dbdata string) {
	switch packet.DBAction {
	case "push":
		hmap_engine := golhashmap.GetHashMapEngine(packet.ValType)
		dbdata = hmap_engine.FromHashMap(packet.HashMap)
	case "read", "delete":
		list_engine := gollist.GetListEngine(packet.ValType)
		dbdata = list_engine.FromList(packet.KeyList)
	default:
		//to-be-log
	}
	return dbdata
}
