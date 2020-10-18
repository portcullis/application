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

// PreStarter allows for additional functionality before any module.Start is called
type PreStarter interface {
	PreStart(ctx context.Context) error
}

// PostStarter allows for additional functionality after all module.Start are called
type PostStarter interface {
	PostStart(ctx context.Context) error
}
