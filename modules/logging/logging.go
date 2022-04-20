package logging

import (
	"context"
	"flag"
	"os"

	"github.com/portcullis/config"
	"github.com/portcullis/logging"
	writer "github.com/portcullis/logging/format/simple"
)

type module struct {
	cfg    *cfg
	writer *writer.Writer
}

type cfg struct {
	Level logging.Level `flag:"level" description:"Logging level to output"`
}

// New returns a new logging module.Module
func New() *module {
	m := &module{
		cfg: &cfg{
			Level: logging.LevelInformational,
		},
	}

	m.writer = writer.New(
		os.Stdout,
		writer.Level(m.cfg.Level),
	)

	config.Default.
		Subset("Logging").
		Bind(m.cfg).
		Notify(config.NotifyFunc(func(s *config.Setting) {
			writer.Level(m.cfg.Level)(m.writer)
		}))

	logging.DefaultLog = logging.New(
		logging.WithWriter(m.writer),
	)

	// BUG: There is an issue where if you do -level=debug that it isn't read until Initialize() of the Flag package, we might have to duplicate that, and eventually the ENV variable...
	//      This isn't totally ideal, but we dont' want to miss any logging at the beginning of the application

	flag.CommandLine.Parse(os.Args[1:])

	return m
}

func (m module) Start(ctx context.Context) error {
	return nil
}

func (m module) Stop(ctx context.Context) error {
	return nil
}
