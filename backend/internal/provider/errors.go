package provider

const (
	ErrCodeProviderTimeout      = "PROVIDER_TIMEOUT"
	ErrCodeProviderUnavailable  = "PROVIDER_UNAVAILABLE"
	ErrCodeProviderAuthFailed   = "PROVIDER_AUTH_FAILED"
	ErrCodeNodeOffline          = "NODE_OFFLINE"
	ErrCodeResourceExhausted    = "RESOURCE_EXHAUSTED"
	ErrCodeImageNotFound        = "IMAGE_NOT_FOUND"
	ErrCodeInstanceNotFound     = "INSTANCE_NOT_FOUND"
	ErrCodeInstanceAlreadyExist = "INSTANCE_ALREADY_EXISTS"
	ErrCodePortExhausted        = "PORT_EXHAUSTED"
	ErrCodeNetworkError         = "NETWORK_ERROR"
	ErrCodeUnsupportedOperation = "UNSUPPORTED_OPERATION"
	ErrCodeUnknown              = "UNKNOWN_PROVIDER_ERROR"
)

func NewProviderError(code, provider, rawCode, rawMsg string, retryable bool, cause error) *Error {
	return &Error{
		Code:       code,
		Provider:   provider,
		RawCode:    rawCode,
		RawMessage: rawMsg,
		Retryable:  retryable,
		Cause:      cause,
	}
}
