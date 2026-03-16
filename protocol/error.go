package protocol

import (
	"fmt"
)

// ProtocolError 协议错误类型
type ProtocolError struct {
	Code    ErrorCode
	Message string
	Details map[string]any
}

type ErrorCode int

const (
	ErrErrorCode ErrorCode = iota
	ErrUnknownMessageType
	ErrInvalidFlags
	ErrInvalidMagic
	ErrInvalidVersion
	ErrInvalidFrameLength
	ErrIncompleteFrame
	ErrNameTooLong
	ErrStreamTooLong
	ErrLengthOverflow
	ErrInvalidOffset
	ErrCrcMismatch
	ErrMalformedFrame
	ErrEmptyName
	ErrInvalidUuidLength
	ErrInvalidUuidFormat
	ErrInvalidTargetCount
	ErrInvalidName
	ErrInvalidProtocol
	ErrEmptyAddress
	ErrInvalidPort
	ErrInvalidUtf8
	ErrMissingUuid
	ErrMissingName
	ErrMissingPassword
	ErrEmptyPassword
	ErrInvalidJson
	ErrInvalidHeaderType
	ErrInvalidContentType
	ErrMissingPath
	ErrMissingState
	ErrUnexpectedBody
	ErrMissingBody
	ErrInvalidStateCode
)

func (e *ProtocolError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Details)
	}
	return e.Message
}

// 获取错误类型
func GetErrorCode(err error) ErrorCode {
	if err == nil {
		return ErrErrorCode
	}
	if protocolError, ok := err.(*ProtocolError); ok {
		return protocolError.Code
	}
	return ErrErrorCode
}

// 便捷构造函数
func NewUnknownMessageType(v uint8) error {
	return &ProtocolError{
		Code:    ErrUnknownMessageType,
		Message: fmt.Sprintf("unknown message type: 0x%02X", v),
		Details: map[string]any{"value": v},
	}
}

func NewInvalidFlags(v uint8) error {
	return &ProtocolError{
		Code:    ErrInvalidFlags,
		Message: fmt.Sprintf("invalid flags: 0x%02X (reserved bits must be zero)", v),
		Details: map[string]any{"value": v},
	}
}

func NewInvalidMagic(received, expected uint32) error {
	return &ProtocolError{
		Code:    ErrInvalidMagic,
		Message: fmt.Sprintf("magic mismatch: received 0x%08X, expected 0x%08X", received, expected),
		Details: map[string]any{"received": received, "expected": expected},
	}
}

func NewInvalidVersion(received, expected uint8) error {
	return &ProtocolError{
		Code:    ErrInvalidVersion,
		Message: fmt.Sprintf("version mismatch: received %d, expected %d", received, expected),
		Details: map[string]any{"received": received, "expected": expected},
	}
}

func NewInvalidFrameLength(msg string) error {
	return &ProtocolError{
		Code:    ErrInvalidFrameLength,
		Message: fmt.Sprintf("invalid frame length: %s", msg),
	}
}

func NewIncompleteFrame(required, actual int, context string) error {
	return &ProtocolError{
		Code:    ErrIncompleteFrame,
		Message: fmt.Sprintf("incomplete frame (%s): required %d bytes, got %d", context, required, actual),
		Details: map[string]any{"required": required, "actual": actual, "context": context},
	}
}

func NewNameTooLong(nameType string, length int, max uint16) error {
	return &ProtocolError{
		Code:    ErrNameTooLong,
		Message: fmt.Sprintf("%s name too long: %d bytes (max: %d)", nameType, length, max),
		Details: map[string]any{"type": nameType, "length": length, "max": max},
	}
}

func NewStreamTooLong(length int, max uint32) error {
	return &ProtocolError{
		Code:    ErrStreamTooLong,
		Message: fmt.Sprintf("stream too long: %d bytes (max: %d bytes)", length, max),
		Details: map[string]any{"length": length, "max": max},
	}
}

func NewLengthOverflow(context string) error {
	return &ProtocolError{
		Code:    ErrLengthOverflow,
		Message: fmt.Sprintf("arithmetic overflow in: %s", context),
		Details: map[string]any{"context": context},
	}
}

func NewInvalidOffset(offset, total uint32) error {
	return &ProtocolError{
		Code:    ErrInvalidOffset,
		Message: fmt.Sprintf("invalid offset: %d (total: %d)", offset, total),
		Details: map[string]any{"offset": offset, "total": total},
	}
}

func NewCrcMismatch(received, calculated uint16) error {
	return &ProtocolError{
		Code:    ErrCrcMismatch,
		Message: fmt.Sprintf("CRC mismatch: received 0x%04X, calculated 0x%04X", received, calculated),
		Details: map[string]any{"received": received, "calculated": calculated},
	}
}

