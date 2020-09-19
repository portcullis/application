package flags

import (
	"context"
	"flag"
	"os"

	"github.com/portcullis/module"
)

type flagModule struct {
	fs *flag.FlagSet
}

// New returns a new module.Module that will expose and parse command line flags using the go flag package
func New() module.Module {
	m := &flagModule{
		fs: flag.CommandLine,
	}

	m.fs.Init("Application", flag.ContinueOnError)

	return m
}

func (m *flagModule) PreStart(ctx context.Context) error {
	return m.fs.Parse(os.Args[1:])
}

func (m *flagModule) Start(ctx context.Context) error {
	return nil
}

func (m *flagModule) Stop(ctx context.Context) error {
	return nil
}
