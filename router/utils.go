package router

import (
	"strings"
)

// ParseFernqURL 解析 fernq:// 协议的URL字符串
// 输入: fernq://aaa/abd/dac 或 fernq://aaa/ 或 fernq://aaa
// 输出: target (aaa), path (/abd/dac 或 / 或 空字符串), error
func ParseFernqURL(input string) (target string, path string, err error) {
	const prefix = "fernq://"

	if !strings.HasPrefix(input, prefix) {
		return "", "", NewInvalidProtocol(input)
	}

	remaining := input[len(prefix):]
	if remaining == "" {
		return "", "", NewMissingTarget()
	}

	if idx := strings.Index(remaining, "/"); idx != -1 {
		target = remaining[:idx]
		path = remaining[idx:]
	} else {
		target = remaining
		path = ""
	}

	if target == "" {
		return "", "", NewMissingTarget()
	}

	return target, path, nil
}
