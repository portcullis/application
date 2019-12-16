package application

import (
	"context"
	"time"
)

type contextKey string

var (
	applicationContextKey = contextKey("application")
)

// FromContext extracts the Appliation instance if it exists from the provided context or nil if not found
func FromContext(ctx context.Context) *Application {
	app := ctx.Value(applicationContextKey)
	if app == nil {
		return nil
	}

	return app.(*Application)
}

// Deadline returns the time when work done on behalf of this context should be canceled.
func (a *Application) Deadline() (deadline time.Time, ok bool) {
	if a.innerContext == nil {
		return
	}

	return a.innerContext.Deadline()
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled. Done may return nil if this context can
// never be canceled.
func (a *Application) Done() <-chan struct{} {
	if a.innerContext == nil {
		return nil
	}

	return a.innerContext.Done()
}

// Err returns the error for the Application
func (a *Application) Err() error {
	if a.innerContext == nil {
		return nil
	}

	return a.innerContext.Err()
}

// Value returns the value associated with this context for key, or nil
func (a *Application) Value(key interface{}) interface{} {
	if a.innerContext == nil {
		return nil
	}

	return a.innerContext.Value(key)
}
