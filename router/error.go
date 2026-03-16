package router

import (
	"fmt"
)

// RouterError 路由错误类型
type RouterError struct {
	Code    ErrorCode
	Message string
	Details map[string]any
}

type ErrorCode int

const (
	ErrNilRequest ErrorCode = iota
	ErrInvalidContentType
	ErrNilBody
	ErrBindFailed
	ErrUnsupportedContentType
	ErrAlreadyBound
	ErrEmptyPath
	ErrPathNotAbsolute
	ErrRouteNotFound
	ErrInvalidProtocol
	ErrMissingTarget
	ErrRequestTooLarge
	ErrRequestTimeout
	AlreadyUsedAsIndex
)

// 错误信息
func (e *RouterError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Details)
	}
	return e.Message
}

// 便捷构造函数
func NewNilRequest() error {
	return &RouterError{
		Code:    ErrNilRequest,
		Message: "request is nil",
	}
}

// NewInvalidContentTypeForBinding 创建无效的媒体类型错误
func NewInvalidContentTypeForBinding(actual any) error {
	return &RouterError{
		Code:    ErrInvalidContentType,
		Message: "invalid content type for binding: expected json",
		Details: map[string]any{"actual": actual, "expected": "json"},
	}
}

// NewNilBody 创建空请求体错误
func NewNilBody() error {
	return &RouterError{
		Code:    ErrNilBody,
		Message: "request body is nil",
	}
}

// NewBindFailed 创建绑定失败错误
func NewBindFailed(reason string) error {
	return &RouterError{
		Code:    ErrBindFailed,
		Message: fmt.Sprintf("failed to bind request body: %s", reason),
		Details: map[string]any{"reason": reason},
	}
}

// NewUnsupportedContentType 创建不支持的媒体类型错误
func NewUnsupportedContentType(contentType string) error {
	return &RouterError{
		Code:    ErrUnsupportedContentType,
		Message: fmt.Sprintf("unsupported content type: %s", contentType),
		Details: map[string]any{"content_type": contentType},
	}
}

// NewAlreadyBound 创建已绑定错误
func NewAlreadyBound(current BindType) error {
	return &RouterError{
		Code:    ErrAlreadyBound,
		Message: "context already bound, cannot bind again",
		Details: map[string]any{"current_bound": "json"},
	}
}

// NewEmptyPath 创建空路径错误
func NewEmptyPath() error {
	return &RouterError{
		Code:    ErrEmptyPath,
		Message: "path is empty",
	}
}

// NewPathNotAbsolute 创建路径格式错误
func NewPathNotAbsolute(path string) error {
	return &RouterError{
		Code:    ErrPathNotAbsolute,
		Message: fmt.Sprintf("path must start with '/': %s", path),
		Details: map[string]any{"path": path},
	}
}

// NewRouteNotFound 创建路由未找到错误
func NewRouteNotFound(path string) error {
	return &RouterError{
		Code:    ErrRouteNotFound,
		Message: fmt.Sprintf("route not found: %s", path),
		Details: map[string]any{"path": path},
	}
}

// NewInvalidProtocol 创建无效协议错误
func NewInvalidProtocol(actual string) error {
	return &RouterError{
		Code:    ErrInvalidProtocol,
		Message: "invalid protocol: expected fernq://",
		Details: map[string]any{"actual": actual},
	}
}

// NewMissingTarget 创建缺少目标错误
func NewMissingTarget() error {
	return &RouterError{
		Code:    ErrMissingTarget,
		Message: "missing target: expected host after protocol",
	}
}

// NewRequestTooLarge 创建请求过大错误
func NewRequestTooLarge(maxSize int) error {
	return &RouterError{
		Code:    ErrRequestTooLarge,
		Message: fmt.Sprintf("request too large: max size %d", maxSize),
		Details: map[string]any{"max_size": maxSize},
	}
}

// NewRequestTimeout 创建请求超时错误
func NewRequestTimeout() error {
	return &RouterError{
		Code:    ErrRequestTimeout,
		Message: "request timeout",
	}
}

// NewAlreadyUsedAsIndex 创建已使用为索引错误
func NewAlreadyUsedAsIndex() error {
	return &RouterError{
		Code:    AlreadyUsedAsIndex,
		Message: "the router's path already used as index",
	}
}
