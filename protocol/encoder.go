package protocol

import (
	"encoding/binary"
)

// PingFrame 生成Ping帧
func PingFrame() []byte {
	frame := make([]byte, 0, FIXED_HEADER_LEN+2)
	frame = binary.BigEndian.AppendUint32(frame, MAGIC)
	frame = append(frame, VERSION)
	frame = append(frame, MessageTypePing.AsU8())
	frame = append(frame, NewMessageFlags(true, true, false).AsU8())
	frame = append(frame, 0, 0, 0, 2) // frame_length = 2 (CRC占位)

	// 计算CRC
	crc := NET_CRC.Checksum(frame[MAGIC_LEN:])
	frame = append(frame, uint8(crc>>8), uint8(crc))
	return frame
}

// PongFrame 生成Pong帧
func PongFrame() []byte {
	frame := make([]byte, 0, FIXED_HEADER_LEN+2)
	frame = binary.BigEndian.AppendUint32(frame, MAGIC)
	frame = append(frame, VERSION)
	frame = append(frame, MessageTypePong.AsU8())
	frame = append(frame, NewMessageFlags(true, true, true).AsU8())
	frame = append(frame, 0, 0, 0, 2) // frame_length = 2 (CRC占位)

	// 计算CRC
	crc := NET_CRC.Checksum(frame[MAGIC_LEN:])
	frame = append(frame, uint8(crc>>8), uint8(crc))
	return frame
}

// GenerateMessageDataStream 生成消息数据流
func GenerateMessageDataStream(
	messageType MessageType,
	endChannel bool,
	isResponse bool,
	channelId *ChannelId,
	sourceName string,
	targetName string,
	data []byte,
) ([][]byte, error) {
	var frames [][]byte

	// 验证名字
	if sourceName == "" {
		return nil, NewEmptyName("source")
	}
	if targetName == "" {
		return nil, NewEmptyName("target")
	}
	sourceNameBytes := []byte(sourceName)
	targetNameBytes := []byte(targetName)
	sourceNameLen := len(sourceNameBytes)
	targetNameLen := len(targetNameBytes)

	if sourceNameLen > MAX_NAME_LENGTH {
		return nil, NewNameTooLong("source", sourceNameLen, MAX_NAME_LENGTH)
	}
	if targetNameLen > MAX_NAME_LENGTH {
		return nil, NewNameTooLong("target", targetNameLen, MAX_NAME_LENGTH)
	}

	// 计算每个帧payload的最大长度
	maxPayloadLen := MAX_FRAME_SIZE - MESSAGE_FIXED_LEN - sourceNameLen - targetNameLen
	if maxPayloadLen <= 0 {
		return nil, NewInvalidFrameLength("names too long, no space for payload")
	}

	totalStreamLength := len(data)
	if totalStreamLength > MAX_STREAM_LENGTH {
		return nil, NewStreamTooLong(totalStreamLength, MAX_STREAM_LENGTH)
	}

	offset := 0
	for offset < totalStreamLength {
		payloadLen := min(maxPayloadLen, totalStreamLength-offset)
		capacity := MESSAGE_FIXED_LEN + sourceNameLen + targetNameLen + payloadLen
		frameLen := capacity - FIXED_HEADER_LEN

		frame := make([]byte, 0, capacity)

		// 魔数
		frame = binary.BigEndian.AppendUint32(frame, MAGIC)
		// 版本
		frame = append(frame, VERSION)
		// 消息类型
		frame = append(frame, messageType.AsU8())
		// 标志位
		endStream := offset+payloadLen >= totalStreamLength
		frame = append(frame, NewMessageFlags(endStream, endChannel, isResponse).AsU8())
		// 帧长度
		frame = binary.BigEndian.AppendUint32(frame, uint32(frameLen))
		// Channel ID
		frame = append(frame, channelId[:]...)
		// Source name 长度
		frame = binary.BigEndian.AppendUint16(frame, uint16(sourceNameLen))
		// Source name
		frame = append(frame, sourceNameBytes...)
		// Target name 长度
		frame = binary.BigEndian.AppendUint16(frame, uint16(targetNameLen))
		// Target name
		frame = append(frame, targetNameBytes...)
		// Stream 总长度
		frame = binary.BigEndian.AppendUint32(frame, uint32(totalStreamLength))
		// Stream 偏移
		frame = binary.BigEndian.AppendUint32(frame, uint32(offset))
		// Payload
		frame = append(frame, data[offset:offset+payloadLen]...)
		// CRC
		crc := NET_CRC.Checksum(frame[MAGIC_LEN:])
		frame = append(frame, uint8(crc>>8), uint8(crc))

		frames = append(frames, frame)
		offset += payloadLen
	}

	return frames, nil
}
