package application

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/portcullis/logging"
	"github.com/portcullis/module"
)

// Application defines an instance of an application
type Application struct {
	sync.Mutex

	Name    string
	Version string

	Controller *module.Controller

	// inner context if set
	innerContext context.Context
}

// Run creates an application with the specified name and version and applies the provided Option and begins execution
func Run(ctx context.Context, name, version string, opts ...Option) error {
	app := &Application{
		Name:       name,
		Version:    version,
		Controller: &module.Controller{},
	}

	for _, opt := range opts {
		opt(app)
	}

	return app.Run(ctx)
}

// Run the application returning the error that terminated execution
func (a *Application) Run(ctx context.Context) error {
	a.Lock()
	defer a.Unlock()

	// create one if it doesn't exist
	if a.Controller == nil {
		a.Controller = &module.Controller{}
	}

	// set the context for this run
	a.innerContext = ctx

	// listen to OS signals
	schan := make(chan os.Signal, 1)
	defer close(schan)

	signal.Notify(schan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(schan)

	// create a cancellation for the signals
	cancelCtx, cancel := context.WithCancel(a)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	var applicationError error
	go func() {
		// run the application in the background
		applicationError = a.Controller.Run(context.WithValue(cancelCtx, applicationContextKey, a))
		wg.Done()
		cancel()
	}()

	// wait for exit scenarios
	select {
	case <-ctx.Done():

	case sig := <-schan:
		logging.Info("Signal %v received", sig)
		cancel()
	}

	wg.Wait()

	// unset the context for the run, we are complete
	a.innerContext = nil

	if applicationError != nil && applicationError != context.Canceled {
		return applicationError
	}

	return nil
}
