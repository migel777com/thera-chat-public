package models

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e ErrorResponse) Error() string {
	return e.Message
}

type AdvancedErrorResponse struct {
	Key     string `json:"-"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e AdvancedErrorResponse) Error() string {
	return e.Message
}
