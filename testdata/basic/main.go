package basic

import (
	"context"
)

// ComplexInterface defines a complex interface for testing
type ComplexInterface interface {
	Method1(ctx context.Context) error
	Method2(s string, i int) (bool, error)
	Method3(data []byte) string
}

// User represents a user in the system
type User struct {
	ID   int
	Name string
}

// String implements fmt.Stringer
func (u *User) String() string {
	return u.Name
}

// Options represents configuration options
type Options struct {
	Debug bool
}

// Result represents a result type
type Result struct {
	Success bool
	Data    interface{}
}

// Method1 implements a basic method
func Method1(ctx context.Context) error {
	return nil
}

// Method2 implements a method with multiple parameters and results
func Method2(s string, i int) (bool, error) {
	return true, nil
}

// Method3 implements a method with byte slice parameter
func Method3(data []byte) string {
	return string(data)
}

// Method4 implements a method with complex parameters
func Method4(ctx context.Context, data map[string]interface{}, opts *Options) (*Result, error) {
	return &Result{Success: true}, nil
}

// Method5 implements a method with variadic parameters
func Method5(prefix string, values ...interface{}) (string, error) {
	return prefix, nil
}

// Method6 implements a method with channel parameters
func Method6(input chan string, output chan<- interface{}) error {
	return nil
}
