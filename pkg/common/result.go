package common

// Result is a generic container holding a value or an error.
type Result[T any] struct {
	Value T
	Err   error
}

// Ok creates a successful Result.
func Ok[T any](val T) Result[T] {
	return Result[T]{Value: val, Err: nil}
}

// Err creates a failed Result.
func Err[T any](err error) Result[T] {
	var zero T
	return Result[T]{Value: zero, Err: err}
}

// IsOk returns true if Result contains no error.
func (r Result[T]) IsOk() bool {
	return r.Err == nil
}

// Unwrap returns the value or panics if error.
func (r Result[T]) Unwrap() T {
	if r.Err != nil {
		panic(r.Err)
	}
	return r.Value
}