func NewMalformedFrame(msg string) error {
	return &ProtocolError{
		Code:    ErrMalformedFrame,
		Message: fmt.Sprintf("malformed frame: %s", msg),
	}
}

func NewEmptyName(nameType string) error {
	return &ProtocolError{
		Code:    ErrEmptyName,
		Message: fmt.Sprintf("%s name cannot be empty", nameType),
		Details: map[string]any{"type": nameType},
	}
}

func NewInvalidUuidLength(expected, actual int) error {
	return &ProtocolError{
		Code:    ErrInvalidUuidLength,
		Message: fmt.Sprintf("invalid UUID length: expected %d bytes, got %d", expected, actual),
		Details: map[string]any{"expected": expected, "actual": actual},
	}
}

func NewInvalidUuidFormat(msg string) error {
	return &ProtocolError{
		Code:    ErrInvalidUuidFormat,
		Message: fmt.Sprintf("invalid UUID format: %s", msg),
	}
}

func NewInvalidTargetCount(received int, reason string) error {
	return &ProtocolError{
		Code:    ErrInvalidTargetCount,
		Message: fmt.Sprintf("invalid target count: %d (%s)", received, reason),
		Details: map[string]any{"received": received, "reason": reason},
	}
}

func NewInvalidName(nameType string) error {
	return &ProtocolError{
		Code:    ErrInvalidName,
		Message: fmt.Sprintf("%s name is invalid", nameType),
		Details: map[string]any{"type": nameType},
	}
}

func NewInvalidProtocol(url string) error {
	return &ProtocolError{
		Code:    ErrInvalidProtocol,
		Message: fmt.Sprintf("invalid protocol scheme: %s (expected fernq://)", url),
		Details: map[string]any{"url": url},
	}
}

func NewEmptyAddress() error {
	return &ProtocolError{
		Code:    ErrEmptyAddress,
		Message: "address is empty after fernq://",
	}
}

func NewInvalidPort(port string) error {
	return &ProtocolError{
		Code:    ErrInvalidPort,
		Message: fmt.Sprintf("invalid port number: %s", port),
		Details: map[string]any{"port": port},
	}
}

func NewInvalidUtf8() error {
	return &ProtocolError{
		Code:    ErrInvalidUtf8,
		Message: "payload is not valid UTF-8",
	}
}

func NewMissingUuid() error {
	return &ProtocolError{
		Code:    ErrMissingUuid,
		Message: "missing UUID in URL path",
	}
}

func NewMissingName() error {
	return &ProtocolError{
		Code:    ErrMissingName,
		Message: "missing room name (expected #name)",
	}
}

func NewMissingPassword() error {
	return &ProtocolError{
		Code:    ErrMissingPassword,
		Message: "missing room_pass in query string",
	}
}

func NewEmptyPassword() error {
	return &ProtocolError{
		Code:    ErrEmptyPassword,
		Message: "room_pass cannot be empty",
	}
}

func NewInvalidJson(msg string) error {
	return &ProtocolError{
		Code:    ErrInvalidJson,
		Message: fmt.Sprintf("json parse error: %s", msg),
	}
}

func NewInvalidHeaderType(ty string) error {
	return &ProtocolError{
		Code:    ErrInvalidHeaderType,
		Message: fmt.Sprintf("invalid header_type: %s (expected 'request' or 'response')", ty),
		Details: map[string]any{"type": ty},
	}
}

func NewInvalidContentType(ty string) error {
	return &ProtocolError{
		Code:    ErrInvalidContentType,
		Message: fmt.Sprintf("invalid content_type: %s (expected 'json', 'form', or 'form_flow')", ty),
		Details: map[string]any{"type": ty},
	}
}

func NewMissingPath() error {
	return &ProtocolError{
		Code:    ErrMissingPath,
		Message: "missing required field 'path' for request",
	}
}

func NewMissingState() error {
	return &ProtocolError{
		Code:    ErrMissingState,
		Message: "missing required field 'state' for response",
	}
}

func NewUnexpectedBody() error {
	return &ProtocolError{
		Code:    ErrUnexpectedBody,
		Message: "unexpected body field for form/form_flow content type",
	}
}

func NewMissingBody() error {
	return &ProtocolError{
		Code:    ErrMissingBody,
		Message: "missing required body field for string/json content type",
	}
}

func NewInvalidStateCode(code int64) error {
	return &ProtocolError{
		Code:    ErrInvalidStateCode,
		Message: fmt.Sprintf("invalid state code: %d (expected HTTP status code)", code),
		Details: map[string]any{"code": code},
	}
}
