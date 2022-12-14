package main

import (
	"github.com/mach-composer/mach-composer-plugin-sdk/plugin"

	"github.com/mach-composer/mach-composer-plugin-aws/internal"
)

func main() {
	p := internal.NewAWSPlugin()
	plugin.ServePlugin(p)
}
