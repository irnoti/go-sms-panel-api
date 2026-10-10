package irnoti

import "fmt"

// APIError is returned for every non-2xx answer. Code is the API's machine-readable code
// ("" for plain 401/403/404 answers).
type APIError struct {
	Status  int
	Code    string
	Message string
	// Details holds invalid numbers, missing template params or per-field validation errors.
	Details any
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("irnoti: %s (HTTP %d, %s)", e.Message, e.Status, e.Code)
	}
	return fmt.Sprintf("irnoti: %s (HTTP %d)", e.Message, e.Status)
}

func (e *APIError) IsValidationError() bool {
	return e.Code == "validation_error" || e.Code == "invalid_recipients" || e.Code == "missing_params"
}
func (e *APIError) IsInvalidRecipients() bool { return e.Code == "invalid_recipients" }
func (e *APIError) IsInsufficientBalance() bool {
	return e.Status == 402 || e.Code == "insufficient_balance"
}
func (e *APIError) IsNoPrice() bool         { return e.Code == "no_price" }
func (e *APIError) IsRateLimited() bool     { return e.Status == 429 }
func (e *APIError) IsUnauthenticated() bool { return e.Status == 401 }
func (e *APIError) IsForbidden() bool       { return e.Status == 403 }
func (e *APIError) IsNotFound() bool        { return e.Status == 404 }
func (e *APIError) IsConflict() bool        { return e.Status == 409 }
func (e *APIError) IsServerError() bool     { return e.Status >= 500 }
