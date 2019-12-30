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
	onPublish        func(clientID string, topic, payload []byte, dup, retain, qos byte)
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
			f, ok := h.(func(clientID string, topic, payload []byte, dup, retain, qos byte))
			log.Println("plugin loader OnPublish", pstring, ok)
			if ok {
				pluginObj.onPublish = f
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
	}

	return true
}
