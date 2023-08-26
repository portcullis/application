package application

import (
	"log/slog"

	"github.com/portcullis/application/modules/flags"
)

// Bootstrap a new application with the default modules added.
//
// Default Modules.
//
// - Flags: Parse the flag.CommandLine.
//
// - Logging: Configure and setup logging to stdout, as well as set the level through configuration.
func Bootstrap(name, version string, opts ...Option) *Application {
	app := &Application{
		Name:       name,
		Version:    version,
		Controller: &Controller{},
		Logger:     slog.Default(),
	}

	// add the default modules here
	app.Controller.Add("Flags", flags.New())

	for _, opt := range opts {
		opt(app)
	}

	return app
}
