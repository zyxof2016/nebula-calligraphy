# 脚本

本目录包含仓库级验证、备份恢复和发布工具。

- `build-ip-release.sh`：构建 Flutter Web、Linux amd64 后端和完整裸机发布包。
- `build-docker-image.sh`：先构建 Flutter Web，再构建本地 Docker 镜像。
- `calligraphy-backup.sh` / `calligraphy-restore.sh`：备份和恢复本地生产试用状态。
- `calligraphy-web-auth-test.mjs`：验证浏览器认证配置和跳转流程。
