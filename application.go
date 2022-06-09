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
	errorCh       chan error
}

// Run creates an application with the specified name and version, applies the provided options, and begins execution
func Run(name, version string, opts ...Option) error {
	return New(name, version, opts...).Run(context.Background())
}

// New creates a new application and executes the options
func New(name, version string, opts ...Option) *Application {
	app := &Application{
		Name:       name,
		Version:    version,
		Controller: &Controller{},
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

// Validate will validate the configuration and return any errors
func (a *Application) Validate(ctx context.Context) error {
	a.Lock()
	defer a.Unlock()

	return a.loadConfig(a.initialize(ctx))
}

// Install will execute all modules that have an application.Initializer implementation, then all modules with that implement the application.Installer
func (a *Application) Install(ctx context.Context) error {
	a.Lock()
	defer a.Unlock()

	ctx = a.initialize(ctx)
	if err := a.loadConfig(ctx); err != nil {
		return err
	}

	initializeModules := sortModules(a.Controller.modules)
	for _, im := range initializeModules {
		if initializer, ok := im.implementation.(Initializer); ok {
			itx, err := initializer.Initialize(ctx)
			if err != nil {
				return errors.Wrapf(err, "failed to initialize module %q", im.name)
			}

			if itx != nil {
				ctx = itx
			}
		}
	}

	installModules := sortModules(a.Controller.modules)
	for _, im := range installModules {
		installer, ok := im.implementation.(Installer)
		if !ok {
			continue
		}

		if err := installer.Install(ctx); err != nil {
			return errors.Wrapf(err, "failed to install module %q", im.name)
		}
	}

	return nil
}

// Run the application returning the error that terminated execution or nil if terminated normally
func (a *Application) Run(ctx context.Context) error {
	a.Lock()
	defer a.Unlock()

	ctx = a.initialize(ctx)
	if err := a.loadConfig(ctx); err != nil {
		return err
	}

	a.errorCh = make(chan error, 1)
	defer func() { close(a.errorCh); a.errorCh = nil }()

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
		case err := <-a.errorCh:
			cancel()

			// we wait for the controller to stop running so we can set the error
			wg.Wait()

			applicationError = (&Error{Errors: []error{err}}).Append(applicationError)

		case <-cancelCtx.Done():
			break APP_RUN

		case sig := <-schan:
			logging.Debug("Signal %v received", sig)
			if sig == syscall.SIGHUP || sig == syscall.Signal(21) {
				logging.Warning("TODO: Implement application reload hooks")
				// TODO: implement reload
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

// Exit will shutdown the application with the specified error.
//
// This call can be made from any go routine, only the first call to Exit will be read (first in) and shutdown the application
func (a *Application) Exit(err error) {
	if a.errorCh == nil || err == nil {
		return
	}

	a.errorCh <- err
}

func (a *Application) initialize(ctx context.Context) context.Context {
	if a.Controller == nil {
		a.Controller = &Controller{}
	}

	if a.Settings == nil {
		// use whatever is configured in the context
		a.Settings = config.FromContext(ctx)
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

	return ctx
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
