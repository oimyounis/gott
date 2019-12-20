# GOTT
GOTT is a MQTT Broker written in pure Go. Aims to be a high-performance pluggable broker with most of the features that could be embedded in a broker available out of the box.  
  
It needs a lot of optimizations and improvements to be functional and usable for production.  
**Hopefully with your contribution we could build something great!**

## Project Status
#### Under Development
- GOTT is currently in a very early stage and is under heavy development. It is not stable or complete enough to be used in production.
- Currently implementing the MQTT v3.1.1 spec.

## Planned for v1
- [x] Ping (client -> server)
- [x] Topic filtering with wildcards support
- [x] Subscriptions
- [x] QoS 0 messages
- [x] QoS 1 messages
- [x] QoS 2 messages
- [x] Unsubscribe a client when it disconnects (finished but still needs optimizations)
- [x] Retained messages
- [ ] Will messages
- [ ] Username and password authentication
- [ ] Sessions
- [ ] Plugins

### Planned for v2
- [ ] MQTT v5
- [ ] TLS
- [ ] Clustering (maybe)
- [ ] WebSockets

## Installation
Just clone/download this repo and build/run `main/main.go`.

## License
Apache License 2.0, see LICENSE.

## Contribution
You are very welcome to submit a new feature, fix a bug, an optimization to the code, report a bug or even a benchmark would be helpful.  
### To Contribute:  
Open an issue or:
1. Fork this repo.
2. Create a new branch with a descriptive name (example: *feature/some-new-function* or *fix/something-somewhere*).
3. Commit and push your code to your new branch.
4. Create a new Pull Request here.  
