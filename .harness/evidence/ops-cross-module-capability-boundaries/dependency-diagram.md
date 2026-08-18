# Dependency Diagram

```text
business overlay wiring
  -> BizScope Service (BizScopeReader)
  -> CMDB capability (CMDBHostReader + CMDBOwnershipCommand)
  -> CMDB Host/Group/Label services
  -> Deploy service (typed readers and commands)
  -> K8s Cluster service (BizScopeReader)

BizScope service --reader/command--> CMDB capability
Deploy service --reader/command--> CMDB capability
Deploy/K8s --reader--> BizScope service
```

All edges are same-process typed Go interfaces. No loopback HTTP was added and
no cross-module DTO contains a GORM model or `*gorm.DB`.
