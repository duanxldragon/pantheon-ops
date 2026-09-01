# 权限点和菜单解耦实施指南

**版本**: 1.0  
**日期**: 2026-06-22  
**状态**: 实施中

---

## 背景

当前系统存在权限点和菜单混用的情况，导致：
1. 按钮权限依赖菜单权限
2. 菜单权限和接口权限耦合
3. 难以独立管理权限点

## 目标

实现权限点和菜单的完全解耦：
- 菜单控制导航显示
- 权限点控制功能访问（按钮、接口）
- 两者可独立管理和授权

---

## 权限点命名规范

### 格式

```
{domain}:{resource}:{action}
```

### 示例

| 权限点 | 说明 |
|--------|------|
| `system:user:view` | 查看用户 |
| `system:user:create` | 创建用户 |
| `system:user:update` | 更新用户 |
| `system:user:delete` | 删除用户 |
| `system:user:export` | 导出用户 |
| `system:user:import` | 导入用户 |
| `system:user:reset-password` | 重置密码 |
| `system:role:view` | 查看角色 |
| `system:role:assign` | 分配角色 |

### 域（Domain）

- `system` - 系统管理
- `auth` - 认证授权
- `business` - 业务功能

### 资源（Resource）

- `user` - 用户
- `role` - 角色
- `menu` - 菜单
- `dept` - 部门
- `dict` - 字典

### 动作（Action）

- `view` / `list` - 查看/列表
- `create` / `add` - 创建
- `update` / `edit` - 更新
- `delete` / `remove` - 删除
- `export` - 导出
- `import` - 导入
- `assign` - 分配

---

## 数据库设计

### 权限点表（已存在）

```sql
CREATE TABLE system_permission (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    permission_key VARCHAR(100) NOT NULL UNIQUE COMMENT '权限标识',
    permission_name VARCHAR(100) NOT NULL COMMENT '权限名称',
    resource_type VARCHAR(50) COMMENT '资源类型',
    description TEXT COMMENT '描述',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_permission_key (permission_key)
);
```

### 角色权限关联表

```sql
CREATE TABLE system_role_permission (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    role_id BIGINT UNSIGNED NOT NULL,
    permission_id BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_role_permission (role_id, permission_id),
    INDEX idx_role_id (role_id),
    INDEX idx_permission_id (permission_id)
);
```

---

## 后端实现

### 1. 权限检查中间件

```go
// backend/internal/middleware/permission_middleware.go
package middleware

import (
    "github.com/gin-gonic/gin"
    "pantheon-platform/backend/pkg/common"
)

// RequirePermission 检查是否拥有指定权限
func RequirePermission(permission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 获取用户权限列表
        perms, exists := c.Get("permissions")
        if !exists {
            common.Fail(c, common.CodeForbidden, "permission.denied")
            c.Abort()
            return
        }

        permissions, ok := perms.([]string)
        if !ok {
            common.Fail(c, common.CodeForbidden, "permission.invalid")
            c.Abort()
            return
        }

        // 检查权限
        if !hasPermission(permissions, permission) {
            common.Fail(c, common.CodeForbidden, "permission.denied")
            c.Abort()
            return
        }

        c.Next()
    }
}

// RequireAnyPermission 检查是否拥有任一权限
func RequireAnyPermission(permissions ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userPerms, exists := c.Get("permissions")
        if !exists {
            common.Fail(c, common.CodeForbidden, "permission.denied")
            c.Abort()
            return
        }

        userPermissions, ok := userPerms.([]string)
        if !ok {
            common.Fail(c, common.CodeForbidden, "permission.invalid")
            c.Abort()
            return
        }

        // 检查是否拥有任一权限
        for _, perm := range permissions {
            if hasPermission(userPermissions, perm) {
                c.Next()
                return
            }
        }

        common.Fail(c, common.CodeForbidden, "permission.denied")
        c.Abort()
    }
}

func hasPermission(userPermissions []string, permission string) bool {
    for _, p := range userPermissions {
        if p == permission || p == "*:*:*" { // 超级管理员
            return true
        }
    }
    return false
}
```

### 2. 路由应用权限

```go
// backend/modules/system/iam/user/routes.go
func RegisterRoutes(r *gin.RouterGroup) {
    users := r.Group("/users")
    {
        // 查看用户列表 - 需要 system:user:view 权限
        users.GET("", 
            middleware.RequirePermission("system:user:view"),
            handler.ListUsers,
        )

        // 创建用户 - 需要 system:user:create 权限
        users.POST("",
            middleware.RequirePermission("system:user:create"),
            handler.CreateUser,
        )

        // 更新用户 - 需要 system:user:update 权限
        users.PUT("/:id",
            middleware.RequirePermission("system:user:update"),
            handler.UpdateUser,
        )

        // 删除用户 - 需要 system:user:delete 权限
        users.DELETE("/:id",
            middleware.RequirePermission("system:user:delete"),
            middleware.SecureActionMiddleware(),
            handler.DeleteUser,
        )

        // 导出用户 - 需要 system:user:export 权限
        users.POST("/export",
            middleware.RequirePermission("system:user:export"),
            handler.ExportUsers,
        )

        // 批量删除 - 需要 system:user:delete 权限
        users.POST("/batch-delete",
            middleware.RequirePermission("system:user:delete"),
            middleware.SecureActionMiddleware(),
            handler.BatchDeleteUsers,
        )
    }
}
```

---

## 前端实现

### 1. 权限指令

