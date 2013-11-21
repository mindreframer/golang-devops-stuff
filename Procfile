web: python -m SimpleHTTPServer 8000
replayed_web: python -m SimpleHTTPServer 8001
listener: sudo -E go run ./bin/gor.go --input-raw :8000 --output-tcp :8002 --verbose
replay: go run ./bin/gor.go --input-tcp :8002 --output-http localhost:8001 --verbose
