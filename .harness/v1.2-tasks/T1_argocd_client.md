# 任务包 T1: ArgoCD Client 封装

**任务ID**: T1  
**任务名称**: ArgoCD Client 封装  
**估计工作量**: 3天  
**优先级**: P0  
**负责人**: [待分配]  
**状态**: 未开始  

---

## 1. 任务目标

封装 ArgoCD API 客户端，提供类型安全的 Go 接口来与 ArgoCD 服务器交互。这是所有 ArgoCD 功能的基础层。

**核心输出**:
- 完整的 ArgoCD API 客户端
- 支持 Application 的 CRUD 操作
- 支持 Sync/Rollback 操作
- 错误处理和重试机制

---

## 2. 前置依赖

### 依赖任务
- 无（这是基础任务）

### 依赖资源
- [ ] ArgoCD 测试服务器 (可用 Kind + ArgoCD 搭建)
- [ ] ArgoCD API Token
- [ ] ArgoCD API 文档: https://argo-cd.readthedocs.io/en/stable/developer-guide/api-docs/

### 前置知识
- ArgoCD 基本概念 (Application/Sync/Health)
- Go HTTP 客户端编程
- JSON 序列化/反序列化
- TLS/证书验证

---

## 3. 功能需求

### 核心功能
1. **连接管理**
   - 创建 ArgoCD 客户端
   - Bearer Token 认证
   - TLS 证书验证 (支持跳过验证)
   - 超时控制

2. **Application 管理**
   - CreateApplication: 创建应用
   - GetApplication: 获取应用详情
   - ListApplications: 列出所有应用
   - UpdateApplication: 更新应用配置
   - DeleteApplication: 删除应用

3. **操作接口**
   - SyncApplication: 手动同步
   - RollbackApplication: 回滚到指定版本
   - TerminateOperation: 终止正在执行的操作
   - GetApplicationSyncHistory: 获取同步历史

4. **健康检查**
   - HealthCheck: 测试服务器连通性
   - GetVersion: 获取 ArgoCD 版本

### 非功能需求
- **性能**: 单次 API 调用 <3s
- **安全**: 支持 HTTPS，Token 不打印日志
- **可靠性**: 网络错误自动重试 3 次

---

## 4. 技术规格

### 4.1 数据结构

```go
// backend/modules/business/cicd/argocd/client.go

package argocd

import (
    "context"
    "crypto/tls"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// Client wraps ArgoCD API operations
type Client struct {
    serverURL  string
    token      string
    httpClient *http.Client
}

// Application represents ArgoCD Application
type Application struct {
    Metadata ApplicationMetadata `json:"metadata"`
    Spec     ApplicationSpec     `json:"spec"`
    Status   ApplicationStatus   `json:"status"`
}

type ApplicationMetadata struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace,omitempty"`
}

type ApplicationSpec struct {
    Source      ApplicationSource      `json:"source"`
    Destination ApplicationDestination `json:"destination"`
    SyncPolicy  *SyncPolicy            `json:"syncPolicy,omitempty"`
}

type ApplicationSource struct {
    RepoURL        string `json:"repoURL"`
    Path           string `json:"path"`
    TargetRevision string `json:"targetRevision"`
}

type ApplicationDestination struct {
    Server    string `json:"server"`
    Namespace string `json:"namespace"`
}

type SyncPolicy struct {
    Automated *AutomatedSync `json:"automated,omitempty"`
}

type AutomatedSync struct {
    Prune    bool `json:"prune"`
    SelfHeal bool `json:"selfHeal"`
}

type ApplicationStatus struct {
    Sync   SyncStatus   `json:"sync"`
    Health HealthStatus `json:"health"`
    OperationState *OperationState `json:"operationState,omitempty"`
    Resources []ResourceStatus `json:"resources"`
}

type SyncStatus struct {
    Status   string `json:"status"` // Synced/OutOfSync/Unknown
    Revision string `json:"revision"`
}

type HealthStatus struct {
    Status  string `json:"status"` // Healthy/Progressing/Degraded/Suspended/Missing/Unknown
    Message string `json:"message,omitempty"`
}

