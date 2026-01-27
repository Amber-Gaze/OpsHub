package mysql

import (
	"gorm.io/gorm"
)

type ProjectRepository struct {
	db *gorm.DB
}

// func NewProjectRepository(db *gorm.DB) *ProjectRepository {
// 	return &ProjectRepository{db: db}
// }

// func (r *ProjectRepository) Create(p *domain.Project) error {
// 	return r.db.Create(p).Error
// }

// func (r *ProjectRepository) List() ([]domain.Project, error) {
// 	var projects []domain.Project
// 	err := r.db.Find(&projects).Error
// 	return projects, err
// }
