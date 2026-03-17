package router

import "fmt"

// Router 路由器
type Router struct {
	root *node // 路径树
}

// NewRouter 创建一个新的 Router 实例
func NewRouter() *Router {
	return &Router{
		root: &node{children: make(map[string]*node), handler: defaultHandler},
	}
}

// 添加路由
func (r *Router) AddRoute(path string, handler func(*Context)) error {
	return r.root.insert(path, handler)
}

// 处理请求的context
func (r *Router) Handle(ctx *Context) {
	// panic 防护：确保单个请求崩溃不会影响整个服务
	defer func() {
		if rec := recover(); rec != nil {
			// 直接设置 500 错误，返回给客户端
			ctx.JSON(StatusInternalServerError, map[string]interface{}{
				"error":   "Internal Server Error",
				"message": fmt.Sprintf("panic: %v", rec),
			})
		}
	}()
	route, err := r.root.find(ctx.request.Path)
	if err != nil {
		defaultHandler(ctx)
		return
	}
	route.handler(ctx)
	if ctx.response == nil {
		ctx.JSON(StatusOK, nil)
	}
}
