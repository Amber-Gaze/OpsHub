package apiserver

import "github.com/LiYaYaoo0/OpsHub/internal/config"

type apiServer struct {
}

func CreateAPIServer(cfg *config.Config) (*apiServer, error) {
	return &apiServer{}, nil
}

func (s *apiServer) Run() error {
	return nil
}
