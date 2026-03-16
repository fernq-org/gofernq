package protocol

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// CreateValidate 验证并解析 fernq 协议的连接 URL，返回规范化后的地址和编码后的验证信息帧
//
// 该函数用于客户端发起连接请求时，验证目标服务器地址格式并生成初始握手帧。
// 生成的帧包含完整的 URL 信息，供服务端解析获取连接参数。
//
// 参数:
//
//   - name: 发送方标识名称，用于在消息帧中标识发送者身份
//
//   - channelId: 通道ID指针，用于消息帧的通道标识和路由
//
//   - urlStr: fernq 协议 URL，标准格式为:
//     fernq://<host>[:<port>]/<uuid>#<room_name>?room_pass=<password>
//
//     各组成部分:
//     ├─ fernq://          : 必需，协议方案名（scheme）
//     ├─ <host>[:<port>]   : 必需，服务器地址
//     │   ├─ IPv4          : 192.168.1.1
//     │   ├─ IPv6          : [2001:db8::1]（必须带方括号）
//     │   ├─ 域名          : example.com
//     │   └─ 端口          : :8080（可选，范围 1-65535）
//     ├─ /<uuid>           : 必需，房间/通道唯一标识符（从最后一个/提取）
//     ├─ #<room_name>      : 必需，房间显示名称（URL fragment）
//     └─ ?room_pass=<pwd>  : 必需，房间访问密码（查询参数）
//
//     有效示例:
//
//   - "fernq://192.168.1.1:8080/550e8400-e29b-41d4-a716-446655440000#lobby?room_pass=123456"
//
//   - "fernq://[::1]:9000/12345678-1234-1234-1234-123456789abc#room-1?room_pass=secret"
//
//   - "fernq://example.com/550e8400-e29b-41d4-a716-446655440000#test?room_pass=mypass"
//
//     注意: 虽然本函数不强制要求 uuid、room_name、room_pass 的存在，
//     但 ParseValidate 函数会要求这三项必须存在且格式正确。
//
// 返回值:
//   - string: 规范化后的服务器地址（host:port 或 host），可直接用于 TCP 连接
//     例如: "192.168.1.1:8080"、"[::1]:9000"、"example.com"
//   - []byte: 编码后的信息帧（MessageTypeHeader 类型），
//     包含原始 urlStr 作为 payload，用于发送给服务端验证
//   - error: 验证过程中发生的错误，具体类型包括:
//   - *InvalidProtocolError: URL 不以 "fernq://" 开头
//   - *EmptyAddressError: 地址部分为空（如 "fernq://"）
//   - *MalformedFrameError: 地址格式错误（如 IPv6 括号不匹配、]:port 格式错误）
//   - *InvalidPortError: 端口号无效（非数字或超出 0-65535 范围）
//   - *InvalidFrameLengthError: 生成的帧数量不为 1（数据过长被分割）
//
// 处理流程:
//  1. 【协议检查】验证 urlStr 以 "fernq://" 开头，否则返回 InvalidProtocolError
//  2. 【地址提取】提取协议头后到第一个 /?# 之前的部分作为地址
//  3. 【地址验证】根据格式选择验证方式:
//     - IPv6 格式（以 [ 开头）: 验证 [addr]:port 或 [addr] 格式，校验端口号
//     - IPv4/域名格式: 验证 host:port 或 host 格式，校验端口号
//  4. 【帧生成】调用 GenerateMessageDataStream 创建消息帧:
//     - 消息类型: MessageTypeHeader
//     - 是否为请求: true
//     - 是否为结束帧: false
//     - payload: 原始 urlStr 的 UTF-8 字节
//  5. 【帧校验】确保只生成 1 个帧（防止 URL 过长被分割）
//
// 与 ParseValidate 的对应关系:
//
//	CreateValidate (客户端)          ParseValidate (服务端)
//	─────────────────────────────────────────────────────────
//	输入: 完整 URL                    输入: 帧 payload（即 URL 字符串）
//	输出: 地址 + 帧                   输出: UUID + 房间名 + 密码
//	职责: 验证地址格式，打包成帧       职责: 解析 URL，提取连接参数
//
// 注意:
//   - 本函数仅验证【服务器地址部分】的格式正确性
//   - 不验证 UUID 格式、房间名非空、密码存在性（这些由 ParseValidate 验证）
//   - 完整 URL 会被原封不动地放入消息帧 payload 中传输
//   - 地址规范化后可用于 net.Dial 等标准库函数建立连接
//
// 并发安全:
//   - 本函数无共享状态，可安全并发调用
//
// 示例:
//
//	channelId := &ChannelId{Id: 1}
//	url := "fernq://127.0.0.1:8080/550e8400-e29b-41d4-a716-446655440000#my-room?room_pass=secret"
//
//	addr, frame, err := CreateValidate("client-1", channelId, url)
//	if err != nil {
//	    log.Fatalf("验证失败: %v", err)
//	}
//
//	// addr: "127.0.0.1:8080"（可直接用于 net.Dial("tcp", addr)）
//	// frame: 编码后的消息帧，应通过连接发送给服务端
//	conn.Write(frame)
func CreateValidate(name string, channelId *ChannelId, urlStr string) (string, []byte, error) {
	// 检查协议头
	const prefix = "fernq://"
	if !strings.HasPrefix(urlStr, prefix) {
		return "", nil, NewInvalidProtocol(urlStr)
	}

	// 去掉协议头，提取地址部分
	rest := urlStr[len(prefix):]
	addrEnd := strings.IndexAny(rest, "/?#")
	if addrEnd == -1 {
		addrEnd = len(rest)
	}
	addrPart := rest[:addrEnd]

	// 检查地址为空
	if addrPart == "" {
		return "", nil, NewEmptyAddress()
	}

	// 验证地址格式并规范化
	normalizedAddr, err := validateAndNormalizeAddress(addrPart)
	if err != nil {
		return "", nil, err
	}

	// 编码
	frames, err := GenerateMessageDataStream(
		MessageTypeHeader,
		true,
		false,
		channelId,
		name,
		"fernq",
		[]byte(urlStr),
	)
	if err != nil {
		return "", nil, err
	}

	// 检查帧生成结果，必须只有一个帧
	if len(frames) != 1 {
		return "", nil, NewInvalidFrameLength("frame too long")
	}

	frame := frames[0]
	return normalizedAddr, frame, nil
}

