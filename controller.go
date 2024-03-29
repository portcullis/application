package application

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Controller for modules
type Controller struct {
	modules map[string]*moduleReference
	orderer *int64
	once    sync.Once
	logger  *slog.Logger
}

type moduleReference struct {
	name           string
	started        bool
	order          int64
	implementation Module
}

func (c *Controller) init() {
	c.modules = make(map[string]*moduleReference)
	c.orderer = new(int64)
	if c.logger == nil {
		c.logger = slog.Default()
	}
}

// Add the specified module, if a module with the same name exists, it will be overwritten
func (c *Controller) Add(name string, m Module) {
	c.once.Do(c.init)

	if name == "" {
		panic("name must not be empty")
	}

	// if the module is nil, we don't add it, this way constructors/functions can return nil to disable adding
	if isNil(m) {
		return
	}

	c.modules[name] = &moduleReference{
		name:           name,
		implementation: m,
		order:          atomic.AddInt64(c.orderer, 1),
	}
}

// Remove the specified module
func (c *Controller) Remove(name string) {
	c.once.Do(c.init)

	if name == "" {
		return
	}

	delete(c.modules, name)
}

// Get the module with the specified name, will return nil if no module is found
func (c *Controller) Get(name string) Module {
	c.once.Do(c.init)

	if name == "" {
		return nil
	}

	ref, found := c.modules[name]
	if !found {
		return nil
	}

	return ref.implementation
}

// Range over the modules
func (c *Controller) Range(cb func(name string, module Module) bool) {
	sorted := sortModules(c.modules)

	for _, m := range sorted {
		if !cb(m.name, m.implementation) {
			break
		}
	}
}

// Run the added modules. This will run the lifetime on modules in the order they were added
//
// Module lifetime is called in the following order:
// * if module is Initializer -> Initialize()
// * if module is PreStarter -> PreStart()
// * Start()
// * if module is PostStarter -> PostStart()
// * wait for context.Done()
// * Stop()
//
// Stop() will be called on all module that Start() was successfully called on, even during error
func (c *Controller) Run(ctx context.Context) error {
	c.once.Do(c.init)

	if ctx == nil {
		ctx = context.Background()
	}

	exitErr := new(Error)

	sts := time.Now()
	c.logger.Debug("Module controller intializations starting")

	// build a list of modules so we can run them in the correct ordering (as added)
	runModules := sortModules(c.modules)
	for _, rm := range runModules {
		if initializer, ok := rm.implementation.(Initializer); ok {
			ts := time.Now()
			c.logger.Debug("Initializing module", "module", rm.name)
			itx, err := initializer.Initialize(ctx)
			if err != nil {
				exitErr = exitErr.Append(fmt.Errorf("failed to initialize module %q: %w", rm.name, err))
				return exitErr.Err()
			}

			if itx != nil {
				ctx = itx
			}
			c.logger.Debug("Initialized module", "module", rm.name, "duration", time.Since(ts))
		}
	}

	// account for any modules added in Initialize
	runModules = sortModules(c.modules)
	for _, rm := range runModules {
		if installer, ok := rm.implementation.(Installer); ok {
			ts := time.Now()
			c.logger.Debug("Installing module", "module", rm.name)
			err := installer.Install(ctx)
			if err != nil {
				exitErr = exitErr.Append(fmt.Errorf("failed to install module %q: %w", rm.name, err))
				goto shutdown
			}
			c.logger.Debug("Installed module", "module", rm.name, "duration", time.Since(ts))
		}
	}

	// account for any modules added in Initialize
	runModules = sortModules(c.modules)
	for _, rm := range runModules {
		if prestarter, ok := rm.implementation.(PreStarter); ok {
			ts := time.Now()
			c.logger.Debug("PreStarting module", "module", rm.name)
			if err := prestarter.PreStart(ctx); err != nil {
				exitErr = exitErr.Append(fmt.Errorf("failed to prestart module %q: %w", rm.name, err))
				goto shutdown
			}
			c.logger.Debug("PreStarted module", "module", rm.name, "duration", time.Since(ts))
		}
	}

	// account for any modules added in PreStart
	runModules = sortModules(c.modules)
	for _, rm := range runModules {
		ts := time.Now()
		c.logger.Debug("Starting module", "module", rm.name)
		if err := rm.implementation.Start(ctx); err != nil {
			exitErr = exitErr.Append(fmt.Errorf("failed to start module %q: %w", rm.name, err))
			goto shutdown
		}
		rm.started = true
		c.logger.Debug("Started module", "module", rm.name, "duration", time.Since(ts))
	}

	for _, rm := range runModules {
		if poststarter, ok := rm.implementation.(PostStarter); ok {
			ts := time.Now()
			c.logger.Debug("PostStarting module", "module", rm.name)
			if err := poststarter.PostStart(ctx); err != nil {
				exitErr = exitErr.Append(fmt.Errorf("failed to poststart module %q: %w", rm.name, err))
				goto shutdown
			}
			c.logger.Debug("PostStarted module", "module", rm.name, "duration", time.Since(ts))
		}
	}

	c.logger.Debug("Module controller intializations completed", "duration", time.Since(sts))

	if ctx.Done() != nil {
		<-ctx.Done()
	}

shutdown:
	sts = time.Now()
	c.logger.Debug("Module controller teardown starting")

	// reverse them
	for i, j := 0, len(runModules)-1; i < j; i, j = i+1, j-1 {
		runModules[i], runModules[j] = runModules[j], runModules[i]
	}

	for _, rm := range runModules {
		// only call stop on started modules
		if !rm.started {
			continue
		}

		ts := time.Now()
		c.logger.Debug("Stopping module", "module", rm.name)
		rm.started = false
		if err := rm.implementation.Stop(ctx); err != nil {
			exitErr = exitErr.Append(fmt.Errorf("failed to stop module %q: %w", rm.name, err))
		}
		c.logger.Debug("Stopped module", "module", rm.name, "duration", time.Since(ts))
	}

	if ctx.Err() != context.Canceled {
		exitErr = exitErr.Append(ctx.Err())
	}

	c.logger.Debug("Module controller teardown completed", "duration", time.Since(sts))

	return exitErr.Err()
}

func sortModules(modules map[string]*moduleReference) []*moduleReference {
	current := 0
	mods := make([]*moduleReference, len(modules))

	for _, mr := range modules {
		mods[current] = mr
		current++
	}

	sort.Slice(mods, func(i, j int) bool {
		return mods[i].order < mods[j].order
	})

	return mods
}

// need this in order to deal with the private type returns of nil
//
// example being:
//
//	type module struct {}
//
//	func New() *module { return nil }
//
// The above code will pass a check for if result == nil {}
//
// Credit: https://medium.com/@mangatmodi/go-check-nil-interface-the-right-way-d142776edef1
func isNil(i interface{}) bool {
	if i == nil {
		return true
	}

	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}

	return false
}
