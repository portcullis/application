package application

import (
	"github.com/portcullis/application/modules/flags"
	"github.com/portcullis/application/modules/logging"
	"github.com/portcullis/module"
)

// Bootstrap a new application with the default modules added
func Bootstrap(name, version string, opts ...Option) *Application {
	app := &Application{
		Name:       name,
		Version:    version,
		Controller: &module.Controller{},
	}

	// add the default modules here
	app.Controller.Add("Flags", flags.New())
	app.Controller.Add("Logging", logging.New())

	for _, opt := range opts {
		opt(app)
	}

	return app
}
