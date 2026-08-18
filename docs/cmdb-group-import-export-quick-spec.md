# CMDB Group 导入导出快速实施指令

**模块**: 主机分组 (CMDB Group)  
**复杂度**: 中等（树形结构）  
**预计时间**: 0.5天

## 参考文件
- 后端参考: `backend/modules/business/cmdb/host/host_import_export.go`
- 前端参考: `frontend/src/modules/business/cmdb/host/CmdbHostList.tsx`
- 树形路径参考: `backend/pkg/impexp/import_helpers.go` 中的 `ResolveDeptPath`

## CSV 字段定义

```csv
name,parentPath,description,conditions
```

**字段说明**:
- `name` (必填): 分组名称，如 "生产环境"
- `parentPath`: 父分组路径，如 "/根分组/一级分组"，空表示根级分组
- `description`: 描述
- `conditions`: 条件表达式（JSON字符串），如 `[{"field":"environment","op":"eq","value":"prod"}]`

**特殊说明**:
- `parentPath` 使用 `/` 分隔的路径表示父子关系
- 导入时会自动解析父分组路径并关联到 `parent_id`
- 如果父路径不存在，会报错（不自动创建父分组）
- 唯一性判断：同一父分组下，分组名称唯一

## 需要修改的文件

### 后端 (5个文件)

1. **创建 `backend/modules/business/cmdb/group/group_import_export.go`**
   - 复制 `host/host_import_export.go` 的结构
   - 添加父路径解析逻辑（参考 `impexp.ResolveDeptPath`）
   - CSV 结构体：
     ```go
     type GroupImportRow struct {
         Name        string `csv:"name"`
         ParentPath  string `csv:"parentPath"`
         Description string `csv:"description"`
         Conditions  string `csv:"conditions"` // JSON string
     }
     ```
   - 唯一性判断：
     ```go
     // 同一父分组下，name 唯一
     db.Where("parent_id = ? AND name = ?", parentID, name).First(&existing)
     ```

2. **修改 `backend/modules/business/cmdb/group/group_handler.go`**
   - 添加 3 个路由和 Handler

3. **修改 `backend/modules/business/cmdb/group/group_service.go`**
   - 添加辅助方法：
     ```go
     // GetByParentAndName 根据父ID和名称查找分组
     func (s *GroupService) GetByParentAndName(parentID *uint, name string) (*Group, error)
     
     // BuildGroupPath 构建分组路径（用于导出）
     func (s *GroupService) BuildGroupPath(groupID uint) (string, error)
     ```

4. **修改 `backend/modules/business/cmdb/module.go`**
   - 添加 2 个权限：`business:cmdb:group:export`, `business:cmdb:group:import`

### 前端 (3个文件)

5. **修改 `frontend/src/modules/business/cmdb/group/api.ts`**
   - 添加 3 个函数

6. **修改 `frontend/src/modules/business/cmdb/group/CmdbGroupList.tsx`**
   - 添加导入导出按钮和处理函数

7. **修改国际化文件**
   - 添加中英文错误信息

## 核心逻辑：父路径解析

### 导出时
```go
func (s *GroupService) BuildGroupPath(groupID uint) (string, error) {
    var parts []string
    currentID := groupID
    
    for {
        var group Group
        if err := s.db.First(&group, currentID).Error; err != nil {
            return "", err
        }
        parts = append([]string{group.Name}, parts...) // 前插
        
        if group.ParentID == nil {
            break
        }
        currentID = *group.ParentID
    }
    
    return "/" + strings.Join(parts, "/"), nil
}
```

### 导入时
```go
func resolveParentID(db *gorm.DB, parentPath string) (*uint, error) {
    if parentPath == "" || parentPath == "/" {
        return nil, nil // 根级分组
    }
    
    parts := strings.Split(strings.Trim(parentPath, "/"), "/")
    var currentParentID *uint = nil
    
    for _, name := range parts {
        var group Group
        query := db.Where("name = ?", name)
        if currentParentID != nil {
            query = query.Where("parent_id = ?", *currentParentID)
        } else {
            query = query.Where("parent_id IS NULL")
        }
        
        if err := query.First(&group).Error; err != nil {
            return nil, fmt.Errorf("父分组路径不存在: %s", parentPath)
        }
        currentParentID = &group.ID
    }
    
    return currentParentID, nil
}
```

## 验证逻辑

### 必填字段
- `name` (必填)

### 业务逻辑
- 如果 `parentPath` 非空，必须能解析到已存在的父分组
- 同一父分组下，分组名称不能重复
- `conditions` 如果非空，必须是有效的 JSON 数组

### 唯一性
- 组合唯一：`(parent_id, name)`
- 导入时：
  - 如果 `(parent_id, name)` 已存在 → 更新
  - 如果不存在 → 创建

## CSV 示例

```csv
name,parentPath,description,conditions
根分组,,所有主机的根分组,
生产环境,/根分组,生产环境主机,"[{""field"":""environment"",""op"":""eq"",""value"":""prod""}]"
数据库服务器,/根分组/生产环境,生产环境数据库,"[{""field"":""tags"",""op"":""contains"",""value"":""database""}]"
```

## 实施步骤

1. 复制 `host/host_import_export.go` 的完整结构
2. 添加父路径解析辅助函数
3. 修改 CSV 字段映射和验证逻辑
4. 导出时：递归构建 `parentPath`
5. 导入时：解析 `parentPath` 为 `parent_id`
6. 修改 Handler、权限、前端 API
7. 测试树形结构的导入导出

## Codex 执行指令（待 BizScope 完成后使用）

```bash
codex exec "为 CMDB Group (主机分组) 模块添加导入导出功能。

特殊点：树形结构，使用 parentPath 表示父子关系

参考: backend/modules/business/cmdb/host/host_import_export.go
参考: backend/pkg/impexp/import_helpers.go (ResolveDeptPath)

CSV字段: name,parentPath,description,conditions
唯一性: (parent_id, name) 组合唯一

详见 docs/cmdb-group-import-export-quick-spec.md" -C pantheon-ops
```

---

**优先级**: P2（在 BizScope 完成后执行）
**难点**: 父路径解析，需要递归查询
