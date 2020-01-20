# GOTT

[![Go Report Card](https://goreportcard.com/badge/github.com/oimyounis/gott)](https://goreportcard.com/report/github.com/oimyounis/gott)
[![GoDoc](https://godoc.org/github.com/oimyounis/gott?status.svg)](https://godoc.org/github.com/oimyounis/gott)
![GitHub](https://img.shields.io/github/license/oimyounis/gott)

GOTT is a MQTT Broker written in pure Go. Aims to be a high-performance, easy-to-use pluggable broker with most of the features that could be embedded in a broker available out of the box.  
    
**Hopefully with your contribution we could build something great!**

## Project Status
### In BETA
- GOTT is currently in a BETA stage. All planned features fully work as tested while developed. Still, there is room for improvement. More tests and optimizations are needed.

### Planned for v1 (MQTT v3.1.1)
- [x] Ping (client -> server)
- [x] Topic filtering with wildcards support
- [x] Subscriptions
- [x] QoS 0 messages
- [x] QoS 1 messages
- [x] QoS 2 messages
- [x] Retained messages
- [x] Will messages
- [x] Sessions
- [x] Plugins
- [x] Logging to disk (with levels and log rotation)
- [x] TLS/SSL

### Planned for v2
- [ ] MQTT v5
- [ ] Clustering (maybe)
- [ ] WebSockets

### Known Issues
- Restarting the broker will reset existing subscriptions. They are not saved to disk (sessions are saved to disk but subscriptions are not restored on broker restart).

## Quick Start
1. Install dependencies:  
```shell script
$ go get -u github.com/google/uuid
$ go get -u github.com/dgraph-io/badger
$ go get -u github.com/json-iterator/go
$ go get -u go.uber.org/zap
$ go get -u gopkg.in/natefinch/lumberjack.v2
$ go get -u gopkg.in/yaml.v2
```
2. Clone/download this repo.
3. Run `cd main` inside the project's directory.
4. Run `go run main.go`.

## Plugins
GOTT implements a plugin system that is very easy to work with. You can easily build your own plugin that does whatever you want.  
  
Start by reading the [plugins documentation](_docs/plugins.md).

## License
Apache License 2.0, see [LICENSE](LICENSE).

## Contribution
You are very welcome to submit a new feature, fix a bug, an optimization to the code, report a bug or even a benchmark would be helpful.  
### To Contribute:  
Open an issue or:
1. Fork this repo.
2. Create a new branch with a descriptive name (example: *feature/some-new-function* or *fix/something-somewhere*).
3. Commit and push your code to your new branch.
4. Create a new Pull Request here.  
