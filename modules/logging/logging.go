package logging

import (
	"context"
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

	return m
}

func (m module) Start(ctx context.Context) error {
	return nil
}

func (m module) Stop(ctx context.Context) error {
	return nil
}
