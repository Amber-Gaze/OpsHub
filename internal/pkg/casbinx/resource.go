package casbinx

import (
	"fmt"
	"strings"
)

// IsConfigResourcePath 判断是否为配置中心相关路径（网关转发鉴权时使用）。
func IsConfigResourcePath(httpPath string) bool {
	p := strings.TrimSpace(httpPath)
	return strings.Contains(p, "/configs")
}

// NormalizeConfigResource 将网关/配置 HTTP 路径转为 Casbin 资源串（斜杠分隔，供 globMatch）。
// 单段 key → config/default/<key>；多段 a/b → config/a/b（模块/子键）。
// 集合操作（路径仅为 /configs）→ config/*。
func NormalizeConfigResource(httpPath, httpMethod string) (obj, act string) {
	act = mapHTTPMethod(httpMethod)
	p := strings.TrimSpace(httpPath)
	for _, prefix := range []string{"/internal/configs", "/configs"} {
		if strings.HasPrefix(p, prefix) {
			p = strings.TrimPrefix(p, prefix)
			break
		}
	}
	p = strings.Trim(p, "/")
	if p == "" {
		return "config/*", act
	}
	if !strings.Contains(p, "/") {
		return fmt.Sprintf("config/default/%s", p), act
	}
	parts := strings.SplitN(p, "/", 2)
	return fmt.Sprintf("config/%s/%s", parts[0], parts[1]), act
}

func mapHTTPMethod(m string) string {
	switch strings.ToUpper(strings.TrimSpace(m)) {
	case "GET", "HEAD":
		return "read"
	case "POST", "PUT", "PATCH":
		return "write"
	case "DELETE":
		return "delete"
	default:
		return strings.ToLower(strings.TrimSpace(m))
	}
}
