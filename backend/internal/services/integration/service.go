package integration

import "github.com/sirupsen/logrus"

// Dependencies aggregates shared dependencies for integration services.
type Dependencies struct {
	Logger *logrus.Entry
}

// NewDependencies constructs a new dependency container with sane defaults.
func NewDependencies(logger *logrus.Entry) *Dependencies {
	return &Dependencies{Logger: logger}
}
