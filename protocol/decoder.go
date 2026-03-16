package protocol

import (
	"encoding/binary"
)

// DecodeHeader 初级解码，只解码固定协议头
func DecodeHeader(bytes []byte) (MessageType, MessageFlags, int, error) {
	if len(bytes) < MAGIC_LEN {
		return 0, 0, 0, NewIncompleteFrame(MAGIC_LEN, len(bytes), "fixed header")
	}

	// 校验魔数
	if _, err := ValidateMagic(bytes[0:MAGIC_LEN]); err != nil {
		return 0, 0, 0, err
	}

	// 校验版本号
	if _, err := ValidateVersion(bytes[MAGIC_LEN]); err != nil {
		return 0, 0, 0, err
	}

	// 获取类型
	msgType, err := MessageTypeFromU8(bytes[MAGIC_LEN+VERSION_LEN])
	if err != nil {
		return 0, 0, 0, err
	}

	// 获取帧标志字段
	flags, err := MessageFlagsFromU8(bytes[MAGIC_LEN+VERSION_LEN+TYPE_LEN])
	if err != nil {
		return 0, 0, 0, err
	}

	// 获取帧长度字段（4字节，大端序）
	frameLength := int(binary.BigEndian.Uint32(bytes[MAGIC_LEN+VERSION_LEN+TYPE_LEN+FLAGS_LEN:]))

	return msgType, flags, frameLength, nil
}

// DecodeBasic 初级解码，解码完整帧并进行CRC校验
func DecodeBasic(bytes []byte) (MessageType, MessageFlags, []byte, []byte, error) {
	if len(bytes) < FIXED_HEADER_LEN {
		return 0, 0, nil, nil, NewIncompleteFrame(FIXED_HEADER_LEN, len(bytes), "fixed header")
	}

	// 获取帧长度
	frameLength := int(binary.BigEndian.Uint32(bytes[MAGIC_LEN+VERSION_LEN+TYPE_LEN+FLAGS_LEN:]))

	// 计算总帧长度
	totalFrameLen := FIXED_HEADER_LEN + frameLength
	if totalFrameLen < FIXED_HEADER_LEN {
		return 0, 0, nil, nil, NewLengthOverflow("total frame length calculation")
	}

	if len(bytes) < totalFrameLen {
		return 0, 0, nil, nil, NewIncompleteFrame(totalFrameLen, len(bytes), "complete frame")
	}

	currentFrame := bytes[0:totalFrameLen]
	remaining := bytes[totalFrameLen:]

	// CRC校验
	if totalFrameLen < 2 {
		return 0, 0, nil, nil, NewMalformedFrame("frame too short for CRC")
	}
	crcOffset := totalFrameLen - 2
	calculatedCrc := NET_CRC.Checksum(currentFrame[MAGIC_LEN:crcOffset])
	receivedCrc := binary.BigEndian.Uint16(currentFrame[crcOffset:])

	if calculatedCrc != receivedCrc {
		return 0, 0, nil, nil, NewCrcMismatch(receivedCrc, calculatedCrc)
	}

	// 校验版本号
	if _, err := ValidateVersion(bytes[MAGIC_LEN]); err != nil {
		return 0, 0, nil, nil, err
	}

	// 解析消息类型和标志位
	msgType, err := MessageTypeFromU8(bytes[MAGIC_LEN+VERSION_LEN])
	if err != nil {
		return 0, 0, nil, nil, err
	}

	flags, err := MessageFlagsFromU8(bytes[MAGIC_LEN+VERSION_LEN+TYPE_LEN])
	if err != nil {
		return 0, 0, nil, nil, err
	}

	return msgType, flags, currentFrame, remaining, nil
}

// GetChannelId 获取当前帧的通道ID
func GetChannelId(bytes []byte) (ChannelId, error) {
	if len(bytes) < MESSAGE_FIXED_LEN {
		return ChannelId{}, NewIncompleteFrame(MESSAGE_FIXED_LEN, len(bytes), "channel id")
	}

	channelIdStart := FIXED_HEADER_LEN
	channelIdEnd := FIXED_HEADER_LEN + CHANNEL_ID_LEN

	return ChannelIdTryFromSlice(bytes[channelIdStart:channelIdEnd])
}

