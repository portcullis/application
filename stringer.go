package application

import "fmt"

// String returns the application name / application version
func (a *Application) String() string {
	return fmt.Sprintf("%s/%s", a.Name, a.Version)
}
