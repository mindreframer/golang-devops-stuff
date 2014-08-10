package goshare

import (
	golhashmap "github.com/abhishekkr/gol/golhashmap"
	levigoNS "github.com/abhishekkr/levigoNS"
	abkleveldb "github.com/abhishekkr/levigoNS/leveldb"
	levigoTSDS "github.com/abhishekkr/levigoTSDS"
)

/* Get value of given key */
func ReadKey(key string) golhashmap.HashMap {
	var hashmap golhashmap.HashMap
	hashmap = make(golhashmap.HashMap)
	val := abkleveldb.GetVal(key, db)
	if val == "" {
		return hashmap
	}
	hashmap[key] = val
	return hashmap
}

/* Get value for all descendents of Namespace */
func ReadKeyNS(key string) golhashmap.HashMap {
	return levigoNS.ReadNSRecursive(key, db)
}

/* Get value for the asked time-frame key, aah same NS */
func ReadKeyTSDS(key string) golhashmap.HashMap {
	return levigoTSDS.ReadTSDS(key, db)
}

/* Delete a key on task-type */
func ReadFuncByKeyType(key_type string) FunkAxnParamKeyReturnMap {
	switch key_type {
	case "tsds":
		return ReadKeyTSDS

	case "ns":
		return ReadKeyNS

	default:
		return ReadKey

	}
}

/* Delete multi-item */
func ReadFromPacket(packet Packet) string {
	var response string
	var hashmap golhashmap.HashMap
	hashmap = make(golhashmap.HashMap)

	axnFunk := ReadFuncByKeyType(packet.KeyType)
	for _, _key := range packet.KeyList {
		hashmap = axnFunk(_key)
		if len(hashmap) == 0 {
			continue
		}
		response += responseByValType(packet.ValType, hashmap)
	}

	return response
}

/* transform response by ValType, if none default:csv */
func responseByValType(valType string, response_map golhashmap.HashMap) string {
	var response string

	switch valType {
	case "csv", "json":
		hashmapEngine := golhashmap.GetHashMapEngine(valType)
		response = hashmapEngine.FromHashMap(response_map)

	default:
		response = golhashmap.HashMapToCSV(response_map)
	}
	return response
}
