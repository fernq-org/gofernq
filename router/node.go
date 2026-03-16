package router

import (
	"strings"
)

// 默认 handler 函数
func defaultHandler(c *Context) {
	// 发布错误码 404 Not Found
	c.JSON(StatusNotFound, "404 Not Found !")
}

// 切分路径
// 输入：
//
//   - repath: /aaa/bbb/ccc 特殊情况是输入 "/" 和 ""
//
// 输出：
//
//   - re: aaa
//   - ma: /bbb/ccc
//   - err: 解析失败返回错误,特殊情况是输入 "/" 和 "" 时返回 "", "", nil
func splitPath(repath string) (re string, ma string, err error) {
	// 处理空字符串
	if repath == "" {
		return "", "", NewEmptyPath()
	}

	// 必须以 "/" 开头
	if !strings.HasPrefix(repath, "/") {
		return "", "", NewPathNotAbsolute(repath)
	}

	// 特殊情况：输入 "/" 时返回 "", "", nil
	if repath == "/" {
		return "", "", nil
	}

	// 去掉开头的 "/"
	trimmed := repath[1:]

	// 找到下一个 "/" 的位置
	slashIdx := strings.Index(trimmed, "/")

	if slashIdx == -1 {
		// 没有更多 "/"，这是最后一个路径段
		return trimmed, "", nil
	}

	// 分割路径
	re = trimmed[:slashIdx]
	ma = trimmed[slashIdx:] // 保留开头的 "/"

	return re, ma, nil
}

// node 路由树节点
type node struct {
	children map[string]*node // 子节点，key 为 "aaa", "bbb"
	is_index bool             // 是否是索引节点
	handler  func(*Context)   // 请求处理器
}

// 寻找路由节点
func (n *node) find(repath string) (*node, error) {
	if repath == "/" || repath == "" {
		// 根节点 或者 叶子节点
		return n, nil
	}

	// 切分路径
	re, ma, err := splitPath(repath)
	if err != nil {
		return nil, err
	}

	// 查找子节点
	child, ok := n.children[re]
	if !ok {
		return nil, NewRouteNotFound(repath)
	}
	return child.find(ma)
}

// 填充路由树
func (n *node) insert(repath string, handler func(*Context)) error {
	if repath == "/" || repath == "" {
		// 根节点 或者 叶子节点
		n.handler = handler
		if n.is_index {
			// 索引节点
			panic(NewAlreadyUsedAsIndex())
		}
		n.is_index = true // 标记为可索引节点
		return nil
	}
	// 切分路径
	re, ma, err := splitPath(repath)
	if err != nil {
		return err
	}

	// 查找子节点
	child, ok := n.children[re]
	if !ok {
		// 创建新节点
		child = &node{children: make(map[string]*node), handler: defaultHandler}
		n.children[re] = child
	}
	return child.insert(ma, handler)
}
