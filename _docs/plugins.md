# GOTT Plugins

Writing a GOTT plugin is easy yet powerful.  
GOTT plugins are typical Go plugins and require zero dependencies.  
They utilize a hooking system where you hook your code to certain events such as `OnConnect`, `OnSubscribe` and others.  
  
**Important**: all event invocations are blocking calls which may reduce the Broker's performance therefore, you should execute in goroutines all code that don't need to complete before resuming the Broker's workflow (eg. saving messages to a DB).  
It's up to you to determine whether you need goroutines or not. For example, following are situations where you don't need to use goroutines:
- Authentication
- Connection throttling
- Etc...

## Tutorial
We will build a simple plugin that limits subscription count per client ID.  
**Please note that this is not meant to be used in production scenarios**. While this plugin will work as expected by limiting the subscription count, it doesn't cover all cases. Code is shortened for the sake of simplicity.

### Implementing the plugin
  
First, we'll implement a small `limiter` struct to make our job easier:

```go
type limiter struct{
	record map[string]int
	max int
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

Next, is the first function that's executed when GOTT is loading plugins, `Bootstrap`:

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
This hook will be ignored for this subscription if `false` was returned from the `OnBeforeSubscribe` hook by any loaded plugin.  
  
We implement the opposite too, the `OnUnsubscribe` hook:
```go
func OnUnsubscribe(clientID, username string, topic []byte) {
	subLimiter.decrease(clientID)
}
```
Same as for the `OnSubscribe` hook, this hook will only be invoked on successful unsubscriptions. Meaning that it won't be invoked if there is no subscription found for this client ID in this topic filter.  
  
That's all the code we need. We move to the next step.  
  
### Building the plugin

Since GOTT plugins are typical Go plugins, we only need to compile them using the `go` CLI. Run the following command inside a plugin's directory:
```shell script
go build -buildmode=plugin -o gottSubLimiter.so main.go
```
***gottSubLimiter.so*** is the name of the output file.  
***main.go*** is the name of the file containing your Go code.

### Loading the plugin

Plugins are loaded in two simple steps:

1. Copy the output file (gottSubLimiter.so) to the `plugins` directory that is located in the same directory as the GOTT binary (or `main/main.go` if running from source).
2. Add the plugin's name to the `config.yml` file (located in the same directory as the `plugins` directory).

### Notes
- Please note that the `plugins` directory and the `config.yml` file will be created automatically on the first run of GOTT.
- Also note that they will be created in the ***Current Working Directory*** that was active at the time of execution. So make sure that you `cd` to the binary/script directory first then run the GOTT binary/script.
