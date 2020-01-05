# GOTT Plugins

Writing a GOTT plugin is easy yet powerful.  
GOTT plugins are typical Go plugins that require zero dependencies.  
They utilize a hooking system where you hook your code to certain events such as `OnConnect`, `OnSubscribe` and others.  

## Tutorial
We will build a simple plugin that limits subscription count per client ID.  
**Please note that this is not meant to be used in production scenarios**. While this plugin will work as expected by limiting the subscription count, it doesn't cover all cases. Code is shortened for the sake of simplicity.

### Implementing The Plugin
  
First, we'll implement a small `limiter` struct to make our job easier:

```go
type limiter struct {
	record map[string]int
	max    int
}
```

Then, we create a constructor function:

```go
func newLimiter(max int) limiter {
	return limiter{
		record: map[string]int{},
		max:    max,
	}
}
```

Lastly, we add some functions to make it work:

```go
func (l *limiter) increase(key string) {
	if l.check(key) {
	    l.record[key]++
	}
}
```
```go
func (l *limiter) decrease(key string) {
	l.record[key]--
}
```
```go
func (l *limiter) check(key string) bool {
	if m, ok := l.record[key]; ok {
		return m < l.max
	}
	return true
}
```

Then, we declare a variable of type `limiter` in the global scope so it could be shared between hooks:

```go
var subLimiter limiter
```

Next, is the first function that's executed while GOTT is loading plugins, the `Bootstrap` function:

```go
func Bootstrap() {
	subLimiter = newLimiter(5)
}
```
Use this function to initialize any instances or code that needs to be executed before the plugin starts receiving event notifications. So we'll use it to initialize our `subLimiter` variable.  
  
Now we implement the `OnBeforeSubscribe` hook:

```go
func OnBeforeSubscribe(clientID, username string, topic []byte, qos byte) bool {
	allowed := subLimiter.check(clientID)
	if !allowed {
		log.Println("{gottSubLimiter} sub limit reached for:", clientID)
	}
	return allowed
}
```
The `OnBeforeSubscribe` hook receives the argument list mentioned above and returns a `bool`; indicating whether to accept the subscription or not.  
When `false` is returned, the Broker will skip registering this subscription for the client (acknowledgements are not affected and will be sent back to the client as normal either way).  
  
Now we implement the `OnSubscribe` hook that gets invoked after a subscription is successful:
```go
func OnSubscribe(clientID, username string, topic []byte, qos byte) {
	subLimiter.increase(clientID)
}
```
This hook will be ignored for this subscription if `false` was returned from the `OnBeforeSubscribe` hook of any loaded plugin.  
  
We implement the opposite too, the `OnUnsubscribe` hook:
```go
func OnUnsubscribe(clientID, username string, topic []byte) {
	subLimiter.decrease(clientID)
}
```
Same as for the `OnSubscribe` hook, this hook will only be invoked on successful unsubscriptions. Meaning that it won't be invoked if there is no subscription found for this client ID in this topic filter.  
  
That's all the code we need. We move to the next step.  
  
### Building The Plugin

Since GOTT plugins are typical Go plugins, we only need to compile them using the `go` CLI. Run the following command inside a plugin's `main` package:
```shell script
go build -buildmode=plugin -o gottSubLimiter.so main.go
```
***gottSubLimiter.so*** is the name of the output file.  
***main.go*** is the name of the file containing your Go code.

### Loading The Plugin

Plugins are loaded in two simple steps:

1. Copy the output file (gottSubLimiter.so) to the `plugins` directory that is located in the same directory as the GOTT binary (or `main/main.go` if running from source).
2. Add the plugin's name to the `config.yml` file (located in the same directory as the `plugins` directory).

### Notes
- Please note that the `plugins` directory and the `config.yml` file will be created automatically on the first run of GOTT.
- Also note that they will be created in the ***Current Working Directory*** that was active at the time of execution. So make sure that you `cd` to the binary/script directory first then run the GOTT binary/script.

### The Complete Code
Put together you should now have the following in your Go file:

