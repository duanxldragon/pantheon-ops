package generator

import "gorm.io/gorm"

type GeneratorService struct {
	db *gorm.DB
}

func NewGeneratorService(db *gorm.DB) *GeneratorService {
	return &GeneratorService{db: db}
}