type OperationState struct {
    Phase      string     `json:"phase"` // Running/Succeeded/Failed/Error/Terminating
    Message    string     `json:"message,omitempty"`
    StartedAt  time.Time  `json:"startedAt"`
    FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

type ResourceStatus struct {
    Kind      string       `json:"kind"`
    Name      string       `json:"name"`
    Namespace string       `json:"namespace"`
    Status    string       `json:"status"`
    Health    HealthStatus `json:"health"`
}

// VersionInfo represents ArgoCD version
type VersionInfo struct {
    Version string `json:"version"`
    BuildDate string `json:"buildDate"`
    GitCommit string `json:"gitCommit"`
}
```

### 4.2 核心接口

```go
// NewClient creates an ArgoCD API client
func NewClient(serverURL, token string, insecureSkipVerify bool) *Client {
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: insecureSkipVerify,
        },
    }
    
    return &Client{
        serverURL: serverURL,
        token:     token,
        httpClient: &http.Client{
            Timeout:   30 * time.Second,
            Transport: tr,
        },
    }
}

// HealthCheck tests connectivity to ArgoCD server
func (c *Client) HealthCheck(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "GET",
        fmt.Sprintf("%s/api/version", c.serverURL), nil)
    if err != nil {
        return err
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("connection failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    
    return nil
}

// GetVersion retrieves ArgoCD version
func (c *Client) GetVersion(ctx context.Context) (*VersionInfo, error) {
    req, err := http.NewRequestWithContext(ctx, "GET",
        fmt.Sprintf("%s/api/version", c.serverURL), nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    
    var version VersionInfo
    if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
        return nil, err
    }
    
    return &version, nil
}

// CreateApplication creates a new ArgoCD Application
func (c *Client) CreateApplication(ctx context.Context, app *Application) (*Application, error) {
    body, err := json.Marshal(app)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST",
        fmt.Sprintf("%s/api/v1/applications", c.serverURL),
        bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("create failed: %d %s", resp.StatusCode, string(bodyBytes))
    }
    
    var result Application
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &result, nil
}

// GetApplication retrieves ArgoCD Application details
func (c *Client) GetApplication(ctx context.Context, name string) (*Application, error) {
    req, err := http.NewRequestWithContext(ctx, "GET",
        fmt.Sprintf("%s/api/v1/applications/%s", c.serverURL, name), nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == http.StatusNotFound {
        return nil, fmt.Errorf("application not found")
    }
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    
    var app Application
    if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
        return nil, err
    }
    
    return &app, nil
}

