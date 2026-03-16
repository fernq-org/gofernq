package protocol

import (
	"encoding/json"
	"strconv"
)

// HeaderType Header类型枚举
type HeaderType int

const (
	HeaderTypeRequest HeaderType = iota
	HeaderTypeResponse
)

func (ht HeaderType) AsStr() string {
	switch ht {
	case HeaderTypeRequest:
		return "request"
	case HeaderTypeResponse:
		return "response"
	default:
		return ""
	}
}

func (ht HeaderType) String() string {
	return ht.AsStr()
}

func HeaderTypeFromStr(s string) (HeaderType, error) {
	switch s {
	case "request":
		return HeaderTypeRequest, nil
	case "response":
		return HeaderTypeResponse, nil
	default:
		return 0, NewInvalidHeaderType(s)
	}
}

// ContentType Content类型枚举（简化版）
type ContentType int

const (
	ContentTypeJson     ContentType = iota // JSON数据，body可选
	ContentTypeForm                        // 表单数据，无body（流式传输，保留给未来实现）
	ContentTypeFormFlow                    // 表单流数据，无body（流式传输，保留给未来实现）
)

func (ct ContentType) AsStr() string {
	switch ct {
	case ContentTypeJson:
		return "json"
	case ContentTypeForm:
		return "form"
	case ContentTypeFormFlow:
		return "form_flow"
	default:
		return ""
	}
}

func (ct ContentType) String() string {
	return ct.AsStr()
}

// AllowsBody 是否允许body字段（只有Json允许，Form/FormFlow禁止）
func (ct ContentType) AllowsBody() bool {
	return ct == ContentTypeJson
}

func ContentTypeFromStr(s string) (ContentType, error) {
	switch s {
	case "json":
		return ContentTypeJson, nil
	case "form":
		return ContentTypeForm, nil
	case "form_flow":
		return ContentTypeFormFlow, nil
	default:
		return 0, NewInvalidContentType(s)
	}
}

// Message 解码后的消息结构
type Message interface {
	isMessage()
}

// RequestMessage 解码后的请求结构
type RequestMessage struct {
	ContentType ContentType
	Path        string
	Body        *json.RawMessage
}

func (r *RequestMessage) isMessage() {}

// ToBytes 将请求消息编码为字节
func (r *RequestMessage) ToBytes() ([]byte, error) {
	return EncodeRequestToBytes(r.Path, r.ContentType, r.Body)
}

// ResponseMessage 解码后的响应结构
type ResponseMessage struct {
	ContentType ContentType
	State       uint16
	Body        *json.RawMessage
}

func (r *ResponseMessage) isMessage() {}

// ToBytes 将响应消息编码为字节
func (r *ResponseMessage) ToBytes() ([]byte, error) {
	return EncodeResponseToBytes(r.State, r.ContentType, r.Body)
}

// NewRequestMessage 创建请求消息（无body）
func NewRequestMessage(path string, contentType ContentType) *RequestMessage {
	return &RequestMessage{
		ContentType: contentType,
		Path:        path,
		Body:        nil,
	}
}

// NewRequestMessageWithBody 创建请求消息，携带body（固定为Json类型，body可为nil）
func NewRequestMessageWithBody(path string, body any) (*RequestMessage, error) {
	// body为nil时，创建无body的Json请求
	if body == nil {
		return &RequestMessage{
			ContentType: ContentTypeJson,
			Path:        path,
			Body:        nil,
		}, nil
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, NewInvalidJson(err.Error())
	}
	rm := json.RawMessage(raw)
	return &RequestMessage{
		ContentType: ContentTypeJson,
		Path:        path,
		Body:        &rm,
	}, nil
}

// NewResponseMessage 创建响应消息（无body）
func NewResponseMessage(state uint16, contentType ContentType) *ResponseMessage {
	return &ResponseMessage{
		ContentType: contentType,
		State:       state,
		Body:        nil,
	}
}

// NewResponseMessageWithBody 创建响应消息，携带body（固定为Json类型，body可为nil）
func NewResponseMessageWithBody(state uint16, body any) (*ResponseMessage, error) {
	if state < 100 || state > 599 {
		return nil, NewInvalidStateCode(int64(state))
	}

	// body为nil时，创建无body的Json响应
	if body == nil {
		return &ResponseMessage{
			ContentType: ContentTypeJson,
			State:       state,
			Body:        nil,
		}, nil
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, NewInvalidJson(err.Error())
	}
	rm := json.RawMessage(raw)
	return &ResponseMessage{
		ContentType: ContentTypeJson,
		State:       state,
		Body:        &rm,
	}, nil
}

// EncodeRequest 编码请求消息
func EncodeRequest(path string, contentType ContentType, body *json.RawMessage) (map[string]any, error) {
	// 唯一约束：form/form_flow绝对不能有body
	if !contentType.AllowsBody() && body != nil {
		return nil, NewUnexpectedBody()
	}

	m := make(map[string]any)
	m["header_type"] = HeaderTypeRequest.AsStr()
	m["content_type"] = contentType.AsStr()
	m["path"] = path

	if body != nil {
		var v any
		if err := json.Unmarshal(*body, &v); err != nil {
			return nil, NewInvalidJson(err.Error())
		}
		m["body"] = v
	}

	return m, nil
}

