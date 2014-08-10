package goshare

import (
	levigoNS "github.com/abhishekkr/levigoNS"
	abkleveldb "github.com/abhishekkr/levigoNS/leveldb"
	levigoTSDS "github.com/abhishekkr/levigoTSDS"
)

/* Empty Val for a given Key */
func DelKey(key string) bool {
	return abkleveldb.DelKey(key, db)
}

/* Delete a Namespace Key and all its value */
func DelKeyNS(key string) bool {
	return levigoNS.DeleteNSRecursive(key, db)
}

/* Delete all keys under given namespace, same as NS */
func DelKeyTSDS(key string) bool {
	return levigoTSDS.DeleteTSDS(key, db)
}

/* Delete a key on task-type */
func DeleteFuncByKeyType(key_type string) FunkAxnParamKey {
	switch key_type {
	case "tsds":
		return DelKeyTSDS

	case "ns":
		return DelKeyNS

	default:
		return DelKey

	}
}

/* Delete multi-item */
func DeleteFromPacket(packet Packet) bool {
	status := true
	axnFunk := DeleteFuncByKeyType(packet.KeyType)
	for _, _key := range packet.KeyList {
		status = status && axnFunk(_key)
	}

	return status
}
