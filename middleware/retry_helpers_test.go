package middleware

import "time"

func alwaysRetryable() func(_ error) bool {
	return func(_ error) bool { return true }
}

func retryConfigBasic() RetryConfig {
	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.IsRetryable = alwaysRetryable()

	return config
}

func retryConfigFast() RetryConfig {
	config := DefaultRetryConfig()
	config.MaxAttempts = 3
	config.InitialDelay = time.Millisecond
	config.IsRetryable = alwaysRetryable()

	return config
}

func retryConfigSlow() RetryConfig {
	config := DefaultRetryConfig()
	config.MaxAttempts = 5
	config.InitialDelay = 50 * time.Millisecond
	config.IsRetryable = alwaysRetryable()

	return config
}
