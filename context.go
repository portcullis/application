package application

import (
	"context"
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
