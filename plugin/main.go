package plugin

import (
	"github.com/mach-composer/mach-composer-plugin-sdk/plugin"

	"github.com/mach-composer/mach-composer-plugin-aws/internal"
)

func Serve() {
	p := internal.NewAWSPlugin()
	plugin.ServePlugin(p)
}
