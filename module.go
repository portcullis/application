package application

import "context"

// Module represents an interface into a start stop module
type Module interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Initializer allows for context modifications before start
type Initializer interface {
	Initialize(ctx context.Context) (context.Context, error)
}

// PreStarter allows for additional functionality before any Module.Start is called
type PreStarter interface {
	PreStart(ctx context.Context) error
}

// PostStarter allows for additional functionality after all Module.Start are called
type PostStarter interface {
	PostStart(ctx context.Context) error
}

// Configurable can be optionally implemented by any module to accept user configuration.
type Configurable interface {
	// Config should return a pointer to an allocated configuration
	// structure. This structure will be written to directly with the
	// decoded configuration. If this returns nil, then it is as if
	// Configurable was not implemented.
	Config() (interface{}, error)
}

// Installer allows for modules to install things before and modul.PreStart is called, and is used in the application.Install function
type Installer interface {
	Install(ctx context.Context) error
}

// ConfigurableNotify is an optional interface that can be implemented
// by any module to receive a notification that the configuration
// was decoded.
type ConfigurableNotify interface {
	Configurable

	// ConfigSet is called with the value of the configuration after
	// decoding is complete successfully.
	ConfigSet(interface{}) error
}
