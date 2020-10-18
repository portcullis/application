package flags

import (
	"context"
	"flag"
	"os"
)

type module struct {
	fs *flag.FlagSet
}

// New returns a new application.Module implementation that will expose and parse command line flags using the go flag package
func New() *module {
	m := &module{
		fs: flag.CommandLine,
	}

	m.fs.Init("Application", flag.ContinueOnError)

	return m
}

func (m *module) Initialize(ctx context.Context) (context.Context, error) {
	return nil, m.fs.Parse(os.Args[1:])
}

func (m *module) Start(ctx context.Context) error {
	return nil
}

func (m *module) Stop(ctx context.Context) error {
	return nil
}
