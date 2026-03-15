package utils

import (
	"strconv"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/options"
)

const (
	APIBaseURL string = "http://127.0.0.1:"
	AuthPath   string = "/auth"
	UserPath   string = "/users"
	ConfigPath string = "/configs"
)

func GetAPIAuthBaseURL() string {
	return APIBaseURL + strconv.Itoa(options.GetAuthHTTPPort()) + AuthPath
}

func GetAPIGatewayBaseURL() string {
	return APIBaseURL + strconv.Itoa(options.GetGatewayHTTPPort())
}

// GetAuthServiceBaseURL 返回 IAM 服务根地址（无路径），供 Gateway 转发登录/刷新/鉴权等请求。
func GetAuthServiceBaseURL() string {
	return APIBaseURL + strconv.Itoa(options.GetAuthHTTPPort())
}

// GetConfigCenterServiceBaseURL 返回配置中心服务根地址（无路径），供 Gateway 转发 /internal/configs 等请求。
func GetConfigCenterServiceBaseURL() string {
	return APIBaseURL + strconv.Itoa(options.GetConfigCenterHTTPPort())
}

func GetConfigCenterBaseURL() string {
	return APIBaseURL + strconv.Itoa(options.GetConfigCenterHTTPPort()) + ConfigPath
}

// GetGatewayAuthBaseURL 优先使用配置中的 gateway.auth_base_url，未配置则用 IAM 端口推导。
func GetGatewayAuthBaseURL() string {
	if c := options.GetGatewayConf(); c != nil && c.AuthBaseURL != "" {
		return c.AuthBaseURL
	}
	return GetAuthServiceBaseURL()
}

// GetGatewayConfigCenterBaseURL 优先使用配置中的 gateway.config_center_base_url，未配置则用配置中心端口推导。
func GetGatewayConfigCenterBaseURL() string {
	if c := options.GetGatewayConf(); c != nil && c.ConfigCenterBaseURL != "" {
		return c.ConfigCenterBaseURL
	}
	return GetConfigCenterServiceBaseURL()
}
