package gott

import (
	"log"
	"net"
	"path"
	"plugin"
)

const pluginDir = "plugins"

type gottPlugin struct {
	name                string
	plug                *plugin.Plugin
	onSocketOpen        func(conn net.Conn) bool
	onBeforeConnect     func(clientID, username, password string) bool
	onConnect           func(clientID, username, password string) bool
	onBeforePublish     func(clientID, username string, topic, payload []byte, dup, qos byte, retain bool) bool
	onPublish           func(clientID, username string, topic, payload []byte, dup, qos byte, retain bool)
	onBeforeSubscribe   func(clientID, username string, topic []byte, qos byte) bool
	onSubscribe         func(clientID, username string, topic []byte, qos byte)
	onBeforeUnsubscribe func(clientID, username string, topic []byte) bool
	onUnsubscribe       func(clientID, username string, topic []byte)
	onDisconnect        func(clientID, username string)
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

		h, err := p.Lookup("OnSocketOpen")
		if err == nil {
			f, ok := h.(func(conn net.Conn) bool)
			LogDebug("plugin loader OnSocketOpen", pstring, ok)
			if ok {
				pluginObj.onSocketOpen = f
			}
		}

		h, err = p.Lookup("OnBeforeConnect")
		if err == nil {
			f, ok := h.(func(clientID, username, password string) bool)
			LogDebug("plugin loader OnBeforeConnect", pstring, ok)
			if ok {
				pluginObj.onBeforeConnect = f
			}
		}

		if h, err = p.Lookup("OnConnect"); err == nil {
			f, ok := h.(func(clientID, username, password string) bool)
			LogDebug("plugin loader OnConnect", pstring, ok)
			if ok {
				pluginObj.onConnect = f
			}
		}

		if h, err = p.Lookup("OnBeforePublish"); err == nil {
			f, ok := h.(func(clientID, username string, topic, payload []byte, dup, qos byte, retain bool) bool)
			LogDebug("plugin loader OnBeforePublish", pstring, ok)
			if ok {
				pluginObj.onBeforePublish = f
			}
		}

		if h, err = p.Lookup("OnPublish"); err == nil {
			f, ok := h.(func(clientID, username string, topic, payload []byte, dup, qos byte, retain bool))
			LogDebug("plugin loader OnPublish", pstring, ok)
			if ok {
				pluginObj.onPublish = f
			}
		}

		if h, err = p.Lookup("OnBeforeSubscribe"); err == nil {
			f, ok := h.(func(clientID, username string, topic []byte, qos byte) bool)
			LogDebug("plugin loader OnBeforeSubscribe", pstring, ok)
			if ok {
				pluginObj.onBeforeSubscribe = f
			}
		}

		if h, err = p.Lookup("OnSubscribe"); err == nil {
			f, ok := h.(func(clientID, username string, topic []byte, qos byte))
			LogDebug("plugin loader OnSubscribe", pstring, ok)
			if ok {
				pluginObj.onSubscribe = f
			}
		}

		if h, err = p.Lookup("OnBeforeUnsubscribe"); err == nil {
			f, ok := h.(func(clientID, username string, topic []byte) bool)
			LogDebug("plugin loader OnBeforeUnsubscribe", pstring, ok)
			if ok {
				pluginObj.onBeforeUnsubscribe = f
			}
		}

		if h, err = p.Lookup("OnUnsubscribe"); err == nil {
			f, ok := h.(func(clientID, username string, topic []byte))
			LogDebug("plugin loader OnUnsubscribe", pstring, ok)
			if ok {
				pluginObj.onUnsubscribe = f
			}
		}

		if h, err = p.Lookup("OnDisconnect"); err == nil {
			f, ok := h.(func(clientID, username string))
			LogDebug("plugin loader OnDisconnect", pstring, ok)
			if ok {
				pluginObj.onDisconnect = f
			}
		}

		b.plugins = append(b.plugins, pluginObj)

		LogDebug("plugin loaded:", pstring)
	}
}

func (b *Broker) invokeOnSocketOpen(conn net.Conn) bool {
	for _, p := range b.plugins {
		if p.onSocketOpen != nil {
			if !p.onSocketOpen(conn) {
				return false
			}
		}
	}

	return true
}

func (b *Broker) invokeOnBeforeConnect(clientID, username, password string) bool {
	for _, p := range b.plugins {
		if p.onBeforeConnect != nil {
			if !p.onBeforeConnect(clientID, username, password) {
				return false
			}
		}
	}

	return true
}

func (b *Broker) invokeOnConnect(clientID, username, password string) bool {
	for _, p := range b.plugins {
		if p.onConnect != nil {
			if !p.onConnect(clientID, username, password) {
				return false
			}
		}
	}

	return true
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

func (b *Broker) invokeOnPublish(clientID, username string, topic, payload []byte, dup, qos byte, retain bool) {
	for _, p := range b.plugins {
		if p.onPublish != nil {
			p.onPublish(clientID, username, topic, payload, dup, qos, retain)
		}
	}
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

func (b *Broker) invokeOnDisconnect(clientID, username string) {
	for _, p := range b.plugins {
		if p.onDisconnect != nil {
			p.onDisconnect(clientID, username)
		}
	}
}
