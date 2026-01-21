package service

import "github.com/Amber-Gaze/OpsHub/internal/domain"

type ConfigService struct{}

func NewConfigService() *ConfigService {
	return &ConfigService{}
}

func (s *ConfigService) List() ([]domain.Config, error) {
	return []domain.Config{}, nil
}
