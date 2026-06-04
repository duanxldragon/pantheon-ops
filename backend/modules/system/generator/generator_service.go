package generator

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"

	"pantheon-platform/backend/internal/scaffold"

	"gorm.io/gorm"
)

type GeneratorService struct {
	db            *gorm.DB
	workspaceRoot string
}

func NewGeneratorService(db *gorm.DB) *GeneratorService {
	workspaceRoot, _ := scaffold.ResolveWorkspaceRoot("")
	return &GeneratorService{db: db, workspaceRoot: workspaceRoot}
}

func (s *GeneratorService) PreviewGeneratedFiles(schema *scaffold.ModuleSchema) ([]scaffold.GeneratedFile, error) {
	if schema == nil {
		return nil, errors.New("module.generate.invalid_payload")
	}
	req := &scaffold.RegisterGeneratedModuleRequest{
		Schema: *schema,
	}
	if err := scaffold.ValidateRegisterRequest(req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return nil, errors.New("workspace.not_found")
	}
	return scaffold.GenerateModuleFilesFromSchema(s.workspaceRoot, *schema)
}

func (s *GeneratorService) BuildGeneratedModuleArchive(schema *scaffold.ModuleSchema) ([]byte, string, error) {
	files, err := s.PreviewGeneratedFiles(schema)
	if err != nil {
		return nil, "", err
	}

	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for _, file := range files {
		writer, err := archive.Create(file.Path)
		if err != nil {
			_ = archive.Close()
			return nil, "", err
		}
		if _, err := writer.Write([]byte(file.Content)); err != nil {
			_ = archive.Close()
			return nil, "", err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, "", err
	}

	archiveName := strings.ReplaceAll(strings.TrimSpace(schema.Name), "/", "-")
	if archiveName == "" {
		archiveName = "module"
	}
	return buffer.Bytes(), archiveName + "-module.zip", nil
}
