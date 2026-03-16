package protocol

// CRC-16/XMODEM 实现
var NET_CRC = crc16Xmodem()

func crc16Xmodem() *crc16Table {
	// CRC-16/XMODEM 多项式: 0x1021
	return makeCRC16Table(0x1021)
}

type crc16Table struct {
	table [256]uint16
	poly  uint16
}

func makeCRC16Table(poly uint16) *crc16Table {
	t := &crc16Table{poly: poly}
	for i := 0; i < 256; i++ {
		crc := uint16(i) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
		t.table[i] = crc
	}
	return t
}

func (c *crc16Table) Checksum(data []byte) uint16 {
	var crc uint16 = 0
	for _, b := range data {
		crc = c.table[byte(crc>>8)^b] ^ (crc << 8)
	}
	return crc
}

// ========== 核心常量 ==========
const (
	MAGIC   uint32 = 20262351
	VERSION uint8  = 1
)

// 固定字节长度
const (
	MAGIC_LEN         = 4  // 魔数 4字节
	VERSION_LEN       = 1  // 版本 1字节
	TYPE_LEN          = 1  // 类型 1字节
	FLAGS_LEN         = 1  // 标志 1字节
	FRAME_LENGTH_LEN  = 4  // 帧长度 4字节
	CHANNEL_ID_LEN    = 16 // Channel ID 16字节
	SOURCE_LEN_SIZE   = 2  // Source name 长度标识 2字节
	TARGET_LEN_SIZE   = 2  // Target name 长度标识 2字节
	TOTAL_STREAM_LEN  = 4  // Stream 总长度 4字节
	STREAM_OFFSET_LEN = 4  // Stream 偏移 4字节
	CRC_LEN           = 2  // CRC 2字节
)

// 计算得到的固定头部长度
const (
	FIXED_HEADER_LEN  = MAGIC_LEN + VERSION_LEN + TYPE_LEN + FLAGS_LEN + FRAME_LENGTH_LEN // = 11
	MESSAGE_FIXED_LEN = FIXED_HEADER_LEN + CHANNEL_ID_LEN + SOURCE_LEN_SIZE +
		TARGET_LEN_SIZE + TOTAL_STREAM_LEN + STREAM_OFFSET_LEN + CRC_LEN // = 41
)

// 限制常量
const (
	MAX_FRAME_SIZE    = 8 * 1024        // 8KB
	MAX_STREAM_LENGTH = 8 * 1024 * 1024 // 8MB
	MAX_NAME_LENGTH   = 128             // Source/Target 最大长度
)
