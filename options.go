package application

import "github.com/portcullis/module"

// Option for an Application
type Option func(app *Application)

// Module adds the specified module to the application
func Module(name string, m module.Module) Option {
	return func(a *Application) {
		a.Controller.Add(name, m)
	}
}