// ListApplications lists all applications
func (c *Client) ListApplications(ctx context.Context) ([]*Application, error) {
    req, err := http.NewRequestWithContext(ctx, "GET",
        fmt.Sprintf("%s/api/v1/applications", c.serverURL), nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("list failed: %d", resp.StatusCode)
    }
    
    var result struct {
        Items []*Application `json:"items"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return result.Items, nil
}

// SyncApplication triggers a manual sync
func (c *Client) SyncApplication(ctx context.Context, name string, revision string, prune, dryRun bool) error {
    syncReq := map[string]interface{}{
        "revision": revision,
        "prune":    prune,
        "dryRun":   dryRun,
    }
    
    body, _ := json.Marshal(syncReq)
    
    req, err := http.NewRequestWithContext(ctx, "POST",
        fmt.Sprintf("%s/api/v1/applications/%s/sync", c.serverURL, name),
        bytes.NewReader(body))
    if err != nil {
        return err
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("sync failed: %d %s", resp.StatusCode, string(bodyBytes))
    }
    
    return nil
}

// RollbackApplication rolls back to a specific revision
func (c *Client) RollbackApplication(ctx context.Context, name string, revision string) error {
    rollbackReq := map[string]interface{}{
        "revision": revision,
        "prune":    true,
    }
    
    body, _ := json.Marshal(rollbackReq)
    
    req, err := http.NewRequestWithContext(ctx, "POST",
        fmt.Sprintf("%s/api/v1/applications/%s/rollback", c.serverURL, name),
        bytes.NewReader(body))
    if err != nil {
        return err
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("rollback failed: %d %s", resp.StatusCode, string(bodyBytes))
    }
    
    return nil
}

// DeleteApplication deletes an ArgoCD Application
func (c *Client) DeleteApplication(ctx context.Context, name string, cascade bool) error {
    req, err := http.NewRequestWithContext(ctx, "DELETE",
        fmt.Sprintf("%s/api/v1/applications/%s?cascade=%v", c.serverURL, name, cascade), nil)
    if err != nil {
        return err
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("delete failed: %d", resp.StatusCode)
    }
    
    return nil
}
```

---

## 5. 实现指南

### 5.1 目录结构

```
backend/modules/business/cicd/argocd/
├── client.go         - ArgoCD 客户端核心实现
├── client_test.go    - 单元测试
├── types.go          - 类型定义
└── errors.go         - 错误定义
```

### 5.2 关键步骤

1. **创建项目结构**
   ```bash
   mkdir -p backend/modules/business/cicd/argocd
   cd backend/modules/business/cicd/argocd
   ```

2. **实现 types.go**
   - 定义所有 ArgoCD API 的数据结构
   - 使用 json 标签
   - 添加必要的辅助方法

3. **实现 client.go**
   - 实现 `NewClient` 构造函数
   - 实现 `HealthCheck` 和 `GetVersion`
   - 实现 Application CRUD 方法
   - 实现 Sync/Rollback 方法
   - 添加错误处理和重试逻辑

4. **实现 client_test.go**
   - Mock HTTP 服务器
   - 测试所有公开方法
   - 测试错误情况

5. **手工测试**
   - 搭建测试 ArgoCD 服务器
   - 获取 API Token
   - 编写测试脚本验证所有功能

---

## 6. 测试要求

### 6.1 单元测试

**必须覆盖**:
- [x] NewClient 构造函数
- [x] HealthCheck 正常和失败情况
- [x] GetVersion
- [x] CreateApplication 正常和失败情况
- [x] GetApplication 存在和不存在情况
- [x] ListApplications
- [x] SyncApplication
- [x] RollbackApplication
- [x] DeleteApplication

**测试框架**:
```go
// client_test.go
package argocd

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHealthCheck(t *testing.T) {
    // Mock ArgoCD server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/api/version" {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte(`{"version":"v2.5.0"}`))
            return
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    defer server.Close()
    
    client := NewClient(server.URL, "test-token", true)
    
    err := client.HealthCheck(context.Background())
    if err != nil {
        t.Errorf("HealthCheck failed: %v", err)
    }
}

func TestGetApplication(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/api/v1/applications/test-app" {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte(`{
                "metadata": {"name": "test-app"},
                "spec": {
                    "source": {"repoURL": "https://github.com/test/repo", "path": ".", "targetRevision": "main"},
                    "destination": {"server": "https://kubernetes.default.svc", "namespace": "default"}
                },
                "status": {
                    "sync": {"status": "Synced", "revision": "abc123"},
                    "health": {"status": "Healthy"}
                }
            }`))
            return
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    defer server.Close()
    
    client := NewClient(server.URL, "test-token", true)
    
    app, err := client.GetApplication(context.Background(), "test-app")
    if err != nil {
        t.Errorf("GetApplication failed: %v", err)
    }
    
    if app.Metadata.Name != "test-app" {
        t.Errorf("Expected name 'test-app', got '%s'", app.Metadata.Name)
    }
    
    if app.Status.Sync.Status != "Synced" {
        t.Errorf("Expected sync status 'Synced', got '%s'", app.Status.Sync.Status)
    }
}

// TODO: 添加更多测试用例
```

**覆盖率要求**: >80%

### 6.2 集成测试

**测试环境**:
- 使用 Kind 部署 K8s 集群
- 部署 ArgoCD: `kubectl create namespace argocd && kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml`
- 获取 admin 密码: `kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d`
- Port-forward: `kubectl port-forward svc/argocd-server -n argocd 8080:443`

**测试场景**:
1. 连接到真实 ArgoCD 服务器
2. 创建测试 Application
3. 查询 Application 详情
4. 列出所有 Applications
5. 触发 Sync
6. 删除 Application

**测试脚本** (`test_integration.sh`):
```bash
#!/bin/bash
set -e

echo "=== ArgoCD Client 集成测试 ==="

# 1. 检查 ArgoCD 是否可达
echo "1. 检查 ArgoCD 连接..."
go run test_client.go health-check

# 2. 创建 Application
echo "2. 创建测试 Application..."
go run test_client.go create-app --name test-app --repo https://github.com/argoproj/argocd-example-apps --path guestbook

# 3. 获取 Application
echo "3. 获取 Application 详情..."
go run test_client.go get-app --name test-app

# 4. 列出 Applications
echo "4. 列出所有 Applications..."
go run test_client.go list-apps

# 5. 同步 Application
echo "5. 同步 Application..."
go run test_client.go sync-app --name test-app

# 6. 删除 Application
echo "6. 删除 Application..."
go run test_client.go delete-app --name test-app

echo "=== 测试通过 ==="
```

---

## 7. 验收标准

### 7.1 代码质量
- [ ] `go build ./backend/modules/business/cicd/argocd` 编译通过
- [ ] `go test ./backend/modules/business/cicd/argocd` 测试通过
- [ ] `go vet ./backend/modules/business/cicd/argocd` 无告警
- [ ] 代码注释完整 (所有公开函数有注释)

### 7.2 功能完整性
- [ ] HealthCheck 功能正常
- [ ] CreateApplication 功能正常
- [ ] GetApplication 功能正常
- [ ] ListApplications 功能正常
- [ ] SyncApplication 功能正常
- [ ] RollbackApplication 功能正常
- [ ] DeleteApplication 功能正常

### 7.3 性能指标
- [ ] 单次 API 调用延迟 <3s
- [ ] 支持并发调用 (线程安全)

### 7.4 安全性
- [ ] Token 不出现在日志中
- [ ] 支持 HTTPS
- [ ] 支持 InsecureSkipVerify 选项

---

## 8. 交付物清单

### 必须交付
- [ ] `backend/modules/business/cicd/argocd/client.go`
- [ ] `backend/modules/business/cicd/argocd/types.go`
- [ ] `backend/modules/business/cicd/argocd/errors.go`
- [ ] `backend/modules/business/cicd/argocd/client_test.go`
- [ ] 集成测试脚本 `test_integration.sh`
- [ ] 实现说明 `IMPLEMENTATION.md`

### 可选交付
- [ ] `test_client.go` (手工测试工具)
- [ ] API 调用示例

---

## 9. 风险与注意事项

### 技术风险
- **风险1**: ArgoCD API 版本不兼容
  - **缓解措施**: 明确支持的 ArgoCD 版本 (v2.5+)，在文档中说明

- **风险2**: 网络超时导致误判
  - **缓解措施**: 设置合理的超时时间 (30s)，添加重试机制

### 注意事项
- ArgoCD API Token 有过期时间，需要处理 401 错误
- Application 名称在 ArgoCD 中必须唯一
- Sync 操作是异步的，需要轮询状态
- 删除 Application 时 cascade=true 会删除 K8s 资源

---

## 10. 参考资料

### 技术文档
- ArgoCD API 文档: https://argo-cd.readthedocs.io/en/stable/developer-guide/api-docs/
- ArgoCD 架构: https://argo-cd.readthedocs.io/en/stable/operator-manual/architecture/

### 示例代码
- ArgoCD Go Client (官方): https://github.com/argoproj/argo-cd/tree/master/pkg/apiclient

---

## 11. 变更记录

| 日期 | 变更内容 | 变更人 |
|------|---------|--------|
| 2026-08-28 | 创建任务包 | Claude |

---

## 12. 问题追踪

### 待解决问题
- [ ] 是否需要支持 ArgoCD v1.x 版本？
- [ ] 是否需要支持 gRPC 接口？

### 已解决问题
(暂无)
