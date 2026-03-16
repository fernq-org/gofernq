package client

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/fernq-org/gofernq/protocol"
	"github.com/fernq-org/gofernq/router"
	"github.com/puzpuzpuz/xsync/v3"
)

const (
	MAX_HEADER_LEN  = 4 * 1024 * 1024   // 4M 最大头部长度
	REQUEST_TIMEOUT = 300 * time.Second // 请求超时时间
	AWAIT_TIMEOUT   = 5 * time.Minute   // 等待超时
	CHANN_SIZE      = 4096              // 默认的通道大小
)

// fernqConn represents a connection to a Fernq server.
type fernqConn struct {
	name              string
	router            *router.Router
	conn              net.Conn
	outChan           chan any
	wg                *sync.WaitGroup
	ctx               context.Context
	cancel            context.CancelFunc
	onceRun           sync.Once
	internalCloseChan chan struct{} // 关闭通道
	// 存储 ChannelId -> chan *protocol.ResponseMessage 的映射
	chMap *xsync.MapOf[protocol.ChannelId, chan *protocol.ResponseMessage]
	// 存储 ChannelId -> chan *protocol.Frame 的映射
	chFMap *xsync.MapOf[protocol.ChannelId, chan *protocol.Frame]
}

// newFernqConn create a new fernqConn
func newFernqConn(name string, router *router.Router) *fernqConn {
	ctx, cancel := context.WithCancel(context.Background())
	return &fernqConn{
		name:              name,
		router:            router,
		outChan:           make(chan any, CHANN_SIZE),
		wg:                &sync.WaitGroup{},
		ctx:               ctx,
		cancel:            cancel,
		onceRun:           sync.Once{},
		internalCloseChan: make(chan struct{}),
		chFMap:            xsync.NewMapOf[protocol.ChannelId, chan *protocol.Frame](),
		chMap:             xsync.NewMapOf[protocol.ChannelId, chan *protocol.ResponseMessage](),
	}
}

// request Fernq
func (fc *fernqConn) request(target string, path string, body any) (*protocol.ResponseMessage, error) {
	var request *protocol.RequestMessage
	var err error
	if body == nil {
		request = protocol.NewRequestMessage(path, protocol.ContentTypeJson)
	} else {
		request, err = protocol.NewRequestMessageWithBody(path, body)
		if err != nil {
			return nil, err
		}
	}
	if target == fc.name { // 本地处理
		// 创建context
		context := router.NewContext(nil, request)
		// 处理请求
		fc.router.Handle(context)
		// 等待处理完成,返回响应
		return context.GetResponse(), nil
	}
	// 非本地处理
	// 创建channel
	ch := make(chan *protocol.ResponseMessage, 1)
	// 创建channelId
	channelId := protocol.NewChannelId()
	// 存储channelId -> ch
	fc.chMap.Store(channelId, ch)
	// 创建请求帧
	frames, err := fc.requestToBytes(&channelId, target, request)
	// 发送请求帧
	for _, frame := range frames {
		fc.outChan <- frame
	}
	// 等待响应
	var resp *protocol.ResponseMessage
	var recvOk bool

	select {
	case <-fc.ctx.Done():
		err = fc.ctx.Err()
	case <-time.After(REQUEST_TIMEOUT):
		err = router.NewRequestTimeout()
	case resp, recvOk = <-ch:
		if !recvOk {
			err = router.NewRequestTimeout()
		}
	}

	// 统一清理
	fc.chMap.Delete(channelId)
	close(ch)

	if err != nil {
		return nil, err
	}
	return resp, nil
}

