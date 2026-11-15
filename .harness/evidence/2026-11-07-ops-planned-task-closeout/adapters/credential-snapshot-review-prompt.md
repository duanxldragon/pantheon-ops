Read-only bounded source review, no tools, max180 words. O1 file-distribution target snapshot duplicated plaintext credentials although target rows are encrypted. This repair copies the request slice, removes password/privateKey from the task JSON and preserves encrypted worker rows. Real MySQL regression proves snapshot omission, caller immutability, nonsecret facts, decryption and zero retry preservation. Scheduled future tasks avoid any SSH actions. Review only this change; scope/authz/state/engine internals, historical snapshots and audit middleware omitted. Report proved defects or APPROVE supplied scope only.
func (s *DistributionService) CreateTask(dto DistributionTaskDTO, createdBy string) (*DistributionTaskResponse, error) {
// Verify file exists
file, err := s.storageService.GetFileByID(dto.FileID)
if err != nil {
return nil, fmt.Errorf("file.not_found: %w", err)
}

    // Keep execution credentials only in the encrypted target rows. Copy the
    // request slice so snapshot sanitization cannot mutate caller-owned input.
    snapshotTargets := append([]TargetConfig(nil), dto.Targets...)
    for i := range snapshotTargets {
    	snapshotTargets[i].Password = ""
    	snapshotTargets[i].PrivateKey = ""
    }
    targetsJSON, _ := json.Marshal(snapshotTargets)

    task := &DistributionTask{
    	Name:         dto.Name,
    	FileID:       dto.FileID,
    	Targets:      datatypes.JSON(targetsJSON),
    	Status:       statusPending,
    	TotalTargets: len(dto.Targets),
    	ScheduledAt:  dto.ScheduledAt,
    	CreatedBy:    createdBy,
    }

    if err := s.db.Transaction(func(tx *gorm.DB) error {
    	if err := tx.Create(task).Error; err != nil {
    		return err
    	}
    	for _, target := range dto.Targets {
    		encryptedPassword, err := secrets.Encrypt(target.Password)
    		if err != nil {
    			return fmt.Errorf("encrypt_password: %w", err)
    		}
    		encryptedKey, err := secrets.Encrypt(target.PrivateKey)
    		if err != nil {
    			return fmt.Errorf("encrypt_private_key: %w", err)
    		}

    		targetRecord := &DistributionTarget{
    			TaskID:          task.ID,
    			Host:            target.Host,
    			Port:            target.Port,
    			TargetPath:      target.TargetPath,
    			Username:        target.Username,
    			Password:        encryptedPassword,
    			PrivateKey:      encryptedKey,
    			HostFingerprint: target.HostFingerprint,
    			Status:          statusPending,
    			MaxRetries:      target.MaxRetries,
    		}
    		if targetRecord.Port == 0 {
    			targetRecord.Port = 22
    		}
    		if err := tx.Create(targetRecord).Error; err != nil {
    			return err
    		}
    	}
    	return nil
    }); err != nil {
    	return nil, err
    }

    // Start task if not scheduled
    if dto.ScheduledAt == nil || dto.ScheduledAt.Before(time.Now()) {
    	go s.executeTask(task.ID)
    }

    return s.toTaskResponse(task, file), nil

}

describes where a distribution task pushes a file on one target.
type TargetConfig struct {
Host string `json:"host" binding:"required"`
Port int `json:"port"`
TargetPath string `json:"targetPath" binding:"required"`
Username string `json:"username" binding:"required"`
Password string `json:"password,omitempty"`
PrivateKey string `json:"privateKey,omitempty"`
HostFingerprint string `json:"hostFingerprint" binding:"required"` // SSH host key fingerprint (SHA256)
MaxRetries int `json:"maxRetries"`
}
