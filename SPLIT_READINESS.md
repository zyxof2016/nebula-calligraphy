# nebula-calligraphy 拆仓与生产就绪状态

`nebula-calligraphy` 已完成独立仓库、独立编译、独立部署和星云底座适配。Go API、Flutter 客户端、契约、裸机发布、Docker 和 Helm 均在本仓库维护。

## 已完成

- 注册、登录、查字、临摹、收藏、学习档案、章法、草稿和 PNG/SVG 导出主流程。
- 本地文件生产试用模式，以及 PostgreSQL、S3、Identity、HTTP 审计托管模式。
- Argon2id 密码、24 小时会话、owner 权限校验和托管模式关闭本地认证。
- PostgreSQL、对象存储和 Identity 真实就绪检查。
- 裸机 systemd + Nginx、Docker Compose、Kubernetes Helm 三种部署入口。
- Go、Flutter、Android debug、契约、脚本、视觉快照和容器构建 CI。

## 上线边界

技术运行闭环已经具备。公开商业上线前仍必须准备真实授权碑帖、至少两个书体的高频字内容和专家审校发布流程。当前通用服务端字体仅用于流程验证，不能宣称为真实欧体、颜体、柳体、赵体或瘦金体范字。

## 必跑检查

```bash
make -f Makefile.split split-check
cd apps/mobile && flutter analyze && flutter test
helm lint deploy/helm/nebula-calligraphy
```
