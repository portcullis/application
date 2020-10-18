package application

// Option for an Application
type Option func(app *Application)

// RunModule adds the specified module to the application for execution
func RunModule(name string, m Module) Option {
	return func(a *Application) {
		a.Controller.Add(name, m)
	}
}
