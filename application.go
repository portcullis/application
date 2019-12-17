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
}

// Run creates an application with the specified name and version and applies the provided Option and begins execution
func Run(name, version string, opts ...Option) error {
	app := &Application{
		Name:       name,
		Version:    version,
		Controller: &module.Controller{},
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
		a.Controller = &module.Controller{}
	}

	// listen to OS signals
	schan := make(chan os.Signal, 1)
	defer close(schan)

	// wire these up early
	signal.Notify(schan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(schan)

	// create a cancellation for the signals
	cancelCtx, cancel := context.WithCancel(ctx)
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
	case <-cancelCtx.Done():

	case sig := <-schan:
		// TODO: Handle all the signals explicitly, and a way to SIGHUP things
		logging.Info("Signal %v received", sig)
		cancel()
	}

	wg.Wait()

	if applicationError != nil && applicationError != context.Canceled {
		return applicationError
	}

	return nil
}
