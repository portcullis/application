package application

import "log/slog"

// Option for an Application
type Option func(app *Application)

// WithModule adds the specified module to the application for execution
func WithModule(name string, m Module) Option {
	return func(a *Application) {
		a.Controller.Add(name, m)
	}
}

// WithConfigFile adds hcl parsing capability to the application and loads the provided filename
func WithConfigFile(filename string) Option {
	return func(a *Application) {
		a.configuration = &Configuration{}
		a.configFile = filename
	}
}

// WithLogger will set the internal slog.Logger instance
func WithLogger(logger *slog.Logger) Option {
	return func(a *Application) {
		a.Logger = logger
	}
}
