package database

import "qingzhi/enterprise-api/internal/platform/redact"

// ConnectionError is a dependency failure that is safe to return to callers.
// Diagnostic is redacted and is only for server logs.
type ConnectionError struct {
	Dependency string
	diagnostic string
}

func (e *ConnectionError) Error() string {
	if e.Dependency == "" {
		return "dependency unavailable"
	}
	return e.Dependency + " unavailable"
}

// Diagnostic returns the redacted driver message for logs.
func (e *ConnectionError) Diagnostic() string {
	return e.diagnostic
}

// Unavailable wraps a driver error. The returned error text names only the dependency.
func Unavailable(dependency string, err error, secrets ...string) error {
	return unavailable(dependency, err, secrets...)
}

func unavailable(dependency string, err error, secrets ...string) error {
	if err == nil {
		return nil
	}
	return &ConnectionError{
		Dependency: dependency,
		diagnostic: redact.Apply(err.Error(), secrets...),
	}
}
