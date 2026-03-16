package router

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
