package main

import "github.com/mach-composer/mach-composer-plugin-sdk/plugin"

func Serve() {
	p := NewAWSPlugin()
	plugin.ServePlugin(p)
}
