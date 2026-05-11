-- Pantheon CMDB for Host Ops
-- 目标：开发运维平台的 Linux 主机纳管，不覆盖网络设备和存储建模
-- 范围：主机资产、分组、标签、连接凭据引用

SET NAMES utf8mb4;

-- 清理历史样板表，避免旧版通用 CMDB 模型污染当前运维版设计
DROP TABLE IF EXISTS `biz_cmdb_host_auth_ref`;
DROP TABLE IF EXISTS `biz_cmdb_host_tag_rel`;
DROP TABLE IF EXISTS `biz_cmdb_host_tag`;
DROP TABLE IF EXISTS `biz_cmdb_host_group_rel`;
DROP TABLE IF EXISTS `biz_cmdb_host_group`;
DROP TABLE IF EXISTS `biz_cmdb_host`;
DROP TABLE IF EXISTS `biz_cmdb_relation`;
DROP TABLE IF EXISTS `biz_cmdb_item`;
DROP TABLE IF EXISTS `biz_cmdb_type`;

CREATE TABLE `biz_cmdb_host` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `host_code` VARCHAR(64) NOT NULL COMMENT '主机编码',
  `hostname` VARCHAR(128) NOT NULL COMMENT '主机名',
  `display_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '展示名称',
  `ip_address` VARCHAR(64) NOT NULL COMMENT '主 IP 地址',
  `ssh_port` INT UNSIGNED NOT NULL DEFAULT 22 COMMENT 'SSH 端口',
  `os_family` ENUM('linux') NOT NULL DEFAULT 'linux' COMMENT '操作系统族',
  `os_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '操作系统名称',
  `kernel_version` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '内核版本',
  `arch` ENUM('x86_64','arm64') NOT NULL DEFAULT 'x86_64' COMMENT 'CPU 架构',
  `environment` ENUM('dev','test','staging','prod') NOT NULL DEFAULT 'dev' COMMENT '环境',
  `status` ENUM('active','inactive','retired') NOT NULL DEFAULT 'active' COMMENT '资产状态',
  `lifecycle_status` ENUM('provisioning','running','offline','decommissioned') NOT NULL DEFAULT 'running' COMMENT '生命周期状态',
  `provider` ENUM('physical','vmware','kvm','aws','aliyun','tencent') NOT NULL DEFAULT 'physical' COMMENT '资源提供方',
  `region_code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '地域编码',
  `idc_code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '机房编码',
  `cluster_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '集群名称',
  `owner_user_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '资产负责人用户 ID',
  `owner_name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '资产负责人',
  `maintainer_team` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '维护团队',
  `purpose` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '用途说明',
  `last_check_in_at` DATETIME NULL DEFAULT NULL COMMENT '最近心跳时间',
  `last_inventory_at` DATETIME NULL DEFAULT NULL COMMENT '最近盘点时间',
  `last_operated_at` DATETIME NULL DEFAULT NULL COMMENT '最近运维操作时间',
  `remark` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '备注',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cmdb_host_code` (`host_code`),
  UNIQUE KEY `uk_cmdb_host_ip` (`ip_address`),
  KEY `idx_cmdb_host_environment_status` (`environment`, `status`),
  KEY `idx_cmdb_host_owner_user_id` (`owner_user_id`),
  KEY `idx_cmdb_host_cluster_name` (`cluster_name`),
  KEY `idx_cmdb_host_last_check_in_at` (`last_check_in_at`),
  KEY `idx_cmdb_host_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='CMDB 主机主表';

CREATE TABLE `biz_cmdb_host_group` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `group_code` VARCHAR(64) NOT NULL COMMENT '分组编码',
  `group_name` VARCHAR(128) NOT NULL COMMENT '分组名称',
  `group_type` ENUM('business','project','environment','cluster','custom') NOT NULL DEFAULT 'custom' COMMENT '分组类型',
  `parent_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '父分组 ID',
  `owner_name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '分组负责人',
  `status` ENUM('active','inactive') NOT NULL DEFAULT 'active' COMMENT '状态',
  `sort` INT NOT NULL DEFAULT 100 COMMENT '排序',
  `remark` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '备注',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME NULL DEFAULT NULL COMMENT '删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cmdb_host_group_code` (`group_code`),
  KEY `idx_cmdb_host_group_parent_id` (`parent_id`),
  KEY `idx_cmdb_host_group_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='CMDB 主机分组';

CREATE TABLE `biz_cmdb_host_group_rel` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `host_id` BIGINT UNSIGNED NOT NULL COMMENT '主机 ID',
  `group_id` BIGINT UNSIGNED NOT NULL COMMENT '分组 ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cmdb_host_group_rel` (`host_id`, `group_id`),
  KEY `idx_cmdb_host_group_rel_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='CMDB 主机与分组关联';

CREATE TABLE `biz_cmdb_host_tag` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tag_key` VARCHAR(64) NOT NULL COMMENT '标签键',
  `tag_name` VARCHAR(128) NOT NULL COMMENT '标签名称',
  `tag_color` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '标签颜色',
  `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cmdb_host_tag_key` (`tag_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='CMDB 主机标签定义';

CREATE TABLE `biz_cmdb_host_tag_rel` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `host_id` BIGINT UNSIGNED NOT NULL COMMENT '主机 ID',
  `tag_id` BIGINT UNSIGNED NOT NULL COMMENT '标签 ID',
  `tag_value` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '标签值',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cmdb_host_tag_rel` (`host_id`, `tag_id`),
  KEY `idx_cmdb_host_tag_lookup` (`tag_id`, `tag_value`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='CMDB 主机标签关联';

CREATE TABLE `biz_cmdb_host_auth_ref` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `host_id` BIGINT UNSIGNED NOT NULL COMMENT '主机 ID',
  `auth_type` ENUM('password','ssh_key') NOT NULL DEFAULT 'ssh_key' COMMENT '认证方式',
  `username` VARCHAR(64) NOT NULL COMMENT '登录用户名',
  `credential_ref` VARCHAR(255) NOT NULL COMMENT '凭据引用标识',
  `sudo_enabled` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否启用 sudo',
  `sudo_username` VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'sudo 用户名',
  `connection_strategy` ENUM('direct','bastion') NOT NULL DEFAULT 'direct' COMMENT '连接方式',
  `bastion_host_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '跳板机主机 ID',
  `remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_cmdb_host_auth_ref` (`host_id`, `username`),
  KEY `idx_cmdb_host_auth_bastion_host_id` (`bastion_host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='CMDB 主机连接凭据引用';
