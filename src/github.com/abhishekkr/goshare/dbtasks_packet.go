package goshare

import (
	"fmt"
	"strings"

	golhashmap "github.com/abhishekkr/gol/golhashmap"
	gollist "github.com/abhishekkr/gol/gollist"
	goltime "github.com/abhishekkr/gol/goltime"
)

/* structure for packet metadata */
type Packet struct {
	DBAction string
	TaskType string

	KeyType string // key: default, namespace key: ns, timeseries key: tsds, timeseries for goshare time: now
	ValType string // single: default, csv, json

	HashMap golhashmap.HashMap
	KeyList []string

	ParentNS string // allowed for ns|tsds|now
	TimeDot  goltime.Timestamp
}

type FunkAxnParamKeyVal func(key string, val string) bool
type FunkAxnParamKey func(key string) bool
type FunkAxnParamKeyReturnMap func(key string) golhashmap.HashMap

/*
Create Packet from passed message array
*/
func CreatePacket(packet_array []string) Packet {
	packet := Packet{}
	packet.HashMap = make(golhashmap.HashMap)

	packet.DBAction = packet_array[0]
	packet.TaskType = packet_array[1]
	data_starts_from := 2

	task_type_tokens := strings.Split(packet.TaskType, "-")
	packet.KeyType = task_type_tokens[0]
	if packet.KeyType == "tsds" && packet.DBAction == "push" {
		packet.TimeDot = goltime.CreateTimestamp(packet_array[2:8])
		data_starts_from += 6
	}

	if len(task_type_tokens) > 1 {
		packet.ValType = task_type_tokens[1]

		if len(task_type_tokens) == 3 {
			// if packet requirement grows more than 3, that's the limit
			// go get 'msgpack' to handle it instead...
			thirdTokenFeature(&packet, packet_array, &data_starts_from, task_type_tokens[2])
		}
	}

	decodeData(&packet, packet_array[data_starts_from:])
	return packet
}

/* Special 3rd token feature */
func thirdTokenFeature(packet *Packet, packet_array []string, data_starts_from *int, token string) {
	switch token {
	case "parent":
		packet.ParentNS = packet_array[*data_starts_from]
		*data_starts_from += 1
	}
}

/*
Handles Packet formation: DBData calls according to Axn; handles TimeDot
*/
func decodeData(packet *Packet, message_array []string) {
	switch packet.DBAction {
	case "read", "delete":
		packet.KeyList = decodeKeyData(packet.ValType, message_array)
		if packet.ParentNS != "" {
			PrefixKeyParentNamespace(packet)
		}

	case "push":
		packet.HashMap = decodeKeyValData(packet.ValType, message_array)
		if packet.ParentNS != "" {
			PrefixKeyValParentNamespace(packet)
		}
	}
}

/*
Handles Packet formation: DBData based on ValType for GET|DELETE
*/
func decodeKeyData(valType string, message_array []string) []string {
	switch valType {
	case "csv", "json":
		multi_value := strings.Join(message_array, "\n")
		listEngine := gollist.GetListEngine(valType)
		return listEngine.ToList(multi_value)

	default:
		return []string{message_array[0]}
	}
}

/*
Handles Packet formation: DBData based on ValType for PUSH
*/
func decodeKeyValData(valType string, message_array []string) golhashmap.HashMap {
	var hashmap golhashmap.HashMap
	hashmap = make(golhashmap.HashMap)

	switch valType {
	case "csv", "json":
		multi_value := strings.Join(message_array, "\n")
		hashmapEngine := golhashmap.GetHashMapEngine(valType)
		hashmap = hashmapEngine.ToHashMap(multi_value)

	default:
		key := message_array[0]
		value := strings.Join(message_array[1:], " ")
		hashmap[key] = value
	}
	return hashmap
}

/*
Prefixes Parent Namespaces to all keys in List if val for 'parent_namespace'
*/
func PrefixKeyParentNamespace(packet *Packet) {
	var new_list []string
	new_list = make([]string, len(packet.KeyList))

	parent_namespace := packet.ParentNS
	for idx, key := range packet.KeyList {
		new_list[idx] = fmt.Sprintf("%s:%s", parent_namespace, key)
	}
	packet.KeyList = new_list
}

/*
Prefixes Parent Namespaces to all key-val in HashMap if it has val for 'parent_namespace'
*/
func PrefixKeyValParentNamespace(packet *Packet) {
	var new_hmap golhashmap.HashMap
	new_hmap = make(golhashmap.HashMap)

	parent_namespace := packet.ParentNS
	for key, val := range packet.HashMap {
		new_hmap[fmt.Sprintf("%s:%s", parent_namespace, key)] = val
	}
	packet.HashMap = new_hmap
}
