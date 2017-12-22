package error

import (
	"encoding/json"
	"net/http"
)

var returnCode = map[int]int{
	Normal:   0,
	Abnormal: 1,
}

var errors = map[int]string{
	EcodeSuccessed:     "Success",
	EcodeRouteNotFound: "未匹配代理路由",
}

var errorStatus = map[int]int{
	EcodeSuccessed:     http.StatusOK,
	EcodeRouteNotFound: http.StatusNotFound,
	EcodeNotFile:       http.StatusForbidden,
	EcodeDirNotEmpty:   http.StatusForbidden,
	EcodeUnauthorized:  http.StatusUnauthorized,
	EcodeTeestFailed:   http.StatusPreconditionFailed,
	EcodeProxyFailed:   http.StatusBadGateway,
	EcodeInternal:      http.StatusInternalServerError,
}

const (
	// Normal 正常返回
	Normal = 0
	// Abnormal 不正常返回
	Abnormal = 1
)

const (
	// EcodeSuccessed 成功
	EcodeSuccessed = 0
	// EcodeRouteNotFound 未匹配代理路由
	EcodeRouteNotFound = 100
	EcodeNotFile       = 101
	EcodeDirNotEmpty   = 102
	EcodeUnauthorized  = 103
	EcodeTeestFailed   = 104
	EcodeProxyFailed   = 105
	EcodeInternal      = 106
)

type Error struct {
	ReturnCode int    `json:"ret"`
	ErrorCode  int    `json:"errorCode"`
	Message    string `json:"msg"`
	Cause      string `json:"cause,omitempty"`
	Index      uint64 `json:"index"`
}

func NewRequestError(returnCode, errorCode int, cause string) *Error {
	return NewError(returnCode, errorCode, cause, 0)
}

func NewError(returnCode, errorCode int, cause string, index uint64) *Error {
	return &Error{
		ReturnCode: returnCode,
		ErrorCode:  errorCode,
		Message:    errors[errorCode],
		Cause:      cause,
		Index:      index,
	}
}

// Error for the error interface
func (e Error) Error() string {
	return e.Message + " (" + e.Cause + ")"
}

func (e Error) toJsonString() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func (e Error) StatusCode() int {
	status, ok := errorStatus[e.ErrorCode]
	if !ok {
		status = http.StatusBadRequest
	}
	return status
}

func (e Error) Write(w http.ResponseWriter) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(e.StatusCode())
	_, err := w.Write([]byte(e.toJsonString() + "\n"))
	return err
}
