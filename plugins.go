package gott

import (
	"log"
	"path"
	"plugin"
)

const pluginDir = "plugins"

type gottPlugin struct {
	name             string
	plug             *plugin.Plugin
	onConnect        func(clientID, username, password string) bool
	onConnectSuccess func(clientID, username, password string) bool
	onBeforePublish     func(clientID, username string, topic, payload []byte, dup, qos byte, retain bool) bool
	onPublish           func(clientID, username string, topic, payload []byte, dup, qos byte, retain bool)
	onBeforeSubscribe   func(clientID, username string, topic []byte, qos byte) bool
	onSubscribe         func(clientID, username string, topic []byte, qos byte)
	onBeforeUnsubscribe func(clientID, username string, topic []byte) bool
	onUnsubscribe       func(clientID, username string, topic []byte)
}

func (b *Broker) bootstrapPlugins() {
	for _, pstring := range b.config.Plugins {
		p, err := plugin.Open(path.Join(pluginDir, pstring))
		if err != nil {
			log.Fatalf("Error loading plugin %s: %v", pstring, err)
		}

		pluginObj := gottPlugin{
			name: pstring,
			plug: p,
		}

		if bootstrap, err := p.Lookup("Bootstrap"); err == nil {
			if b, ok := bootstrap.(func()); ok {
				b()
			}
		}

		h, err := p.Lookup("OnConnect")
		if err == nil {
			f, ok := h.(func(clientID, username, password string) bool)
			log.Println("plugin loader OnConnect", pstring, ok)
			if ok {
				pluginObj.onConnect = f
			}
		}

		if h, err = p.Lookup("OnConnectSuccess"); err == nil {
			f, ok := h.(func(clientID, username, password string) bool)
			log.Println("plugin loader OnConnectSuccess", pstring, ok)
			if ok {
				pluginObj.onConnectSuccess = f
			}
		}

		if h, err = p.Lookup("OnBeforePublish"); err == nil {
			f, ok := h.(func(clientID, username string, topic, payload []byte, dup, qos byte, retain bool) bool)
			log.Println("plugin loader OnBeforePublish", pstring, ok)
			if ok {
				pluginObj.onBeforePublish = f
			}
		}

		if h, err = p.Lookup("OnPublish"); err == nil {
			f, ok := h.(func(clientID, username string, topic, payload []byte, dup, qos byte, retain bool))
			log.Println("plugin loader OnPublish", pstring, ok)
			if ok {
				pluginObj.onPublish = f
			}
		}

		if h, err = p.Lookup("OnBeforeSubscribe"); err == nil {
			f, ok := h.(func(clientID, username string, topic []byte, qos byte) bool)
			log.Println("plugin loader OnBeforeSubscribe", pstring, ok)
			if ok {
				pluginObj.onBeforeSubscribe = f
			}
		}

		if h, err = p.Lookup("OnSubscribe"); err == nil {
			f, ok := h.(func(clientID, username string, topic []byte, qos byte))
			log.Println("plugin loader OnSubscribe", pstring, ok)
			if ok {
				pluginObj.onSubscribe = f
			}
		}

		if h, err = p.Lookup("OnBeforeUnsubscribe"); err == nil {
			f, ok := h.(func(clientID, username string, topic []byte) bool)
			log.Println("plugin loader OnBeforeUnsubscribe", pstring, ok)
			if ok {
				pluginObj.onBeforeUnsubscribe = f
			}
		}

		if h, err = p.Lookup("OnUnsubscribe"); err == nil {
			f, ok := h.(func(clientID, username string, topic []byte))
			log.Println("plugin loader OnUnsubscribe", pstring, ok)
			if ok {
				pluginObj.onUnsubscribe = f
			}
		}

		b.plugins = append(b.plugins, pluginObj)
	}
}

func (b *Broker) notifyPlugins(event int, args ...interface{}) bool {
	switch event {
	case eventConnect:
		for _, p := range GOTT.plugins {
			if p.onConnect != nil {
				if !p.onConnect(args[0].(string), args[1].(string), args[2].(string)) {
					return false
				}
			}
		}
	case eventConnectSuccess:
		for _, p := range GOTT.plugins {
			if p.onConnectSuccess != nil {
				if !p.onConnectSuccess(args[0].(string), args[1].(string), args[2].(string)) {
					return false
				}
			}
		}
func (b *Broker) invokeOnBeforePublish(clientID, username string, topic, payload []byte, dup, qos byte, retain bool) bool {
	for _, p := range b.plugins {
		if p.onBeforePublish != nil {
			if !p.onBeforePublish(clientID, username, topic, payload, dup, qos, retain) {
				return false
			}
		}
	}

	return true
}
func (b *Broker) invokeOnBeforeSubscribe(clientID, username string, topic []byte, qos byte) bool {
	for _, p := range b.plugins {
		if p.onBeforeSubscribe != nil {
			if !p.onBeforeSubscribe(clientID, username, topic, qos) {
				return false
			}
		}
	}

	return true
}

func (b *Broker) invokeOnSubscribe(clientID, username string, topic []byte, qos byte) {
	for _, p := range b.plugins {
		if p.onSubscribe != nil {
			p.onSubscribe(clientID, username, topic, qos)
		}
	}
}

func (b *Broker) invokeOnBeforeUnsubscribe(clientID, username string, topic []byte) bool {
	for _, p := range b.plugins {
		if p.onBeforeUnsubscribe != nil {
			if !p.onBeforeUnsubscribe(clientID, username, topic) {
				return false
			}
		}
	}

	return true
}

func (b *Broker) invokeOnUnsubscribe(clientID, username string, topic []byte) {
	for _, p := range b.plugins {
		if p.onUnsubscribe != nil {
			p.onUnsubscribe(clientID, username, topic)
		}
	}
}