// GetMessageSourceTarget 获取信息来源和目标
func GetMessageSourceTarget(bytes []byte) (string, string, error) {
	if len(bytes) < MESSAGE_FIXED_LEN {
		return "", "", NewIncompleteFrame(MESSAGE_FIXED_LEN, len(bytes), "fixed header")
	}

	offset := FIXED_HEADER_LEN + CHANNEL_ID_LEN

	// Source name 长度
	sourceLen := int(binary.BigEndian.Uint16(bytes[offset:]))
	if sourceLen > MAX_NAME_LENGTH {
		return "", "", NewNameTooLong("source", sourceLen, MAX_NAME_LENGTH)
	}
	if sourceLen == 0 {
		return "", "", NewEmptyName("source")
	}
	offset += SOURCE_LEN_SIZE

	// Source name
	sourceName := string(bytes[offset : offset+sourceLen])
	offset += sourceLen

	// Target name 长度
	targetLen := int(binary.BigEndian.Uint16(bytes[offset:]))
	if targetLen > MAX_NAME_LENGTH {
		return "", "", NewNameTooLong("target", targetLen, MAX_NAME_LENGTH)
	}
	if targetLen == 0 {
		return "", "", NewEmptyName("target")
	}
	offset += TARGET_LEN_SIZE

	// Target name
	targetName := string(bytes[offset : offset+targetLen])

	return sourceName, targetName, nil
}

// ParseMessage 解析消息
func ParseMessage(bytes []byte) (*MessageHeader, []byte, error) {
	if len(bytes) < MESSAGE_FIXED_LEN {
		return nil, nil, NewIncompleteFrame(MESSAGE_FIXED_LEN, len(bytes), "channel id")
	}

	// Channel ID
	channelIdStart := FIXED_HEADER_LEN
	channelIdEnd := FIXED_HEADER_LEN + CHANNEL_ID_LEN
	chanId, err := ChannelIdTryFromSlice(bytes[channelIdStart:channelIdEnd])
	if err != nil {
		return nil, nil, err
	}

	// Source 和 Target
	offset := FIXED_HEADER_LEN + CHANNEL_ID_LEN

	sourceLen := int(binary.BigEndian.Uint16(bytes[offset:]))
	if sourceLen > MAX_NAME_LENGTH {
		return nil, nil, NewNameTooLong("source", sourceLen, MAX_NAME_LENGTH)
	}
	if sourceLen == 0 {
		return nil, nil, NewEmptyName("source")
	}
	offset += SOURCE_LEN_SIZE

	sourceName := string(bytes[offset : offset+sourceLen])
	offset += sourceLen

	targetLen := int(binary.BigEndian.Uint16(bytes[offset:]))
	if targetLen > MAX_NAME_LENGTH {
		return nil, nil, NewNameTooLong("target", targetLen, MAX_NAME_LENGTH)
	}
	if targetLen == 0 {
		return nil, nil, NewEmptyName("target")
	}
	offset += TARGET_LEN_SIZE

	targetName := string(bytes[offset : offset+targetLen])
	offset += targetLen

	// Stream 总长度
	totalLen := int(binary.BigEndian.Uint32(bytes[offset:]))
	offset += TOTAL_STREAM_LEN

	// Stream 偏移
	streamOffset := int(binary.BigEndian.Uint32(bytes[offset:]))
	offset += STREAM_OFFSET_LEN

	// Payload（去掉最后的CRC 2字节）
	if len(bytes) < offset+2 {
		return nil, nil, NewMalformedFrame("frame too short for payload")
	}
	payload := bytes[offset : len(bytes)-2]

	header := &MessageHeader{
		ChannelId:    chanId,
		Source:       sourceName,
		Target:       targetName,
		TotalLen:     totalLen,
		StreamOffset: streamOffset,
	}

	return header, payload, nil
}