// EncodeResponse 编码响应消息
func EncodeResponse(state uint16, contentType ContentType, body *json.RawMessage) (map[string]any, error) {
	if state < 100 || state > 599 {
		return nil, NewInvalidStateCode(int64(state))
	}

	// 唯一约束：form/form_flow绝对不能有body
	if !contentType.AllowsBody() && body != nil {
		return nil, NewUnexpectedBody()
	}

	m := make(map[string]any)
	m["header_type"] = HeaderTypeResponse.AsStr()
	m["content_type"] = contentType.AsStr()
	m["state"] = state

	if body != nil {
		var v any
		if err := json.Unmarshal(*body, &v); err != nil {
			return nil, NewInvalidJson(err.Error())
		}
		m["body"] = v
	}

	return m, nil
}

// EncodeRequestToBytes 编码请求为字节
func EncodeRequestToBytes(path string, contentType ContentType, body *json.RawMessage) ([]byte, error) {
	val, err := EncodeRequest(path, contentType, body)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(val)
	if err != nil {
		return nil, NewInvalidJson(err.Error())
	}
	return data, nil
}

// EncodeResponseToBytes 编码响应为字节
func EncodeResponseToBytes(state uint16, contentType ContentType, body *json.RawMessage) ([]byte, error) {
	val, err := EncodeResponse(state, contentType, body)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(val)
	if err != nil {
		return nil, NewInvalidJson(err.Error())
	}
	return data, nil
}

// DecodeMessage 统一解码函数
func DecodeMessage(value map[string]any) (Message, error) {
	// 获取 header_type
	headerTypeStr, ok := value["header_type"].(string)
	if !ok {
		return nil, NewMalformedFrame("missing or invalid header_type")
	}

	headerType, err := HeaderTypeFromStr(headerTypeStr)
	if err != nil {
		return nil, err
	}

	// 获取 content_type
	contentTypeStr, ok := value["content_type"].(string)
	if !ok {
		return nil, NewMalformedFrame("missing or invalid content_type")
	}

	contentType, err := ContentTypeFromStr(contentTypeStr)
	if err != nil {
		return nil, err
	}

	switch headerType {
	case HeaderTypeRequest:
		return decodeRequest(value, contentType)
	case HeaderTypeResponse:
		return decodeResponse(value, contentType)
	default:
		return nil, NewInvalidHeaderType(headerTypeStr)
	}
}

// DecodeFromBytes 从字节解码
func DecodeFromBytes(data []byte) (Message, error) {
	var val map[string]any
	if err := json.Unmarshal(data, &val); err != nil {
		return nil, NewInvalidJson(err.Error())
	}
	return DecodeMessage(val)
}

// ToJsonString 便捷函数：转为JSON字符串
func ToJsonString(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", NewInvalidJson(err.Error())
	}
	return string(data), nil
}

// FromJsonString 便捷函数：从JSON字符串解析
func FromJsonString(s string) (map[string]any, error) {
	var val map[string]any
	if err := json.Unmarshal([]byte(s), &val); err != nil {
		return nil, NewInvalidJson(err.Error())
	}
	return val, nil
}

// EncodeRequestToString 编码请求为JSON字符串
func EncodeRequestToString(path string, contentType ContentType, body *json.RawMessage) (string, error) {
	val, err := EncodeRequest(path, contentType, body)
	if err != nil {
		return "", err
	}
	return ToJsonString(val)
}

// EncodeResponseToString 编码响应为JSON字符串
func EncodeResponseToString(state uint16, contentType ContentType, body *json.RawMessage) (string, error) {
	val, err := EncodeResponse(state, contentType, body)
	if err != nil {
		return "", err
	}
	return ToJsonString(val)
}

// DecodeFromString 从JSON字符串解码
func DecodeFromString(s string) (Message, error) {
	val, err := FromJsonString(s)
	if err != nil {
		return nil, err
	}
	return DecodeMessage(val)
}

// decodeRequest 解码请求
func decodeRequest(obj map[string]any, contentType ContentType) (*RequestMessage, error) {
	pathVal, ok := obj["path"]
	if !ok {
		return nil, NewMissingPath()
	}
	path, ok := pathVal.(string)
	if !ok {
		return nil, NewMissingPath()
	}

	var body *json.RawMessage
	if bodyVal, exists := obj["body"]; exists {
		// 唯一约束：form/form_flow不能有body
		if !contentType.AllowsBody() {
			return nil, NewUnexpectedBody()
		}
		raw, err := json.Marshal(bodyVal)
		if err != nil {
			return nil, NewInvalidJson(err.Error())
		}
		rm := json.RawMessage(raw)
		body = &rm
	}

	return &RequestMessage{
		ContentType: contentType,
		Path:        path,
		Body:        body,
	}, nil
}

// decodeResponse 解码响应
func decodeResponse(obj map[string]any, contentType ContentType) (*ResponseMessage, error) {
	stateVal, ok := obj["state"]
	if !ok {
		return nil, NewMissingState()
	}

	var state uint16
	switch v := stateVal.(type) {
	case float64:
		state = uint16(v)
	case int:
		state = uint16(v)
	case string:
		s, err := strconv.Atoi(v)
		if err != nil {
			return nil, NewInvalidStateCode(int64(state))
		}
		state = uint16(s)
	default:
		return nil, NewMissingState()
	}

	if state < 100 || state > 599 {
		return nil, NewInvalidStateCode(int64(state))
	}

	var body *json.RawMessage
	if bodyVal, exists := obj["body"]; exists {
		// 唯一约束：form/form_flow不能有body
		if !contentType.AllowsBody() {
			return nil, NewUnexpectedBody()
		}
		raw, err := json.Marshal(bodyVal)
		if err != nil {
			return nil, NewInvalidJson(err.Error())
		}
		rm := json.RawMessage(raw)
		body = &rm
	}

	return &ResponseMessage{
		ContentType: contentType,
		State:       state,
		Body:        body,
	}, nil
}
