load utils

@test "Should serve A records" {
  set_etcd_record "com/example/A" "123.123.123.123"
  ip=$(dig_record "example.com." "A")
  [ "$ip" = "123.123.123.123" ]
}

@test "Should serve PTR records" {
  set_etcd_record "arpa/in-addr/12/34/56/78/PTR" "example.com."
  addr=$(dig_record "78.56.34.12.in-addr.arpa." "PTR")
  [ "$addr" = "example.com." ]
}

@test "Should serve CNAME records" {
  set_etcd_record "com/example2/CNAME" "example.com."
  addr=$(dig_record "example2.com." "CNAME")
  [ "$addr" = "example.com." ]
}

@test "Should forward queries to -forward if not in etcd" {
  ip=$(dig_record "probablyfine.co.uk." "A")
  [ "$ip" = "162.243.71.204" ]
}
