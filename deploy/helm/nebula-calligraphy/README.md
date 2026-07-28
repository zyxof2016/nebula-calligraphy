# Helm 部署

该 Chart 只用于托管底座模式，应用保持无状态。PostgreSQL、S3 兼容对象存储、Nebula Identity 和审计服务必须先准备好。

创建 Secret：

```bash
kubectl create secret generic nebula-calligraphy \
  --from-literal=database-url='postgres://...' \
  --from-literal=object-storage-access-key='...' \
  --from-literal=object-storage-secret-key='...' \
  --from-literal=audit-token='...'
```

使用云厂商临时访问凭证时，再加入 `--from-literal=object-storage-session-token='...'`。对象存储凭证至少需要目标桶的检查权限和 `PutObject` 权限。

安装：

```bash
helm upgrade --install calligraphy deploy/helm/nebula-calligraphy \
  --set image.repository=ghcr.io/zyxof2016/nebula-calligraphy \
  --set image.tag=<version> \
  --set ingress.enabled=true \
  --set ingress.host=calligraphy.example.com
```

安装和升级前会通过 Helm hook 执行 PostgreSQL 迁移。`/ready` 会实际检查数据库、对象存储和 Identity JWKS。
Chart 默认使用当前 Flutter 客户端已完整支持的 `nebula-direct`；所有公网入口和 Identity 登录端点必须启用 HTTPS。
