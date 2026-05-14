# monitor

Meshtastic health reporting to track stationary nodes in Grafana

     Meshtastic Node -> (Wi-Fi/USB) -> monitor -> Victoria Metrics (DB) -> Grafana (display)

Collects useful stats for performance tests over time.




# Development

Recompile protobufs (meshtastic subdmoule)

     rm -rf github.com/meshtastic/go/generated
     protoc -I protobufs --go_out=. protobufs/nanopb.proto $(find protobufs/meshtastic -name '*.proto')