// connect to server
func (fc *fernqConn) tryConnect(serverAddress string, validate []byte) error {
	// 创建连接
	conn, err := net.Dial("tcp", serverAddress)
	if err != nil {
		return NewConnectFailed(err.Error())
	}
	// 连接成功，进行验证
	_, err = conn.Write(validate)
	if err != nil {
		conn.Close()
		return NewConnectFailed(err.Error())
	}
	// 发送成功，获取响应
	// 设置最长总时间 3分钟
	timeout := time.NewTimer(3 * time.Minute)
	defer timeout.Stop()
	// 读取数据
	var buff []byte
	for {
		select {
		case <-fc.ctx.Done():
			conn.Close()
			return nil
		case <-timeout.C:
			conn.Close()
			return NewConnectFailed("timeout")
		default:
		}

		err := conn.SetReadDeadline(time.Now().Add(time.Second * 5))
		if err != nil {
			conn.Close()
			return err
		}

		// 读取数据
		buf := make([]byte, 8*1024)
		n, err := conn.Read(buf)
		// 首先检查是否有错误
		if err != nil {
			// 检查是否为超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// log.Println("读取超时，重新设置超时并继续等待...")
				continue // 超时后重新循环，不执行下面的数据处理
			}
			conn.Close()
			return err // 退出循环，连接会被清理
		}
		// 拼接数据
		buff = append(buff, buf[:n]...)
		// 尝试解析看是否数据完整
		_, _, cf, rm, err := protocol.DecodeBasic(buff)
		if err != nil {
			if protocol.GetErrorCode(err) == protocol.ErrIncompleteFrame {
				// log.Println("数据不完整，继续等待...")
				continue // 数据不完整，继续等待
			}
			conn.Close()
			return err // 退出循环，连接会被清理
		}
		// 当前帧完整，第一帧肯定是响应
		buff = rm
		// 解析响应
		_, res, err := protocol.ParseMessage(cf)
		if err != nil {
			conn.Close()
			return err
		}
		result, msg, err := protocol.ParseVerifyResponse(res)
		if err != nil {
			conn.Close()
			return err
		}
		if !result {
			conn.Close()
			return NewVerifyFailed(msg)
		}
		// 构建连接协程
		fc.wg.Add(2)
		go fc.listen(buff, conn)
		go fc.out(conn)
		fc.conn = conn
		return nil
	}
}

// listen the connection
func (fc *fernqConn) listen(buff []byte, conn net.Conn) {
	defer func() {
		if fc.ctx.Err() == nil {
			fc.onceClose() // 关闭连接
		}
		fc.wg.Done()
	}()
	for {
		err := conn.SetReadDeadline(time.Now().Add(time.Second * 5))
		if err != nil {
			return
		}

		// 读取数据
		buf := make([]byte, 8*1024)
		n, err := conn.Read(buf)
		// 首先检查是否有错误
		if err != nil {
			// 检查是否为超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// log.Println("读取超时，重新设置超时并继续等待...")
				continue // 超时后重新循环，不执行下面的数据处理
			}
			return // 退出循环，连接会被清理
		}
		// 拼接数据
		buff = append(buff, buf[:n]...)
		mt, mf, cf, rm, err := protocol.DecodeBasic(buff)
		if err != nil {
			if protocol.GetErrorCode(err) == protocol.ErrIncompleteFrame {
				// log.Println("数据不完整，继续等待...")
				continue // 数据不完整，继续等待
			}
			return // 退出循环，连接会被清理
		}
		buff = rm                           // 移除已处理的数据
		if mt == protocol.MessageTypePing { // ping 返回 pong 开启新的循环
			fc.outChan <- protocol.PongFrame()
			continue
		}
		fc.dealFrame(mt, mf, cf)
	}
}

// out the connection's message
func (fc *fernqConn) out(conn net.Conn) {
	defer fc.wg.Done()
	for {
		select {
		case <-fc.ctx.Done():
			return
		case msg := <-fc.outChan:
			conn.Write(msg.([]byte))
		}
	}
}

// done the connection
func (fc *fernqConn) done() <-chan struct{} {
	return fc.internalCloseChan
}

// once do conn.close
func (fc *fernqConn) onceClose() {
	fc.onceRun.Do(func() {
		fc.conn.Close()
		fc.cancel()
		close(fc.internalCloseChan)
	})
}

// close the connection
func (fc *fernqConn) close() {
	fc.onceClose()
	fc.wg.Wait()
}

// deal the message's frame
func (fc *fernqConn) dealFrame(mt protocol.MessageType, mf protocol.MessageFlags, data []byte) {
	// 解析数据
	header, payload, err := protocol.ParseMessage(data)
	if err != nil {
		return
	}
	// 根据类型进行处理
	switch mt {
	case protocol.MessageTypeHeader:
		fc.chFMap.Compute(header.ChannelId, func(ch chan *protocol.Frame, loaded bool) (chan *protocol.Frame, bool) {
			if loaded {
				// 已有通道，直接发送帧
				ch <- &protocol.Frame{
					Data:          payload,
					MessageFlags:  mf,
					MessageHeader: *header,
					MessageType:   mt,
				}
				// 保留原值，不删除
				return ch, false
			}
			// 创建新通道
			ch = make(chan *protocol.Frame, CHANN_SIZE)
			// 启动协程处理（此时通道还未正式存储，但 Compute 会原子性地存储）
			fc.wg.Add(1)
			go fc.receiveWork(header.ChannelId, ch)
			// 发送首帧
			ch <- &protocol.Frame{
				Data:          payload,
				MessageFlags:  mf,
				MessageHeader: *header,
				MessageType:   mt,
			}
			// 返回新通道让 Compute 自动存储，false 表示不删除
			return ch, false
		})
	case protocol.MessageTypeBody:
		fc.chFMap.Compute(header.ChannelId, func(ch chan *protocol.Frame, loaded bool) (chan *protocol.Frame, bool) {
			if !loaded {
				// 不存在，静默丢弃，不创建
				return nil, true
			}
			// 存在，发送帧，保留原值
			ch <- &protocol.Frame{
				Data:          payload,
				MessageFlags:  mf,
				MessageHeader: *header,
				MessageType:   mt,
			}
			return ch, false
		})
	default:
		return
	}
}