// ParseValidate 解析验证信息
func ParseValidate(payload []byte) (uuid.UUID, string, string, error) {
	// UTF-8 编码检查
	urlStr := string(payload)
	if urlStr == "" {
		return uuid.UUID{}, "", "", NewInvalidUtf8()
	}

	// 协议头检查
	if !strings.HasPrefix(urlStr, "fernq://") {
		return uuid.UUID{}, "", "", NewInvalidProtocol(urlStr)
	}

	// 提取密码
	urlWithoutQuery, password, err := extractPassword(urlStr)
	if err != nil {
		return uuid.UUID{}, "", "", err
	}

	// 提取房间名称
	pathPart, name, err := extractName(urlWithoutQuery)
	if err != nil {
		return uuid.UUID{}, "", "", err
	}

	// 提取并验证 UUID
	uuidVal, err := extractUuid(pathPart)
	if err != nil {
		return uuid.UUID{}, "", "", err
	}

	return uuidVal, name, password, nil
}

// CreateVerifyResponse 生成验证信息的响应帧
func CreateVerifyResponse(name string, channelId *ChannelId, state bool, message string) ([]byte, error) {
	jsonObj := fmt.Sprintf(`{"state":%t,"message":"%s"}`, state, message)

	frames, err := GenerateMessageDataStream(
		MessageTypeHeader,
		true,
		true,
		channelId,
		"fernq",
		name,
		[]byte(jsonObj),
	)
	if err != nil {
		return nil, err
	}

	if len(frames) != 1 {
		return nil, NewInvalidFrameLength("frame too long")
	}

	frame := frames[0]
	return frame, nil
}

