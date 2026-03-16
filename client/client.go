package client

import (
	"sync"

	"github.com/fernq-org/gofernq/protocol"
	"github.com/fernq-org/gofernq/router"
)

// Client is a simple client for the Fernq protocol.
type Client struct {
	name       string
	router     *router.Router
	conn       *fernqConn
	mu         sync.Mutex
	is_conning sync.Mutex // 连接函数调用锁
}

// NewClient creates a new client with the given name and router.
func NewClient(name string, more ...*router.Router) *Client {
	router := router.NewRouter()
	if len(more) == 1 {
		router = more[0]
	}
	return &Client{
		name:       name,
		router:     router,
		mu:         sync.Mutex{},
		is_conning: sync.Mutex{},
	}
}

// Connect connects to the Fernq server.
func (c *Client) Connect(url string) error {
	// 给该函数上锁
	c.is_conning.Lock()
	defer c.is_conning.Unlock()

	c.mu.Lock()
	if c.conn != nil {
		c.mu.Unlock()
		return NewAlreadyConnected()
	}
	c.mu.Unlock()
	// 解析构建验证信息
	ci := protocol.NewChannelId()
	host, val, err := protocol.CreateValidate(c.name, &ci, url)
	if err != nil {
		return err
	}
	conn := newFernqConn(c.name, c.router)
	err = conn.tryConnect(host, val)
	if err != nil {
		return err
	}
	// 验证成功，开始监听
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	conn.wg.Add(1)
	go func() {
		defer conn.wg.Done()
		<-conn.ctx.Done()
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
	}()
	return nil
}

// Request makes a request to the Fernq server.
func (c *Client) Request(path string, body ...any) (*protocol.ResponseMessage, error) {
	// 解析path
	target, path, err := router.ParseFernqURL(path)
	if err != nil {
		return nil, err
	}
	// 判断是否连接交给外部连接处理
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return nil, NewConnectFailed("client not connected")
	}
	if len(body) > 0 {
		return conn.request(target, path, body[0])
	}
	return conn.request(target, path, nil)
}

// Done get the client' connention done channel
func (c *Client) Done() <-chan struct{} {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		ch := make(chan struct{})
		close(ch) // 模拟"已经取消的 context"
		return ch
	}
	return conn.done() // 用户也能监听 退出信号
}

// Close closes the client.
func (c *Client) Close() {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn != nil {
		conn.close()
	}
}
