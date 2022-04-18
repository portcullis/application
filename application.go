package application

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/portcullis/config"
	"github.com/portcullis/logging"
)

// Application defines an instance of an application
type Application struct {
	sync.Mutex

	Name       string
	Version    string
	Controller *Controller
	Settings   *config.Set

	configuration *Configuration
	configFile    string
}

// Run creates an application with the specified name and version, applies the provided options, and begins execution
func Run(name, version string, opts ...Option) error {
	app := &Application{
		Name:       name,
		Version:    version,
		Controller: &Controller{},
		Settings:   &config.Set{},
	}

	for _, opt := range opts {
		opt(app)
	}

	return app.Run(context.Background())
}

// Run the application returning the error that terminated execution or nil if terminated normally
func (a *Application) Run(ctx context.Context) error {
	a.Lock()
	defer a.Unlock()

	// create one if it doesn't exist
	if a.Controller == nil {
		a.Controller = &Controller{}
	}

	if a.Settings == nil {
		a.Settings = &config.Set{}
	}

	ctx = context.WithValue(ctx, applicationContextKey, a)
	ctx = config.NewContext(ctx, a.Settings)

	// some defaults
	if a.Name == "" {
		a.Name = "Portcullis"
	}
	if a.Version == "" {
		a.Version = "0.0.0"
	}

	if err := a.loadConfig(ctx); err != nil {
		return err
	}

	startupTime := time.Now()
	logging.Info("Starting application %s", a)
	defer func() { logging.Info("Stopped application %s with runtime of %v", a, time.Since(startupTime)) }()

	// listen to OS signals
	schan := make(chan os.Signal, 6)
	defer close(schan)

	// wire these up early
	signal.Notify(schan, syscall.SIGTERM, syscall.SIGABRT, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGHUP, syscall.Signal(21))
	defer signal.Stop(schan)

	// create a cancellation for the signals
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var applicationError error

	var wg sync.WaitGroup
	wg.Add(1)

	// run the modules
	go func() {
		// run the application in the background
		applicationError = a.Controller.Run(cancelCtx)
		wg.Done()
		cancel()
	}()

	// wait for exit scenarios
APP_RUN:
	for {
		select {
		case <-cancelCtx.Done():
			break APP_RUN
		case sig := <-schan:
			logging.Debug("Signal %v received", sig)
			if sig == syscall.SIGHUP || sig == syscall.Signal(21) {
				logging.Warning("TODO: Implement application reload hooks")
				// implement reload
				break
			}

			cancel()
			break APP_RUN
		}
	}

	// wait for finalization of Controller shutdown
	wg.Wait()

	// this is a bit of a hacky thing, but allows us to not return errors for help command line and invalid commands so the top level caller can if err != nil panic(err)
	if applicationError != nil && applicationError != context.Canceled && !errors.Is(applicationError, flag.ErrHelp) && !strings.Contains(applicationError.Error(), "flag provided but not defined:") {
		return applicationError
	}

	return nil
}

func (a *Application) loadConfig(ctx context.Context) error {
	if a.configuration == nil {
		return nil
	}

	if a.configFile == "" {
		return nil
	}

	logging.Info("Loading configuration file %s", a.configFile)
	if diags := a.configuration.DecodeFile(ctx, a.configFile); diags.HasErrors() {
		return errors.Wrap(diags, "failed to load application configuration")
	}

	return nil
}
