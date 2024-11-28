package failurehandler

// FailureHandler is a function that can be used with Gomega to perform extra debugging when an assertion fails
// Note: Needs to be `interface{}` for Gomega to accept this alias type
type FailureHandler interface{}

// Wrap returns a valid FailureHandler for the given function
func Wrap(fn func()) FailureHandler {
	return func() string {
		fn()
		return ""
	}
}

// Bundle allows multiple different FailureHandler functions to be used in a chain
func Bundle(failureHandlers ...FailureHandler) FailureHandler {
	return Wrap(func() {
		for _, fn := range failureHandlers {
			fn.(func() string)()
		}
	})
}
