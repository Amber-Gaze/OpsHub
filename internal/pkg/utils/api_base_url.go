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
	var prot int = options.GetAuthHTTPPort()
	return APIBaseURL + strconv.Itoa(prot) + AuthPath
}

func GetAPIGatewayBaseURL() string {
	var prot int = options.GetGatewayHTTPPort()
	return APIBaseURL + strconv.Itoa(prot)
}

func GatConfigCenterBaseURL() string {
	var prot int = options.GetConfigCenterHTTPPort()
	return APIBaseURL + strconv.Itoa(prot) + ConfigPath
}
