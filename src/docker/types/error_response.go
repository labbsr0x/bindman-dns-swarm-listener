package types

// ErrorResponse Represents an error.
type ErrorResponse struct {

	// The error message.
	// Required: true
	Message string `json:"message"`
}
