package application

// Application defines an instance of an application
type Application struct {
	name    string
	version string
}

// Create a new application with the specified name and version
func Create(name, version string, opts ...Option) *Application {
	app := &Application{
		name:    name,
		version: version,
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}
