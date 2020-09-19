package logging

import (
	"context"
	"flag"
	"os"

	"github.com/portcullis/config"
	"github.com/portcullis/logging"
	writer "github.com/portcullis/logging/format/simple"
	"github.com/portcullis/module"
)

type loggingModule struct {
	cfg    *cfg
	writer *writer.Writer
}

type cfg struct {
	Level logging.Level

	// TODO: remove when config supports binding/sets
	levelSetting *config.Setting
}

// New returns a new logging module.Module
func New() module.Module {
	m := &loggingModule{
		cfg: &cfg{
			Level: logging.LevelInformational,
		},
	}

	m.writer = writer.New(
		os.Stdout,
		writer.Level(m.cfg.Level),
	)

	m.cfg.levelSetting = &config.Setting{
		Name:         "Level",
		Path:         "Logging",
		Description:  "Logging level to output",
		DefaultValue: m.cfg.Level.String(),
		Value:        &m.cfg.Level,
	}

	logging.DefaultLog = logging.New(
		logging.WithWriter(m.writer),
	)

	// watch and update the current log level
	m.cfg.levelSetting.Notify(config.NotifyFunc(func(s *config.Setting) {
		writer.Level(m.cfg.Level)(m.writer)
	}))

	// TODO: Remove once binding is available
	// make it a flag -level
	m.cfg.levelSetting.Flag("level", flag.CommandLine)

	return m
}

func (m loggingModule) Start(ctx context.Context) error {
	return nil
}

func (m loggingModule) Stop(ctx context.Context) error {
	return nil
}
