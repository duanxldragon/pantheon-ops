INSERT INTO `biz_k8s_cluster_credential_ref` (`cluster_id`, `encrypted`, `version`, `status`, `created_at`, `updated_at`)
SELECT c.`id`, c.`kubeconfig_encrypted`, 1, 'active', NOW(3), NOW(3)
FROM `biz_k8s_cluster` c
WHERE c.`kubeconfig_credential_ref_id` = 0
  AND c.`kubeconfig_encrypted` <> ''
  AND NOT EXISTS (
    SELECT 1 FROM `biz_k8s_cluster_credential_ref` r WHERE r.`cluster_id` = c.`id`
  );

UPDATE `biz_k8s_cluster` c
JOIN `biz_k8s_cluster_credential_ref` r ON r.`cluster_id` = c.`id`
SET c.`kubeconfig_credential_ref_id` = r.`id`
WHERE c.`kubeconfig_credential_ref_id` = 0;