```go
package main

import "log"

type limiter struct{
	record map[string]int
	max int
}

func newLimiter(max int) limiter {
	return limiter{
		record: map[string]int{},
		max:    max,
	}
}

func (l *limiter) increase(key string) {
	if l.check(key) {
		l.record[key]++
	}
}

func (l *limiter) decrease(key string) {
	l.record[key]--
}

func (l *limiter) check(key string) bool {
	if m, ok := l.record[key]; ok {
		return m < l.max
	}
	return true
}

var subLimiter limiter

func Bootstrap() {
	subLimiter = newLimiter(5)
}

func OnBeforeSubscribe(clientID, username string, topic []byte, qos byte) bool {
	allowed := subLimiter.check(clientID)
	if !allowed {
		log.Println("{gottsublimiter} sub limit reached for:", clientID, subLimiter.record[clientID])
	}
	return allowed
}

func OnSubscribe(clientID, username string, topic []byte, qos byte) {
	subLimiter.increase(clientID)
}

func OnUnsubscribe(clientID, username string, topic []byte) {
	subLimiter.decrease(clientID)
}
```

## Documentation
GOTT plugins are regular Go plugins that utilize a hooking system that will allow you to connect Go code to certain events.  
*All hooks and functions are optional and must be exported.*  
  
**Important**: event invocations are blocking calls which may reduce the Broker's performance therefore, you should execute in goroutines all the code that don't need to complete before resuming the Broker's workflow (eg. saving messages to a DB).  
It's up to you to decide whether you need goroutines or not. For example, following are situations where you wouldn't need to use goroutines:
- Authentication
- Connection throttling
- Etc...

### Plugin Loading  
Let's say that we want to initialize some instances of some structs or maybe load a config file for our plugin before proceeding with receiving event notifications on our hooks. To do so, GOTT will execute a special exported function (if found) called `Bootstrap`:  
```go
func Bootstrap()
```
This function counts as the main entry point of a plugin but it is not required.

### Events And Hooks
Following is a list of each available event and the hook to implement for it:
 
#### SocketOpen Event  
Invoked when a new client is trying to connect to the Broker as soon as a socket is opened.
```go
func OnSocketOpen(conn net.Conn) bool
```
The `OnSocketOpen` hook is executed before any other hook.  
Receives the underlying connection that was established with the client and returns a `bool` to indicate whether to accept the connection or refuse and close the socket. This is a good place to store the connection to use it with other protocols, implement connection throttling or IP black/whitelisting and so on.

#### BeforeConnect Event  
Invoked when a CONNECT packet is received by the Broker, before initializing/loading sessions and before sending back the CONNACK packet.
```go
func OnBeforeConnect(clientID, username, password string) bool
```
The `OnBeforeConnect` hook receives the above argument list and returns a `bool` to indicate whether to accept the connection as a MQTT client or disconnect and close the connection. This would be a good place to implement different authentication methods, define a format for the *clientID*, etc... 

#### Connect Event
Follows the *BeforeConnect* event when the connection is accepted, sessions are ready and the CONNACK packet has been sent back to the client.
```go
func OnConnect(clientID, username, password string) bool
```
The `OnConnect` hook is treated the same as the `OnBeforeConnect` hook. Receives the same argument list and returns a `bool` that will disconnect the client in case of `false` was returned. 

#### Message Event  
When the Broker receives a PUBLISH packet this event will be invoked. Acknowledgements for QoS 1 & 2 messages are sent prior to this event.
```go
func OnMessage(clientID, username string, topic, payload []byte, dup, qos byte, retain bool)
```
The `OnMessage` hook will be passed the above argument list as received by the publisher of this message. This would be a good place to implement message persistence.

#### BeforePublish Event  
This event is invoked after the *Message* event has been invoked when the Broker is about to send out the message to the subscribers of this Topic.
```go
func OnBeforePublish(clientID, username string, topic, payload []byte, dup, qos byte, retain bool) bool
```
The `OnBeforePublish` hook will be passed the above argument list as received by the publisher of the original message and returns a `bool` to indicate whether to proceed with the publishing process or not (acknowledgements are not affected).

#### Publish Event
Invoked when the Broker successfully publishes a received message. Will be ignored if `false` was returned by the `OnBeforePublish` hook *of any loaded plugin*.
```go
func OnPublish(clientID, username string, topic, payload []byte, dup, qos byte, retain bool)
```
The `OnPublish` hook is also passed the above argument list as received by the publisher of the original message.