// receive work
func (fc *fernqConn) receiveWork(chanID protocol.ChannelId, inChan <-chan *protocol.Frame) {
	// 关闭并清理通道的辅助函数
	cleanup := func() {
		fc.chFMap.Compute(chanID, func(ch chan *protocol.Frame, loaded bool) (chan *protocol.Frame, bool) {
			if loaded {
				// 关闭通道
				close(ch)
			}
			// 删除 key
			return nil, true
		})
	}
	defer func() {
		cleanup()
		fc.wg.Done()
	}()
	// 声明接收请求头的[]byte
	var header []byte
	is_init := false
	timer := time.NewTimer(AWAIT_TIMEOUT)
	defer timer.Stop()
	for {
		select {
		case <-fc.ctx.Done():
			return
		case <-timer.C:
			return
		case fr, ok := <-inChan:
			if !ok {
				return
			}
			// 重置定时器
			if !timer.Stop() {
				select {
				case <-timer.C: // 非阻塞尝试排空
				default: // 通道空时立即走这里
				}
			}
			timer.Reset(AWAIT_TIMEOUT)

			if !is_init {
				header = make([]byte, fr.MessageHeader.TotalLen)
				is_init = true
			}
			copy(header[fr.MessageHeader.StreamOffset:], fr.Data)
			if fr.MessageFlags.EndStream() {
				// 解析请求头
				if fr.MessageType != protocol.MessageTypeHeader {
					return
				}
				// 解析信息
				message, err := protocol.DecodeFromBytes(header)
				if err != nil {
					return
				}
				// 判断信息类型
				switch ms := message.(type) {
				case *protocol.RequestMessage:
					// 创建context
					context := router.NewContext(inChan, ms)
					// 处理请求
					fc.router.Handle(context)
					// 转成响应帧
					frames, err := fc.responseToBytes(context, *fr)
					if err != nil {
						return
					}
					// 发送响应
					for _, frame := range frames {
						fc.outChan <- frame
					}
				case *protocol.ResponseMessage:
					// 原子操作：查找 + 发送（回调执行期间 key 被锁定）
					fc.chMap.Compute(fr.MessageHeader.ChannelId,
						func(ch chan *protocol.ResponseMessage, loaded bool) (chan *protocol.ResponseMessage, bool) {
							if !loaded {
								// key 不存在，静默处理（不创建、不修改）
								return nil, true // delete=true 表示不保留
							}
							// key 存在，在锁保护下发送（此时其他协程无法 Delete/覆盖此 key）
							select {
							case ch <- ms:
								// 发送成功
							default:
								// 通道满，静默丢弃
							}
							// 保留原值，不做修改
							return ch, false // delete=false 表示保留原值
						})
				default:
				}
				return
			}
		}
	}
}

// router context's response to bytes
func (fc *fernqConn) responseToBytes(ctx *router.Context, frame protocol.Frame) ([][]byte, error) {
	data, err := ctx.GetResponse().ToBytes()
	if err != nil {
		return nil, err
	}
	return protocol.GenerateMessageDataStream(protocol.MessageTypeHeader, true, true,
		&frame.MessageHeader.ChannelId, fc.name, frame.MessageHeader.Source, data)
}

// router context's request to bytes
func (fc *fernqConn) requestToBytes(chanID *protocol.ChannelId, target string, request *protocol.RequestMessage) ([][]byte, error) {
	data, err := request.ToBytes()
	if err != nil {
		return nil, err
	}
	if len(data) > MAX_HEADER_LEN {
		return nil, router.NewRequestTooLarge(len(data))
	}
	return protocol.GenerateMessageDataStream(protocol.MessageTypeHeader, true, false,
		chanID, fc.name, target, data)
}
