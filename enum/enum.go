package enum

import "net/http"

// RetMsg ...
var RetMsg = map[int]string{
	RetNormal:     "正常",
	RetAbnormal:   "异常",
	RetProxyError: "代理相关异常",
}

// CodeMsg ...
var CodeMsg = map[int]string{
	CodeSuccessed:          "Success",
	ECodeRouteNotFound:     "未能成功匹配代理路由",
	ECodeProxyFailed:       "服务器错误",
	ECodeUnauthorized:      "未认证的请求",
	ECodeInternal:          "服务器内部错误",
	ECodeParamsError:       "请求参数错误",
	ECodeHmacsha1SignError: "hmacsha1签名错误",
	ECodeClientException:   "clientId错误或client status异常",
}

// CodeStatus ...
var CodeStatus = map[int]int{
	// CodeSuccessed:          http.StatusOK,
	// ECodeRouteNotFound:     http.StatusNotFound,
	// ECodeNotFile:           http.StatusForbidden,
	// ECodeDirNotEmpty:       http.StatusForbidden,
	// ECodeUnauthorized:      http.StatusUnauthorized,
	// ECodeTestFailed:        http.StatusPreconditionFailed,
	// ECodeProxyFailed:       http.StatusBadGateway,
	// ECodeInternal:          http.StatusInternalServerError,
	// ECodeParamsError:       http.StatusBadRequest,
	// ECodeHmacsha1SignError: http.StatusUnauthorized,
	// ECodeClientException:   http.StatusBadRequest,

	CodeSuccessed:      http.StatusOK,
	ECodeRouteNotFound: http.StatusNotFound,
	ECodeNotFile:       http.StatusForbidden,
	ECodeDirNotEmpty:   http.StatusForbidden,
	// ECodeUnauthorized:      http.StatusUnauthorized,
	ECodeUnauthorized:      http.StatusOK,
	ECodeTestFailed:        http.StatusPreconditionFailed,
	ECodeProxyFailed:       http.StatusBadGateway,
	ECodeInternal:          http.StatusInternalServerError,
	ECodeParamsError:       http.StatusOK,
	ECodeHmacsha1SignError: http.StatusOK,
	ECodeClientException:   http.StatusOK,
}

const (
	// RetNormal 正常返回
	RetNormal = 0
	// RetAbnormal 不正常返回
	RetAbnormal = 1

	// RetProxyError 代理相关错误
	RetProxyError = 5
)

const (
	// CodeSuccessed 成功
	CodeSuccessed = 0
	// CodeRouteNotFound 未匹配代理路由
	ECodeRouteNotFound = 100
	ECodeNotFile       = 101
	ECodeDirNotEmpty   = 102
	// ECodeUnauthorized      = 103
	ECodeUnauthorized      = 17
	ECodeTestFailed        = 104
	ECodeProxyFailed       = 105
	ECodeInternal          = 106
	ECodeParamsError       = 107
	ECodeHmacsha1SignError = 108
	ECodeClientException   = 109
)
