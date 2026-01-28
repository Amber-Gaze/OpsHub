package utils

import (
	"strconv"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/config"
)

const (
	APIBaseURL string = "http://127.0.0.1:"
	AuthPath   string = "/auth"
	ConfigPath string = "/configs"
)

func GetAPIAuthBaseURL() string {
	var prot int = config.GetAuthHTTPPort()
	return APIBaseURL + strconv.Itoa(prot) + AuthPath
}

func GetAPIGatewayBaseURL() string {
	var prot int = config.GetGatewayHTTPPort()
	return APIBaseURL + strconv.Itoa(prot)
}

func GatConfigCenterBaseURL() string {
	var prot int = config.GetConfigCenterHTTPPort()
	return APIBaseURL + strconv.Itoa(prot) + ConfigPath
}
