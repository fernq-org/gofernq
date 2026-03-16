package client

// FernqConnError 错误类型
type FernqConnError struct {
	Code    ErrorCode
	Message string
	Details map[string]any
}

type ErrorCode int

const (
	ErrErrorCode ErrorCode = iota
	ErrConnectFailed
	ErrVerifyFailed
	ErrAlreadyConnected
)

func (e *FernqConnError) Error() string {
	return e.Message
}

// 获取错误类型
func GetErrorCode(err error) ErrorCode {
	if err == nil {
		return ErrErrorCode
	}
	if protocolError, ok := err.(*FernqConnError); ok {
		return protocolError.Code
	}
	return ErrErrorCode
}

// NewConnectFailed 创建一个连接失败错误
func NewConnectFailed(message string) *FernqConnError {
	return &FernqConnError{
		Code:    ErrConnectFailed,
		Message: message,
	}
}

// NewVerifyFailed 创建一个验证失败错误
func NewVerifyFailed(message string) *FernqConnError {
	return &FernqConnError{
		Code:    ErrVerifyFailed,
		Message: message,
	}
}

// NewAlreadyConnected 创建一个已经连接的错误
func NewAlreadyConnected() *FernqConnError {
	return &FernqConnError{
		Code:    ErrAlreadyConnected,
		Message: "the client is already connected",
	}
}
