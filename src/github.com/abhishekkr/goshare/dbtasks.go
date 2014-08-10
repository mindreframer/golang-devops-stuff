package goshare

/*
[PATTERN]
action {read, push, delete}
type {default, ns, tsds, now}

## message_array here is devoided of axn and key_type
non-tsds {key&val, :type-data}
tsds(-*) {tdot&key&val, tdot&:type-data}
*/

/* Insulates communication from DBTasks
Communications handled on byte streams can use it by passing standard-ized packet-array
it prepares Packet and passes on to TasksOnPacket, 0MQ utilizes it */
func DBTasks(packet_array []string) ([]byte, bool) {
	packet := CreatePacket(packet_array)
	return DBTasksOnPacket(packet)
}

/* Insulates communication from DBTasks
Communication can directly create packet and pass it here, HTTP utilizes it directly */
func DBTasksOnPacket(packet Packet) ([]byte, bool) {
	response := ""
	axn_status := false

	switch packet.DBAction {
	case "read":
		// returns axn error if key has empty value, if you gotta store then store, don't keep placeholders
		response = ReadFromPacket(packet)
		if response != "" {
			axn_status = true
		}

	case "push":
		axn_status = PushFromPacket(packet)

	case "delete":
		axn_status = DeleteFromPacket(packet)
	}

	return []byte(response), axn_status
}
