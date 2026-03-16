package protocol

import (
	"encoding/binary"
	"fmt"

	"github.com/google/uuid"
)

// 验证魔数
func ValidateMagic(bytes []byte) (uint32, error) {
	if len(bytes) != 4 {
		return 0, NewInvalidFrameLength(fmt.Sprintf("magic validation requires 4 bytes, got %d", len(bytes)))
	}
	magic := binary.BigEndian.Uint32(bytes)
	if magic != MAGIC {
		return 0, NewInvalidMagic(magic, MAGIC)
	}
	return magic, nil
}

// 验证版本号
func ValidateVersion(b byte) (uint8, error) {
	if b != VERSION {
		return 0, NewInvalidVersion(b, VERSION)
	}
	return b, nil
}

// MessageType 消息类型定义
type MessageType uint8

const (
	MessageTypeHeader MessageType = 0x01 // 连接建立时的元数据交换
	MessageTypeBody   MessageType = 0x02 // 业务数据载荷
	MessageTypePing   MessageType = 0x03 // 心跳探测
	MessageTypePong   MessageType = 0x04 // 心跳响应
)

func (mt MessageType) AsU8() uint8 {
	return uint8(mt)
}

func MessageTypeFromU8(value uint8) (MessageType, error) {
	switch value {
	case 0x01:
		return MessageTypeHeader, nil
	case 0x02:
		return MessageTypeBody, nil
	case 0x03:
		return MessageTypePing, nil
	case 0x04:
		return MessageTypePong, nil
	default:
		return 0, NewUnknownMessageType(value)
	}
}

func (mt MessageType) String() string {
	switch mt {
	case MessageTypeHeader:
		return "HEADER"
	case MessageTypeBody:
		return "Body"
	case MessageTypePing:
		return "PING"
	case MessageTypePong:
		return "PONG"
	default:
		return fmt.Sprintf("Unknown(0x%02X)", uint8(mt))
	}
}

// MessageFlags 消息标志位
type MessageFlags uint8

const (
	FlagEndStream  uint8 = 0b1000_0000 // bit 7
	FlagEndChannel uint8 = 0b0100_0000 // bit 6
	FlagIsResponse uint8 = 0b0010_0000 // bit 5 (0=请求, 1=响应)
	FlagReserved   uint8 = 0b0001_1111 // bit 4-0 保留位
)

func NewMessageFlags(endStream, endChannel, isResponse bool) MessageFlags {
	var val uint8 = 0
	if endStream {
		val |= FlagEndStream
	}
	if endChannel {
		val |= FlagEndChannel
	}
	if isResponse {
		val |= FlagIsResponse
	}
	return MessageFlags(val)
}

func (mf MessageFlags) AsU8() uint8 {
	return uint8(mf)
}

func (mf MessageFlags) EndStream() bool {
	return (uint8(mf) & FlagEndStream) != 0
}

func (mf MessageFlags) EndChannel() bool {
	return (uint8(mf) & FlagEndChannel) != 0
}

func (mf MessageFlags) IsResponse() bool {
	return (uint8(mf) & FlagIsResponse) != 0
}

func (mf MessageFlags) IsRequest() bool {
	return !mf.IsResponse()
}

func (mf MessageFlags) IsValid() bool {
	return (uint8(mf) & FlagReserved) == 0
}

func MessageFlagsFromU8(value uint8) (MessageFlags, error) {
	if (value & FlagReserved) != 0 {
		return 0, NewInvalidFlags(value)
	}
	return MessageFlags(value), nil
}

func (mf MessageFlags) String() string {
	return fmt.Sprintf("[END_STREAM=%v, END_CHANNEL=%v, IS_RESPONSE=%v]",
		mf.EndStream(), mf.EndChannel(), mf.IsResponse())
}

func (mf MessageFlags) ToTuple() (bool, bool, bool) {
	return mf.EndStream(), mf.EndChannel(), mf.IsResponse()
}

// ChannelId 通道ID封装
type ChannelId [16]byte

func NewChannelId() ChannelId {
	u := uuid.New()
	var cid ChannelId
	copy(cid[:], u[:])
	return cid
}

func ChannelIdFromUUID(u uuid.UUID) ChannelId {
	var cid ChannelId
	copy(cid[:], u[:])
	return cid
}

func (c ChannelId) AsUUID() uuid.UUID {
	return uuid.UUID(c)
}

func (c ChannelId) AsBytes() []byte {
	return c[:]
}

func (c ChannelId) IntoBytes() [16]byte {
	return c
}

func ChannelIdFromBytes(bytes [16]byte) ChannelId {
	return bytes
}

func ChannelIdTryFromSlice(slice []byte) (ChannelId, error) {
	if len(slice) != 16 {
		return ChannelId{}, NewInvalidUuidLength(16, len(slice))
	}
	var arr [16]byte
	copy(arr[:], slice)
	return arr, nil
}

// HashU64 获取UUID后8字节作为u64（用于一致性哈希）
func (c ChannelId) HashU64() uint64 {
	b := c[:]
	return binary.BigEndian.Uint64(b[8:])
}

// AssignTarget 标准分配：后8字节对总数取模
func (c ChannelId) AssignTarget(totalTargets int) (int, error) {
	if totalTargets == 0 {
		return 0, NewInvalidTargetCount(0, "must be greater than 0")
	}
	return int(c.HashU64() % uint64(totalTargets)), nil
}

// AssignTargetFast 超快分配：位运算版本（目标数必须是2的幂）
func (c ChannelId) AssignTargetFast(totalTargets int) (int, error) {
	if totalTargets == 0 || (totalTargets&(totalTargets-1)) != 0 {
		return 0, NewInvalidTargetCount(totalTargets, "must be power of 2 for fast path (e.g., 16, 32, 64, 128)")
	}
	return int(c.HashU64() & uint64(totalTargets-1)), nil
}

// MessageHeader 消息头部结构
type MessageHeader struct {
	ChannelId    ChannelId
	Source       string
	Target       string
	TotalLen     int
	StreamOffset int
}

// 传递数据帧的结构体
type Frame struct {
	MessageType   MessageType
	MessageFlags  MessageFlags
	MessageHeader MessageHeader
	Data          []byte
}