```typescript
// frontend/src/directives/permission.ts
import { DirectiveBinding } from 'vue';
import { useAuthStore } from '@/store/useAuthStore';

export const permission = {
  mounted(el: HTMLElement, binding: DirectiveBinding) {
    const authStore = useAuthStore();
    const { value } = binding;

    if (!value) {
      throw new Error('Permission directive requires a value');
    }

    const permissions = Array.isArray(value) ? value : [value];
    const hasPermission = permissions.some(perm => 
      authStore.permissions.includes(perm) || 
      authStore.permissions.includes('*:*:*')
    );

    if (!hasPermission) {
      el.remove();
    }
  }
};

// 全局注册
app.directive('permission', permission);
```

### 2. 使用权限指令

```tsx
// frontend/src/modules/system/user/UserList.tsx
import React from 'react';
import { Button } from '@arco-design/web-react';
import { useAuthStore } from '@/store/useAuthStore';

export function UserList() {
  const { hasPermission } = useAuthStore();

  return (
    <div>
      <div className="actions">
        {/* 创建按钮 - 需要 system:user:create 权限 */}
        {hasPermission('system:user:create') && (
          <Button type="primary" onClick={handleCreate}>
            创建用户
          </Button>
        )}

        {/* 导出按钮 - 需要 system:user:export 权限 */}
        {hasPermission('system:user:export') && (
          <Button onClick={handleExport}>
            导出
          </Button>
        )}
      </div>

      <Table
        columns={[
          // ... other columns
          {
            title: '操作',
            render: (_, record) => (
              <>
                {/* 编辑 - 需要 system:user:update 权限 */}
                {hasPermission('system:user:update') && (
                  <Button size="small" onClick={() => handleEdit(record)}>
                    编辑
                  </Button>
                )}

                {/* 删除 - 需要 system:user:delete 权限 */}
                {hasPermission('system:user:delete') && (
                  <Button size="small" status="danger" onClick={() => handleDelete(record)}>
                    删除
                  </Button>
                )}
              </>
            )
          }
        ]}
      />
    </div>
  );
}
```

### 3. 权限 Hook

```typescript
// frontend/src/hooks/usePermission.ts
import { useAuthStore } from '@/store/useAuthStore';

export function usePermission() {
  const authStore = useAuthStore();

  const hasPermission = (permission: string | string[]): boolean => {
    const permissions = Array.isArray(permission) ? permission : [permission];
    return permissions.some(perm => 
      authStore.permissions.includes(perm) || 
      authStore.permissions.includes('*:*:*')
    );
  };

  const hasAllPermissions = (permissions: string[]): boolean => {
    return permissions.every(perm => hasPermission(perm));
  };

  const hasAnyPermission = (permissions: string[]): boolean => {
    return permissions.some(perm => hasPermission(perm));
  };

  return {
    hasPermission,
    hasAllPermissions,
    hasAnyPermission,
  };
}
```

---

## 数据迁移

### 从菜单权限迁移到权限点

```sql
-- 1. 为每个菜单创建对应的权限点
INSERT INTO system_permission (permission_key, permission_name, resource_type, description)
SELECT 
    CONCAT('system:', LOWER(REPLACE(menu_name, ' ', '-')), ':view'),
    CONCAT('查看', menu_name),
    'menu',
    menu_name
FROM system_menu
WHERE menu_type = 1; -- 只迁移菜单类型

-- 2. 创建按钮权限点
INSERT INTO system_permission (permission_key, permission_name, resource_type, description)
VALUES
('system:user:create', '创建用户', 'button', '用户管理-创建按钮'),
('system:user:update', '更新用户', 'button', '用户管理-编辑按钮'),
('system:user:delete', '删除用户', 'button', '用户管理-删除按钮'),
('system:user:export', '导出用户', 'button', '用户管理-导出按钮');

-- 3. 为角色分配权限点（基于菜单权限）
INSERT INTO system_role_permission (role_id, permission_id)
SELECT 
    rm.role_id,
    p.id
FROM system_role_menu rm
JOIN system_menu m ON rm.menu_id = m.id
JOIN system_permission p ON p.permission_key = CONCAT('system:', LOWER(REPLACE(m.menu_name, ' ', '-')), ':view')
WHERE NOT EXISTS (
    SELECT 1 FROM system_role_permission rp 
    WHERE rp.role_id = rm.role_id AND rp.permission_id = p.id
);
```

---

## 测试清单

### 后端测试

- [ ] 权限检查中间件单元测试
- [ ] 权限缺失时返回 403
- [ ] 超级管理员拥有所有权限
- [ ] 权限列表正确加载

### 前端测试

- [ ] 无权限时按钮不显示
- [ ] 有权限时按钮正常显示
- [ ] 权限指令正确工作
- [ ] 权限 Hook 正确返回结果

### 集成测试

- [ ] 角色授权后权限生效
- [ ] 权限撤销后按钮消失
- [ ] 接口调用遵循权限控制

---

## 实施计划

### 第一阶段（1-2天）
- [x] 权限检查中间件实现
- [ ] 核心模块路由应用权限
- [ ] 前端权限 Hook 实现

### 第二阶段（2-3天）
- [ ] 所有模块路由应用权限
- [ ] 前端所有按钮应用权限
- [ ] 数据迁移脚本测试

### 第三阶段（1天）
- [ ] 集成测试
- [ ] 文档更新
- [ ] 生产环境迁移

---

## 注意事项

1. **向后兼容**: 迁移期间保留菜单权限检查，逐步替换
2. **超级管理员**: 始终拥有 `*:*:*` 权限
3. **性能**: 权限列表加载时缓存，避免频繁查询
4. **审计**: 记录权限变更日志

---

**文档版本**: 1.0  
**最后更新**: 2026-06-22  
**负责人**: 后端团队 + 前端团队
