package gott

import (
	"log"
	"path"
	"plugin"
)

const pluginDir = "plugins"

type gottPlugin struct {
	name      string
	plug      *plugin.Plugin
	onConnect func(clientID string) bool
}

var plugins []*gottPlugin

func (b *Broker) bootstrapPlugins() {
	for _, pstring := range b.config.Plugins {
		p, err := plugin.Open(path.Join(pluginDir, pstring))
		if err != nil {
			log.Fatalf("Error loading plugin %s: %v", pstring, err)
		}

		pluginObj := &gottPlugin{
			name: pstring,
			plug: p,
		}

		if bootstrap, err := p.Lookup("Bootstrap"); err == nil {
			if b, ok := bootstrap.(func()); ok {
				b()
			}
		}

		if onconn, err := p.Lookup("OnConnect"); err == nil {
			onconnFunc, ok := onconn.(func(string) bool)
			if ok {
				pluginObj.onConnect = onconnFunc
			}
		}

		plugins = append(plugins, pluginObj)
	}
}
