Read-only independent CODE follow-up. No tools. Max 150 words. Report proved P1/P2, otherwise APPROVE for this bounded change only. No release/project approval.
Follow-up to APPROVE with P2 double-query/legacy duplicates. This loader now derives completeness from one fetched row set; execution wrapper preserves old skip behavior; preview returns no new strings if complete=false. Dedicated real-MySQL tests pass unreadable and legacy duplicate cases. Review correctness of the supplied final source only.
func (s *DeployService) loadDeployTaskSecrets(taskID uint64) (map[string]string, error) {
secrets, _, err := s.loadDeployTaskSecretMaterial(taskID)
return secrets, err
}

// loadDeployTaskSecretMaterial derives completeness from the same row set as
// decryption. Preview must hide values if legacy duplicates or unreadable rows
// prevent reliable redaction; execution retains its existing skip behavior.
func (s *DeployService) loadDeployTaskSecretMaterial(taskID uint64) (map[string]string, bool, error) {
var rows []DeployTaskSecret
if err := s.db.Where(taskIDWhereClause, taskID).Find(&rows).Error; err != nil {
return nil, false, err
}
secrets := make(map[string]string, len(rows))
complete := true
for _, row := range rows {
decrypted, err := decryptDeployCredential(row.SecretEncrypted)
if err != nil {
complete = false
continue
}
if _, duplicate := secrets[row.SecretKey]; duplicate {
complete = false
}
secrets[row.SecretKey] = decrypted
}
return secrets, complete, nil
}

func (s *DeployService) taskPreviewPlan(task *DeployTask) (*TaskPreviewPlan, error) {
snapshot, err := decodeExecutionSnapshot(task.ExecutionSnapshot)
if err != nil {
return nil, err
}
secrets, complete, err := s.loadDeployTaskSecretMaterial(task.ID)
if err != nil {
return nil, err
}
return projectTaskPreviewPlan(snapshot, secrets, complete), nil
}
