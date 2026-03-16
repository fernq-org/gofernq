package router

// StateCode fernq 状态码
type StateCode int16

const (
	// 2xx 成功
	StatusOK        StateCode = 200
	StatusCreated   StateCode = 201
	StatusAccepted  StateCode = 202
	StatusNoContent StateCode = 204

	// 4xx 客户端错误
	StatusBadRequest      StateCode = 400
	StatusUnauthorized    StateCode = 401
	StatusForbidden       StateCode = 403
	StatusNotFound        StateCode = 404
	StatusConflict        StateCode = 409
	StatusPayloadTooLarge StateCode = 413
	StatusTooManyRequests StateCode = 429

	// 5xx 服务端错误
	StatusInternalServerError StateCode = 500
	StatusBadGateway          StateCode = 502
	StatusServiceUnavailable  StateCode = 503
)
