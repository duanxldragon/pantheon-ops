-- Pantheon Base 数据库初始化脚本
-- 引擎：MySQL / PostgreSQL (兼容建议使用 MySQL 8.0+)
-- 模块：System (系统管理)
-- 前缀：system_

SET NAMES utf8mb4;

-- 1. 用户表
CREATE TABLE `system_user` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `username` varchar(64) NOT NULL COMMENT '用户名',
    `password` varchar(255) NOT NULL COMMENT '密码',
    `nickname` varchar(64) DEFAULT NULL COMMENT '昵称',
    `avatar` varchar(255) DEFAULT NULL COMMENT '头像',
    `email` varchar(128) DEFAULT NULL COMMENT '邮箱',
    `phone` varchar(20) DEFAULT NULL COMMENT '手机号',
    `preference_json` text COMMENT '平台壳层偏好 JSON',
    `dept_id` bigint DEFAULT '0' COMMENT '部门ID',
    `post_id` bigint DEFAULT '0' COMMENT '岗位ID',
    `status` tinyint DEFAULT '1' COMMENT '状态 (1:正常, 2:禁用)',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    `deleted_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_username` (`username`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统用户表';

-- 2. 角色表
CREATE TABLE `system_role` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `role_name` varchar(64) NOT NULL COMMENT '角色名称',
    `role_key` varchar(64) NOT NULL COMMENT '角色权限字符串',
    `sort` int DEFAULT '0' COMMENT '显示顺序',
    `status` tinyint DEFAULT '1' COMMENT '状态 (1:正常, 2:禁用)',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    `deleted_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_role_key` (`role_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色信息表';

-- 3. 菜单权限表
CREATE TABLE `system_menu` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `parent_id` bigint DEFAULT '0' COMMENT '父菜单ID',
    `title_key` varchar(64) NOT NULL COMMENT '菜单标题i18n key',
    `path` varchar(255) DEFAULT '' COMMENT '路由地址',
    `component` varchar(255) DEFAULT '' COMMENT '组件路径',
    `page_perm` varchar(128) DEFAULT '' COMMENT '页面权限标识',
    `perms` varchar(128) DEFAULT '' COMMENT '权限标识',
    `type` char(1) DEFAULT 'M' COMMENT '菜单类型 (M目录 C菜单 F按钮)',
    `icon` varchar(128) DEFAULT '' COMMENT '图标',
    `route_name` varchar(128) DEFAULT '' COMMENT '路由名称',
    `module` varchar(64) DEFAULT 'system' COMMENT '菜单归属模块',
    `sort` int DEFAULT '0' COMMENT '显示顺序',
    `is_visible` tinyint DEFAULT '1' COMMENT '是否可见 (1:是, 0:否)',
    `is_cache` tinyint DEFAULT '0' COMMENT '是否缓存 (1:是, 0:否)',
    `is_external` tinyint DEFAULT '0' COMMENT '是否外链 (1:是, 0:否)',
    `active_menu` varchar(255) DEFAULT '' COMMENT '高亮菜单路径',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='菜单权限表';

-- 4. 部门表
CREATE TABLE `system_dept` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `parent_id` bigint DEFAULT '0' COMMENT '父部门ID',
    `ancestors` varchar(255) DEFAULT '' COMMENT '祖级列表',
    `is_root` tinyint DEFAULT '0' COMMENT '是否组织根节点 (1:是, 0:否)',
    `dept_name` varchar(64) NOT NULL COMMENT '部门名称',
    `sort` int DEFAULT '0' COMMENT '显示顺序',
    `leader` varchar(64) DEFAULT NULL COMMENT '负责人',
    `phone` varchar(20) DEFAULT NULL COMMENT '联系电话',
    `email` varchar(128) DEFAULT NULL COMMENT '邮箱',
    `status` tinyint DEFAULT '1' COMMENT '部门状态 (1:正常, 2:停用)',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    `deleted_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='部门表';

-- 5. 岗位表
CREATE TABLE `system_post` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `dept_id` bigint unsigned DEFAULT '0' COMMENT '所属部门ID',
    `post_code` varchar(64) NOT NULL COMMENT '岗位编码',
    `post_name` varchar(64) NOT NULL COMMENT '岗位名称',
    `sort` int DEFAULT '0' COMMENT '显示顺序',
    `status` tinyint DEFAULT '1' COMMENT '状态 (1:正常, 2:停用)',
    `remark` varchar(255) DEFAULT NULL COMMENT '备注',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    `deleted_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_post_code` (`post_code`),
    KEY `idx_post_dept_id` (`dept_id`),
    KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='岗位表';

-- 6. 国际化翻译表
CREATE TABLE `system_i18n` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `lang_key` varchar(128) NOT NULL COMMENT '翻译键名',
    `lang_type` varchar(10) NOT NULL COMMENT '语言类型 (zh-CN, en-US)',
    `lang_value` text COMMENT '翻译内容',
    `module` varchar(64) DEFAULT 'system' COMMENT '所属模块',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_key_lang` (`lang_key`, `lang_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='国际化翻译表';

-- 6.1 系统设置表
CREATE TABLE `system_setting` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `setting_key` varchar(128) NOT NULL COMMENT '配置键',
    `setting_value` text COMMENT '配置值',
    `value_type` varchar(16) NOT NULL DEFAULT 'string' COMMENT '值类型',
    `group_key` varchar(32) NOT NULL COMMENT '配置分组',
    `module` varchar(64) NOT NULL DEFAULT 'system' COMMENT '模块归属',
    `is_public` tinyint DEFAULT '0' COMMENT '是否公开可读 (1:是, 0:否)',
    `is_encrypted` tinyint DEFAULT '0' COMMENT '是否加密存储 (1:是, 0:否)',
    `remark` varchar(255) DEFAULT NULL COMMENT '备注 / i18n key',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_setting_key` (`setting_key`),
    KEY `idx_setting_group_key` (`group_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统设置表';

-- 6.2 字典类型表
CREATE TABLE `system_dict_type` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `dict_code` varchar(64) NOT NULL COMMENT '字典编码',
    `dict_name` varchar(64) NOT NULL COMMENT '字典名称',
    `module` varchar(64) NOT NULL DEFAULT 'system' COMMENT '模块归属',
    `status` tinyint DEFAULT '1' COMMENT '状态 (1:正常, 2:禁用)',
    `remark` varchar(255) DEFAULT NULL COMMENT '备注',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    `deleted_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_dict_code` (`dict_code`),
    KEY `idx_dict_type_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='字典类型表';

-- 6.3 字典项表
CREATE TABLE `system_dict_item` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `dict_code` varchar(64) NOT NULL COMMENT '字典编码',
    `item_label_key` varchar(128) NOT NULL COMMENT '展示文案 i18n key',
    `item_value` varchar(64) NOT NULL COMMENT '字典值',
    `item_color` varchar(32) DEFAULT NULL COMMENT '标签颜色',
    `sort` int DEFAULT '0' COMMENT '排序',
    `status` tinyint DEFAULT '1' COMMENT '状态 (1:正常, 2:禁用)',
    `remark` varchar(255) DEFAULT NULL COMMENT '备注',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    `deleted_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_dict_item_code_value` (`dict_code`, `item_value`),
    KEY `idx_dict_item_code_sort` (`dict_code`, `sort`),
    KEY `idx_dict_item_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='字典项表';

-- 7. 登录日志表
CREATE TABLE `system_log_login` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `username` varchar(64) DEFAULT '' COMMENT '用户名',
    `ipaddr` varchar(128) DEFAULT '' COMMENT '登录IP',
    `login_location` varchar(255) DEFAULT '' COMMENT '登录地点',
    `browser` varchar(128) DEFAULT '' COMMENT '浏览器',
    `os` varchar(128) DEFAULT '' COMMENT '操作系统',
    `status` tinyint DEFAULT '1' COMMENT '登录状态 (1:成功, 0:失败)',
    `msg` varchar(255) DEFAULT '' COMMENT '提示消息',
    `login_time` datetime(3) DEFAULT NULL COMMENT '访问时间',
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统登录日志';

-- 8. 操作日志表
CREATE TABLE `system_log_oper` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `title` varchar(64) DEFAULT '' COMMENT '模块标题',
    `business_type` int DEFAULT '0' COMMENT '业务类型',
    `method` varchar(128) DEFAULT '' COMMENT '方法名称',
    `oper_name` varchar(64) DEFAULT '' COMMENT '操作人员',
    `oper_url` varchar(255) DEFAULT '' COMMENT '请求URL',
    `oper_ip` varchar(128) DEFAULT '' COMMENT '主机地址',
    `oper_param` text COMMENT '请求参数',
    `json_result` text COMMENT '返回参数',
    `status` int DEFAULT '1' COMMENT '操作状态 (1:正常, 2:异常)',
    `error_msg` text COMMENT '错误消息',
    `oper_time` datetime(3) DEFAULT NULL,
    `cost_time` bigint DEFAULT '0' COMMENT '消耗时间(ms)',
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='操作日志记录';

-- 9. 用户与角色关联表 (多对多)
CREATE TABLE `system_user_role` (
    `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
    `role_id` bigint unsigned NOT NULL COMMENT '角色ID',
    PRIMARY KEY (`user_id`, `role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户和角色关联表';

-- 10. 角色与菜单关联表 (多对多)
CREATE TABLE `system_role_menu` (
    `role_id` bigint unsigned NOT NULL COMMENT '角色ID',
    `menu_id` bigint unsigned NOT NULL COMMENT '菜单ID',
    PRIMARY KEY (`role_id`, `menu_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色和菜单关联表';

-- 11. 角色与页面/操作权限关联表 (多对多)
CREATE TABLE `system_role_permission` (
    `role_id` bigint unsigned NOT NULL COMMENT '角色ID',
    `permission_key` varchar(128) NOT NULL COMMENT '权限标识',
    PRIMARY KEY (`role_id`, `permission_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色和权限点关联表';

-- 12. 用户会话表 (refresh token 轮换)
CREATE TABLE `system_user_session` (
    `session_id` varchar(64) NOT NULL COMMENT '会话ID',
    `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
    `refresh_jti` varchar(64) NOT NULL COMMENT '当前 refresh token ID',
    `refresh_expires_at` datetime(3) NOT NULL COMMENT 'refresh token 过期时间',
    `last_refresh_at` datetime(3) DEFAULT NULL COMMENT '最后刷新时间',
    `last_ip` varchar(128) DEFAULT '' COMMENT '最后访问 IP',
    `user_agent` varchar(255) DEFAULT '' COMMENT '客户端 UA',
    `revoked_at` datetime(3) DEFAULT NULL COMMENT '吊销时间',
    `created_at` datetime(3) DEFAULT NULL,
    `updated_at` datetime(3) DEFAULT NULL,
    PRIMARY KEY (`session_id`),
    KEY `idx_system_user_session_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统用户会话表';

-- 13. Casbin 策略表
CREATE TABLE `casbin_rule` (
    `id` bigint unsigned NOT NULL AUTO_INCREMENT,
    `ptype` varchar(100) NOT NULL,
    `v0` varchar(100) DEFAULT '',
    `v1` varchar(100) DEFAULT '',
    `v2` varchar(100) DEFAULT '',
    `v3` varchar(100) DEFAULT '',
    `v4` varchar(100) DEFAULT '',
    `v5` varchar(100) DEFAULT '',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_casbin_rule` (`ptype`, `v0`, `v1`, `v2`, `v3`, `v4`, `v5`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Casbin 策略表';

-- 初始数据：admin 账号由后端迁移创建。开发环境默认密码为 123456；生产环境必须设置 PANTHEON_INITIAL_ADMIN_PASSWORD。
INSERT INTO `system_role` (role_name, role_key, sort, status, created_at) VALUES ('超级管理员', 'admin', 1, 1, NOW());
INSERT INTO `system_user_role` (user_id, role_id) VALUES (1, 1);

-- 初始菜单与权限
INSERT INTO `system_menu` (id, parent_id, title_key, path, component, page_perm, perms, type, icon, route_name, module, sort, is_visible, is_cache, is_external, active_menu, created_at) VALUES
(1, 0, 'system.menu.dashboard', '/dashboard', 'dashboard', 'platform:dashboard:view', '', 'C', 'dashboard', 'dashboard', 'platform', 10, 1, 0, 0, '', NOW()),
(2, 0, 'system.menu.access', '/system/access', '', '', '', 'M', 'apps', 'system-access', 'system.iam', 20, 1, 0, 0, '', NOW()),
(3, 0, 'system.menu.org', '/system/org', '', '', '', 'M', 'storage', 'system-org', 'system.org', 30, 1, 0, 0, '', NOW()),
(4, 0, 'system.menu.config', '/system/config', '', '', '', 'M', 'settings', 'system-config', 'system.config', 40, 1, 0, 0, '', NOW()),
(5, 0, 'system.menu.security', '/system/security', '', '', '', 'M', 'safe', 'system-security', 'system.auth', 50, 1, 0, 0, '', NOW()),
(10, 2, 'system.menu.user', '/system/user', 'system/user/UserList', 'system:user:list', '', 'C', 'user', 'system-user', 'system.iam', 10, 1, 0, 0, '', NOW()),
(11, 2, 'system.menu.role', '/system/role', 'system/role/RoleList', 'system:role:list', '', 'C', 'safe', 'system-role', 'system.iam', 20, 1, 0, 0, '', NOW()),
(12, 2, 'system.menu.permission', '/system/permission', 'system/permission/PermissionList', 'system:permission:list', '', 'C', 'safe', 'system-permission', 'system.iam', 30, 1, 0, 0, '', NOW()),
(13, 2, 'system.menu.menu', '/system/menu', 'system/menu/MenuList', 'system:menu:list', '', 'C', 'menu', 'system-menu', 'system.iam', 40, 1, 0, 0, '', NOW()),
(20, 3, 'system.menu.dept', '/system/dept', 'system/dept/DeptList', 'system:dept:list', '', 'C', 'storage', 'system-dept', 'system.org', 10, 1, 0, 0, '', NOW()),
(21, 3, 'system.menu.post', '/system/post', 'system/post/PostList', 'system:post:list', '', 'C', 'storage', 'system-post', 'system.org', 20, 1, 0, 0, '', NOW()),
(30, 4, 'system.menu.dict', '/system/dict', 'system/dict/DictPage', 'system:dict:list', '', 'C', 'list', 'system-dict', 'system.config', 10, 1, 1, 0, '', NOW()),
(31, 4, 'system.menu.setting', '/system/setting', 'system/setting/SettingPage', 'system:setting:list', '', 'C', 'settings', 'system-setting', 'system.config', 20, 1, 1, 0, '', NOW()),
(40, 5, 'system.menu.loginLog', '/system/login-log', 'auth/LoginLogList', 'system:login-log:list', '', 'C', 'safe', 'system-login-log', 'system.auth', 10, 1, 0, 0, '', NOW()),
(41, 5, 'system.menu.session', '/system/session', 'auth/SessionList', 'system:session:list', '', 'C', 'safe', 'system-session', 'system.auth', 20, 1, 0, 0, '', NOW()),
(42, 5, 'system.menu.operationLog', '/system/operation-log', 'system/audit/OperationLogList', 'system:operation-log:list', '', 'C', 'safe', 'system-operation-log', 'system.audit', 30, 1, 0, 0, '', NOW()),
(100, 10, 'system.permission.user.view', '', '', '', 'system:user:view', 'F', '', '', 'system.iam', 1, 1, 0, 0, '', NOW()),
(101, 10, 'system.permission.user.create', '', '', '', 'system:user:create', 'F', '', '', 'system.iam', 2, 1, 0, 0, '', NOW()),
(102, 10, 'system.permission.user.update', '', '', '', 'system:user:update', 'F', '', '', 'system.iam', 3, 1, 0, 0, '', NOW()),
(103, 10, 'system.permission.user.delete', '', '', '', 'system:user:delete', 'F', '', '', 'system.iam', 4, 1, 0, 0, '', NOW()),
(104, 10, 'system.permission.user.reset', '', '', '', 'system:user:reset', 'F', '', '', 'system.iam', 5, 1, 0, 0, '', NOW()),
(105, 10, 'system.permission.user.export', '', '', '', 'system:user:export', 'F', '', '', 'system.iam', 6, 1, 0, 0, '', NOW()),
(106, 10, 'system.permission.user.import', '', '', '', 'system:user:import', 'F', '', '', 'system.iam', 7, 1, 0, 0, '', NOW()),
(107, 10, 'system.permission.user.batch_update', '', '', '', 'system:user:batch-update', 'F', '', '', 'system.iam', 8, 1, 0, 0, '', NOW()),
(110, 11, 'system.permission.role.create', '', '', '', 'system:role:create', 'F', '', '', 'system.iam', 1, 1, 0, 0, '', NOW()),
(111, 11, 'system.permission.role.update', '', '', '', 'system:role:update', 'F', '', '', 'system.iam', 2, 1, 0, 0, '', NOW()),
(112, 11, 'system.permission.role.delete', '', '', '', 'system:role:delete', 'F', '', '', 'system.iam', 3, 1, 0, 0, '', NOW()),
(113, 11, 'system.permission.role.batch_update', '', '', '', 'system:role:batch-update', 'F', '', '', 'system.iam', 4, 1, 0, 0, '', NOW()),
(114, 11, 'system.permission.role.export', '', '', '', 'system:role:export', 'F', '', '', 'system.iam', 5, 1, 0, 0, '', NOW()),
(120, 12, 'system.permission.policy.create', '', '', '', 'system:permission:create', 'F', '', '', 'system.iam', 1, 1, 0, 0, '', NOW()),
(121, 12, 'system.permission.policy.update', '', '', '', 'system:permission:update', 'F', '', '', 'system.iam', 2, 1, 0, 0, '', NOW()),
(122, 12, 'system.permission.policy.delete', '', '', '', 'system:permission:delete', 'F', '', '', 'system.iam', 3, 1, 0, 0, '', NOW()),
(123, 12, 'system.permission.policy.export', '', '', '', 'system:permission:export', 'F', '', '', 'system.iam', 4, 1, 0, 0, '', NOW()),
(124, 12, 'system.permission.policy.import', '', '', '', 'system:permission:import', 'F', '', '', 'system.iam', 5, 1, 0, 0, '', NOW()),
(130, 13, 'system.permission.menu.create', '', '', '', 'system:menu:create', 'F', '', '', 'system.iam', 1, 1, 0, 0, '', NOW()),
(131, 13, 'system.permission.menu.update', '', '', '', 'system:menu:update', 'F', '', '', 'system.iam', 2, 1, 0, 0, '', NOW()),
(132, 13, 'system.permission.menu.delete', '', '', '', 'system:menu:delete', 'F', '', '', 'system.iam', 3, 1, 0, 0, '', NOW()),
(140, 20, 'system.permission.dept.create', '', '', '', 'system:dept:create', 'F', '', '', 'system.org', 1, 1, 0, 0, '', NOW()),
(141, 20, 'system.permission.dept.update', '', '', '', 'system:dept:update', 'F', '', '', 'system.org', 2, 1, 0, 0, '', NOW()),
(142, 20, 'system.permission.dept.delete', '', '', '', 'system:dept:delete', 'F', '', '', 'system.org', 3, 1, 0, 0, '', NOW()),
(143, 20, 'system.permission.dept.export', '', '', '', 'system:dept:export', 'F', '', '', 'system.org', 4, 1, 0, 0, '', NOW()),
(144, 20, 'system.permission.dept.import', '', '', '', 'system:dept:import', 'F', '', '', 'system.org', 5, 1, 0, 0, '', NOW()),
(145, 20, 'system.permission.dept.batch_update', '', '', '', 'system:dept:batch-update', 'F', '', '', 'system.org', 6, 1, 0, 0, '', NOW()),
(150, 21, 'system.permission.post.create', '', '', '', 'system:post:create', 'F', '', '', 'system.org', 1, 1, 0, 0, '', NOW()),
(151, 21, 'system.permission.post.update', '', '', '', 'system:post:update', 'F', '', '', 'system.org', 2, 1, 0, 0, '', NOW()),
(152, 21, 'system.permission.post.delete', '', '', '', 'system:post:delete', 'F', '', '', 'system.org', 3, 1, 0, 0, '', NOW()),
(153, 21, 'system.permission.post.export', '', '', '', 'system:post:export', 'F', '', '', 'system.org', 4, 1, 0, 0, '', NOW()),
(154, 21, 'system.permission.post.import', '', '', '', 'system:post:import', 'F', '', '', 'system.org', 5, 1, 0, 0, '', NOW()),
(155, 21, 'system.permission.post.batch_update', '', '', '', 'system:post:batch-update', 'F', '', '', 'system.org', 6, 1, 0, 0, '', NOW()),
(160, 30, 'system.permission.dict.create', '', '', '', 'system:dict:create', 'F', '', '', 'system.config', 1, 1, 0, 0, '', NOW()),
(161, 30, 'system.permission.dict.update', '', '', '', 'system:dict:update', 'F', '', '', 'system.config', 2, 1, 0, 0, '', NOW()),
(162, 30, 'system.permission.dict.delete', '', '', '', 'system:dict:delete', 'F', '', '', 'system.config', 3, 1, 0, 0, '', NOW()),
(163, 30, 'system.permission.dict.refresh', '', '', '', 'system:dict:refresh', 'F', '', '', 'system.config', 4, 1, 0, 0, '', NOW()),
(164, 30, 'system.permission.dict.export', '', '', '', 'system:dict:export', 'F', '', '', 'system.config', 5, 1, 0, 0, '', NOW()),
(165, 30, 'system.permission.dict.import', '', '', '', 'system:dict:import', 'F', '', '', 'system.config', 6, 1, 0, 0, '', NOW()),
(170, 31, 'system.permission.setting.update', '', '', '', 'system:setting:update', 'F', '', '', 'system.config', 1, 1, 0, 0, '', NOW()),
(171, 31, 'system.permission.setting.refresh', '', '', '', 'system:setting:refresh', 'F', '', '', 'system.config', 2, 1, 0, 0, '', NOW()),
(180, 40, 'system.permission.login_log.export', '', '', '', 'system:login-log:export', 'F', '', '', 'system.auth', 1, 1, 0, 0, '', NOW()),
(181, 41, 'system.permission.session.delete', '', '', '', 'system:session:delete', 'F', '', '', 'system.auth', 1, 1, 0, 0, '', NOW()),
(190, 42, 'system.permission.operation_log.delete', '', '', '', 'system:operation-log:delete', 'F', '', '', 'system.audit', 1, 1, 0, 0, '', NOW()),
(191, 42, 'system.permission.operation_log.clear', '', '', '', 'system:operation-log:clear', 'F', '', '', 'system.audit', 2, 1, 0, 0, '', NOW()),
(192, 42, 'system.permission.operation_log.export', '', '', '', 'system:operation-log:export', 'F', '', '', 'system.audit', 3, 1, 0, 0, '', NOW());

INSERT INTO `system_role_menu` (role_id, menu_id) VALUES
(1, 1), (1, 2), (1, 3), (1, 4), (1, 5),
(1, 10), (1, 11), (1, 12), (1, 13), (1, 20), (1, 21), (1, 30), (1, 31), (1, 40), (1, 41), (1, 42);

INSERT INTO `system_role_permission` (role_id, permission_key) VALUES
(1, 'platform:dashboard:view'),
(1, 'system:user:list'), (1, 'system:user:view'), (1, 'system:user:create'), (1, 'system:user:update'), (1, 'system:user:delete'), (1, 'system:user:reset'), (1, 'system:user:export'), (1, 'system:user:import'), (1, 'system:user:batch-update'),
(1, 'system:role:list'), (1, 'system:role:create'), (1, 'system:role:update'), (1, 'system:role:delete'), (1, 'system:role:batch-update'), (1, 'system:role:export'),
(1, 'system:permission:list'), (1, 'system:permission:create'), (1, 'system:permission:update'), (1, 'system:permission:delete'), (1, 'system:permission:export'), (1, 'system:permission:import'),
(1, 'system:menu:list'), (1, 'system:menu:create'), (1, 'system:menu:update'), (1, 'system:menu:delete'),
(1, 'system:dept:list'), (1, 'system:dept:create'), (1, 'system:dept:update'), (1, 'system:dept:delete'), (1, 'system:dept:export'), (1, 'system:dept:import'), (1, 'system:dept:batch-update'),
(1, 'system:post:list'), (1, 'system:post:create'), (1, 'system:post:update'), (1, 'system:post:delete'), (1, 'system:post:export'), (1, 'system:post:import'), (1, 'system:post:batch-update'),
(1, 'system:dict:list'), (1, 'system:dict:create'), (1, 'system:dict:update'), (1, 'system:dict:delete'), (1, 'system:dict:refresh'), (1, 'system:dict:export'), (1, 'system:dict:import'),
(1, 'system:setting:list'), (1, 'system:setting:update'), (1, 'system:setting:refresh'),
(1, 'system:login-log:list'), (1, 'system:login-log:export'),
(1, 'system:session:list'), (1, 'system:session:delete'),
(1, 'system:operation-log:list'), (1, 'system:operation-log:delete'), (1, 'system:operation-log:clear'), (1, 'system:operation-log:export');

-- 初始 Casbin 策略
INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`) VALUES
('p', 'admin', '/api/v1/*', 'GET'),
('p', 'admin', '/api/v1/*', 'POST'),
('p', 'admin', '/api/v1/*', 'PUT'),
('p', 'admin', '/api/v1/*', 'PATCH'),
('p', 'admin', '/api/v1/*', 'DELETE');

-- 初始多语言资源
INSERT INTO `system_i18n` (lang_key, lang_type, lang_value, module, created_at) VALUES
('app.name', 'zh-CN', 'Pantheon Base', 'system', NOW()),
('app.slogan', 'zh-CN', '赋能企业数字化', 'system', NOW()),
('system.menu.dashboard', 'zh-CN', '仪表盘', 'system', NOW()),
('system.menu.access', 'zh-CN', '访问控制', 'system', NOW()),
('system.menu.org', 'zh-CN', '组织管理', 'system', NOW()),
('system.menu.config', 'zh-CN', '平台配置', 'system', NOW()),
('system.menu.security', 'zh-CN', '安全与审计', 'system', NOW()),
('system.menu.user', 'zh-CN', '用户管理', 'system', NOW()),
('system.menu.role', 'zh-CN', '角色管理', 'system', NOW()),
('system.menu.dept', 'zh-CN', '部门管理', 'system', NOW()),
('system.menu.post', 'zh-CN', '岗位管理', 'system', NOW()),
('system.menu.permission', 'zh-CN', '权限管理', 'system', NOW()),
('system.menu.menu', 'zh-CN', '菜单管理', 'system', NOW()),
('system.menu.loginLog', 'zh-CN', '登录日志', 'system', NOW()),
('system.menu.session', 'zh-CN', '会话管理', 'system', NOW()),
('system.menu.setting', 'zh-CN', '系统设置', 'system', NOW()),
('system.menu.dict', 'zh-CN', '字典管理', 'system', NOW()),
('system.menu.operationLog', 'zh-CN', '操作日志', 'system', NOW()),
('app.name', 'en-US', 'Pantheon Base', 'system', NOW()),
('app.slogan', 'en-US', 'Empowering Enterprise Digitalization', 'system', NOW()),
('system.menu.dashboard', 'en-US', 'Dashboard', 'system', NOW()),
('system.menu.access', 'en-US', 'Access Control', 'system', NOW()),
('system.menu.org', 'en-US', 'Organization', 'system', NOW()),
('system.menu.config', 'en-US', 'Platform Config', 'system', NOW()),
('system.menu.security', 'en-US', 'Security & Audit', 'system', NOW()),
('system.menu.user', 'en-US', 'User Management', 'system', NOW()),
('system.menu.role', 'en-US', 'Role Management', 'system', NOW()),
('system.menu.dept', 'en-US', 'Department Management', 'system', NOW()),
('system.menu.post', 'en-US', 'Post Management', 'system', NOW()),
('system.menu.permission', 'en-US', 'Permission Management', 'system', NOW()),
('system.menu.menu', 'en-US', 'Menu Management', 'system', NOW()),
('system.menu.loginLog', 'en-US', 'Login Logs', 'system', NOW()),
('system.menu.session', 'en-US', 'Session Management', 'system', NOW()),
('system.menu.setting', 'en-US', 'System Settings', 'system', NOW()),
('system.menu.dict', 'en-US', 'Dictionary Management', 'system', NOW()),
('system.menu.operationLog', 'en-US', 'Operation Logs', 'system', NOW());

INSERT INTO `system_dept` (`parent_id`, `ancestors`, `is_root`, `dept_name`, `sort`, `status`, `created_at`, `updated_at`) VALUES
(0, '', 1, 'Pantheon Base', 0, 1, NOW(), NOW());

-- 初始系统设置
INSERT INTO `system_setting` (`setting_key`, `setting_value`, `value_type`, `group_key`, `module`, `is_public`, `is_encrypted`, `remark`, `created_at`, `updated_at`) VALUES
('site.name', 'Pantheon Base', 'string', 'basic', 'system', 1, 0, 'system.setting.remark.site.name', NOW(), NOW()),
('site.logo', '', 'string', 'basic', 'system', 1, 0, 'system.setting.remark.site.logo', NOW(), NOW()),
('security.password_min_length', '6', 'number', 'security', 'system', 0, 0, 'system.setting.remark.security.password_min_length', NOW(), NOW()),
('login.max_failed_attempts', '5', 'number', 'login', 'system', 0, 0, 'system.setting.remark.login.max_failed_attempts', NOW(), NOW()),
('login.lock_minutes', '15', 'number', 'login', 'system', 0, 0, 'system.setting.remark.login.lock_minutes', NOW(), NOW()),
('i18n.default_language', 'zh-CN', 'string', 'i18n', 'system', 1, 0, 'system.setting.remark.i18n.default_language', NOW(), NOW()),
('ui.default_theme', 'indigo', 'string', 'ui', 'system', 1, 0, 'system.setting.remark.ui.default_theme', NOW(), NOW()),
('ui.enable_tab_bar', 'true', 'boolean', 'ui', 'system', 1, 0, 'system.setting.remark.ui.enable_tab_bar', NOW(), NOW()),
('upload.storage_driver', 'local', 'string', 'upload', 'system', 0, 0, 'system.setting.remark.upload.storage_driver', NOW(), NOW()),
('upload.max_file_size', '20', 'number', 'upload', 'system', 0, 0, 'system.setting.remark.upload.max_file_size', NOW(), NOW()),
('upload.allowed_types', '["jpg","jpeg","png","pdf","doc","docx","xls","xlsx"]', 'json', 'upload', 'system', 0, 0, 'system.setting.remark.upload.allowed_types', NOW(), NOW()),
('upload.local_path', './uploads', 'string', 'upload', 'system', 0, 0, 'system.setting.remark.upload.local_path', NOW(), NOW()),
('upload.public_base_url', '', 'string', 'upload', 'system', 0, 0, 'system.setting.remark.upload.public_base_url', NOW(), NOW()),
('upload.s3_endpoint', '', 'string', 'upload', 'system', 0, 0, 'system.setting.remark.upload.s3_endpoint', NOW(), NOW()),
('upload.s3_bucket', '', 'string', 'upload', 'system', 0, 0, 'system.setting.remark.upload.s3_bucket', NOW(), NOW()),
('upload.s3_access_key_id', '', 'string', 'upload', 'system', 0, 1, 'system.setting.remark.upload.s3_access_key_id', NOW(), NOW()),
('upload.s3_secret_access_key', '', 'string', 'upload', 'system', 0, 1, 'system.setting.remark.upload.s3_secret_access_key', NOW(), NOW());

-- 初始字典
INSERT INTO `system_dict_type` (`dict_code`, `dict_name`, `module`, `status`, `remark`, `created_at`, `updated_at`) VALUES
('system_yes_no', 'system.dict.seed.system_yes_no', 'system', 1, 'system.dict.remark.system_yes_no', NOW(), NOW()),
('system_user_status', 'system.dict.seed.system_user_status', 'system', 1, 'system.dict.remark.system_user_status', NOW(), NOW());

INSERT INTO `system_dict_item` (`dict_code`, `item_label_key`, `item_value`, `item_color`, `sort`, `status`, `remark`, `created_at`, `updated_at`) VALUES
('system_yes_no', 'dict.system_yes_no.yes', '1', 'green', 1, 1, '', NOW(), NOW()),
('system_yes_no', 'dict.system_yes_no.no', '0', 'gray', 2, 1, '', NOW(), NOW()),
('system_user_status', 'dict.system_user_status.enabled', '1', 'green', 1, 1, '', NOW(), NOW()),
('system_user_status', 'dict.system_user_status.disabled', '2', 'red', 2, 1, '', NOW(), NOW());