// CreateMessageResponse 生成响应信息帧
func CreateMessageResponse(name string, channelId *ChannelId, state uint16, body string) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, NewInvalidJson(err.Error())
	}

	data, err := EncodeResponseToBytes(state, ContentTypeJson, (*json.RawMessage)(&jsonBody))
	if err != nil {
		return nil, err
	}

	frames, err := GenerateMessageDataStream(
		MessageTypeHeader,
		true,
		true,
		channelId,
		"fernq",
		name,
		data,
	)
	if err != nil {
		return nil, err
	}

	if len(frames) != 1 {
		return nil, NewInvalidFrameLength("frame too long")
	}

	frame := frames[0]
	return frame, nil
}

// ParseVerifyResponse 解析验证响应信息
func ParseVerifyResponse(data []byte) (bool, string, error) {
	jsonStr := string(data)

	state := gjson.Get(jsonStr, "state")
	if !state.Exists() {
		return false, "", NewInvalidJson("field 'state' must be boolean")
	}

	message := gjson.Get(jsonStr, "message")
	if !message.Exists() {
		return false, "", NewInvalidJson("field 'message' must be string")
	}

	return state.Bool(), message.String(), nil
}

// 内部辅助函数

func validateAndNormalizeAddress(addr string) (string, error) {
	// 检查IPv6格式 [addr]:port 或 [addr]
	if strings.HasPrefix(addr, "[") {
		return validateIPv6Address(addr)
	}
	// IPv4或域名格式
	return validateIPv4OrHostname(addr)
}

func validateIPv4OrHostname(addr string) (string, error) {
	if colonPos := strings.LastIndex(addr, ":"); colonPos != -1 {
		// 可能有端口
		host := addr[:colonPos]
		portStr := addr[colonPos+1:]

		// 验证端口是数字且在有效范围
		if _, err := strconv.ParseUint(portStr, 10, 16); err != nil {
			return "", NewInvalidPort(portStr)
		}

		if host == "" {
			return "", NewMalformedFrame("empty host before port")
		}
		return addr, nil
	}
	// 无端口，返回原地址
	return addr, nil
}

func validateIPv6Address(addr string) (string, error) {
	closeBracket := strings.Index(addr, "]")
	if closeBracket == -1 {
		return "", NewMalformedFrame("unclosed IPv6 bracket")
	}

	// 检查括号后是否有端口 :port
	if closeBracket+1 < len(addr) {
		if closeBracket+2 > len(addr) || addr[closeBracket+1] != ':' {
			return "", NewMalformedFrame("invalid IPv6 format, expected ]:port")
		}
		portStr := addr[closeBracket+2:]
		if _, err := strconv.ParseUint(portStr, 10, 16); err != nil {
			return "", NewInvalidPort(portStr)
		}
	}

	return addr, nil
}

func extractPassword(urlStr string) (string, string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", "", NewMissingPassword()
	}

	password := u.Query().Get("room_pass")
	if password == "" {
		return "", "", NewMissingPassword()
	}

	u.RawQuery = ""
	return u.String(), password, nil
}

func extractName(urlStr string) (string, string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", "", NewMissingName()
	}

	name := u.Fragment
	if name == "" {
		return "", "", NewMissingName()
	}

	u.Fragment = ""
	return u.String(), name, nil
}

func extractUuid(urlStr string) (uuid.UUID, error) {
	// 取最后一个 / 后面的内容作为 UUID
	lastSlash := strings.LastIndex(urlStr, "/")
	if lastSlash == -1 || lastSlash == len(urlStr)-1 {
		return uuid.UUID{}, NewMissingUuid()
	}

	uuidStr := urlStr[lastSlash+1:]
	if uuidStr == "" {
		return uuid.UUID{}, NewMissingUuid()
	}

	u, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.UUID{}, NewInvalidUuidFormat(fmt.Sprintf("'%s' is not a valid UUID: %v", uuidStr, err))
	}

	return u, nil
}
