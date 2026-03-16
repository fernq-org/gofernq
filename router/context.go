package router

import (
	"encoding/json"
	"sync"

	"github.com/fernq-org/gofernq/protocol"
)

// BindType 绑定类型标记
type BindType int

const (
	BindTypeNone BindType = iota
	BindTypeJSON
)

// Context 请求上下文
type Context struct {
	request        *protocol.RequestMessage
	response       *protocol.ResponseMessage
	forms          map[string]any
	messageChannel <-chan *protocol.Frame

	bindType BindType
	bindMu   sync.Mutex
}

// NewContext 创建请求上下文
func NewContext(messageChannel <-chan *protocol.Frame, request *protocol.RequestMessage) *Context {
	return &Context{
		request:        request,
		messageChannel: messageChannel,
		bindType:       BindTypeNone,
	}
}

// ShouldBindJSON 将请求体绑定到目标对象
//
// 限制:
//   - 只能调用一次，重复调用返回 ErrAlreadyBound
//   - 请求必须是 ContentTypeJson 且 Body 不为 nil
func (c *Context) ShouldBindJSON(obj any) error {
	if c.bindType != BindTypeNone {
		return NewAlreadyBound(c.bindType)
	}

	c.bindMu.Lock()
	defer c.bindMu.Unlock()

	if c.bindType != BindTypeNone {
		return NewAlreadyBound(c.bindType)
	}

	if c.request == nil {
		return NewNilRequest()
	}

	if c.request.ContentType != protocol.ContentTypeJson {
		return NewInvalidContentTypeForBinding(c.request.ContentType)
	}

	if c.request.Body == nil {
		return NewNilBody()
	}

	if err := json.Unmarshal(*c.request.Body, obj); err != nil {
		return NewBindFailed(err.Error())
	}

	c.bindType = BindTypeJSON
	return nil
}

// JSON 设置 JSON 响应（body 可为 nil）
func (c *Context) JSON(code StateCode, body any) error {
	var resp *protocol.ResponseMessage
	var err error

	if body == nil {
		resp = protocol.NewResponseMessage(uint16(code), protocol.ContentTypeJson)
	} else {
		resp, err = protocol.NewResponseMessageWithBody(uint16(code), body)
		if err != nil {
			return err
		}
	}

	c.response = resp
	return nil
}

// GetResponse 获取响应
func (c *Context) GetResponse() *protocol.ResponseMessage {
	return c.response
}
