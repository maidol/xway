package error

import (
	"encoding/json"
	"net/http"

	"xway/enum"
)

// Error ...
type Error struct {
	ReturnCode int    `json:"ret"`
	ErrorCode  int    `json:"errorCode"`
	Message    string `json:"msg"`
	Cause      string `json:"cause,omitempty"`
	Index      uint64 `json:"index"`
}

// NewRequestError ...
func NewRequestError(returnCode, errorCode int, cause string) *Error {
	return NewError(returnCode, errorCode, cause, 0)
}

// NewError ...
func NewError(returnCode, errorCode int, cause string, index uint64) *Error {
	cm := enum.RetMsg[returnCode] + ": " + cause
	return &Error{
		ReturnCode: returnCode,
		ErrorCode:  errorCode,
		Message:    enum.CodeMsg[errorCode],
		Cause:      cm,
		Index:      index,
	}
}

// Error for the error interface
func (e Error) Error() string {
	return e.Message + " (" + e.Cause + ")"
}

func (e Error) toJSONString() string {
	b, _ := json.Marshal(e)
	return string(b)
}

// StatusCode ...
func (e Error) StatusCode() int {
	status, ok := enum.CodeStatus[e.ErrorCode]
	if !ok {
		status = http.StatusBadRequest
	}
	return status
}

func (e Error) Write(w http.ResponseWriter) error {
	// w.Header().Add("Content-Type", "application/json")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.StatusCode())
	_, err := w.Write([]byte(e.toJSONString() + "\n"))
	return err
}
