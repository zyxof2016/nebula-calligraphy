# nebula-calligraphy

`nebula-calligraphy` 是独立的 C 端产品线，用于 AI 辅助书法学习和基于碑帖字库的集字创作。

它接入星云底座的身份、存储、AI 网关、审计以及后续计费能力，但不属于 Signage 的排程、播放器和设备生命周期主流程。

## 产品范围

MVP 聚焦日常书法学习闭环：

1. 查询单字。
2. 对比名家碑帖字形。
3. 下载每日临摹练习模板。
4. 标记练习并查看近期历史。
5. 收藏常练字，便于重复学习。
6. 将文本排版为书法作品草稿。
7. 导出 SVG/PNG 学习模板。

暂缓范围：

- AR 临摹。
- 社区信息流。
- 教师和课堂工作流。
- 完整 AI 书写评分。
- 个性化书体训练。

## 仓库结构

```text
nebula-calligraphy/
├── apps/mobile/              # Flutter C 端应用
├── assets/copybooks/         # 碑帖 manifest、来源说明和本地资产目录约定
├── assets/fonts/             # 服务端渲染兜底字体；不打进前端 Web 包
├── web/admin/                # React + Ant Design 内容和标注后台
├── services/calligraphy/     # Go API 服务
├── pkg/layout/               # 章法排版算法
├── pkg/glyph/                # 字形匹配和元数据逻辑
├── pkg/render/               # 后续复用的 PNG/SVG 导出辅助
├── docs/products/            # 产品说明
├── docs/contracts/           # 机器可读契约
└── scripts/
```

## 底座依赖

| 能力 | 星云依赖 |
|------|----------|
| 用户身份 | 当前支持本地 MVP 认证；生产接入 `nebula-platform` Identity |
| API 网关 | `nebula-platform` Gateway |
| 对象存储 | S3/OSS/MinIO 部署形态 |
| AI 能力 | 星云 AI 网关和模型适配器 |
| 审计 | `nebula-analytics-audit` 事件模型 |
| 开放 API | `nebula-open-platform` SDK 和 Webhook 模式 |

## MVP 运行方式

第一阶段运行单元是 `services/calligraphy` 下的 Go API 服务。

```bash
cd services/calligraphy
GOCACHE=/tmp/nebula-calligraphy-go-cache go test ./...
PORT=8090 go run ./cmd/calligraphy
```

如果需要可重启的本地试用环境，启用文件持久化草稿和本地导出产物：

```bash
mkdir -p .local/artifacts
CALLIGRAPHY_AUTH_FILE=.local/auth.json \
CALLIGRAPHY_DATA_FILE=.local/drafts.json \
CALLIGRAPHY_LEARNING_FILE=.local/learning.json \
CALLIGRAPHY_AUDIT_FILE=.local/audit.jsonl \
CALLIGRAPHY_EXPORT_DIR=.local/artifacts \
CALLIGRAPHY_RENDER_FONT_FILE=../../assets/fonts/MaShanZheng-Regular.ttf \
CALLIGRAPHY_RENDER_CACHE_DIR=.local/render-cache \
CALLIGRAPHY_WEB_DIR=../../web/app \
PORT=8090 \
go run ./cmd/calligraphy
```

然后打开 `http://127.0.0.1:8090/` 使用试用工作台。工作台覆盖本地注册/登录、常用字分组、查字、单字详情、练习模板预览/下载、收藏、练习记录、学习档案、章法预览、草稿保存/列表/载入/删除、导出历史以及 PNG/SVG 导出。Flutter 客户端中文 UI 使用设备系统字体，书法参考字和章法预览优先加载服务端渲染 PNG；Web 包不再内置书法字体或完整 CJK 字体。

如需加载真实碑帖裁切字库，先校验 manifest，再通过环境变量接入：

```bash
cd services/calligraphy
go run ./cmd/calligraphy-glyph-manifest validate ../../assets/copybooks/jiuchenggong/manifest.sample.json

CALLIGRAPHY_GLYPH_MANIFEST_FILE=/data/nebula-calligraphy/copybooks/jiuchenggong/manifest.json \
PORT=8090 \
go run ./cmd/calligraphy
```

`CALLIGRAPHY_GLYPH_MANIFEST_FILE` 中只有 `review_status=published` 且非 `restricted` 的字会对外服务。`production` 和 `managed` 模式不会混入内置常用字样本，并要求清单至少包含一个可发布字形。

### 服务端字图渲染

前端不再下载书法字体。API 会在字形搜索、详情和章法预览中返回 `render_asset`：

- `GET /api/v1/calligraphy/glyphs/{glyph_id}/render.png`：按需生成 512x512 PNG 字图，支持 `grid=mi|jiugong|outline|none`。
- `CALLIGRAPHY_RENDER_FONT_FILE`：服务端兜底字体文件路径。后续导入自有字体库时替换该路径即可。
- `CALLIGRAPHY_RENDER_CACHE_DIR`：PNG 渲染缓存目录。相同字形和网格参数只渲染一次。

后续导入真实碑帖裁切图或自建字体库时，应继续保持“服务端管理字库、前端消费 `render_asset`”的契约，不把字体文件下发到浏览器。

Flutter Web 本地开发默认使用 `http://localhost:8088`，trial 模式会自动放行该本机源。生产或托管环境需要跨源访问时，使用 `CALLIGRAPHY_ALLOWED_ORIGINS=https://calligraphy.example` 显式配置允许源。

可用 MVP 接口：

| 接口 | 用途 |
|------|------|
| `GET /health` | 服务健康探针 |
| `GET /ready` | 就绪探针；托管模式实际检查 PostgreSQL 迁移、对象存储和 Identity |
| `GET /metrics` | Prometheus 文本指标，包含进程运行时长和请求数 |
| `POST /api/v1/calligraphy/auth/register` | 创建本地 MVP 学习者账号并返回会话 |
| `POST /api/v1/calligraphy/auth/login` | 登录并返回本地会话 |
| `POST /api/v1/calligraphy/auth/logout` | 吊销当前本地会话令牌 |
| `GET /api/v1/calligraphy/auth/me` | 通过 `Authorization: Bearer <token>` 解析当前用户 |
| `GET /api/v1/calligraphy/glyphs/search?character=山&style=ou` | 查询已授权且已发布的字形样本 |
| `GET /api/v1/calligraphy/glyphs/presets` | 获取预置常用练习字分组 |
| `GET /api/v1/calligraphy/glyphs/{glyph_id}` | 获取单字详情、结构说明和练习模板 |
| `POST /api/v1/calligraphy/layouts/preview` | 生成传统竖排从右到左的章法预览，包含边距、落款和印章位 |
| `POST /api/v1/calligraphy/artworks/drafts` | 根据排版请求保存作品草稿 |
| `GET /api/v1/calligraphy/artworks/drafts?owner_user_id=user-1` | 查询用户草稿列表 |
| `GET /api/v1/calligraphy/artworks/drafts/{artwork_id}` | 读取单个草稿 |
| `DELETE /api/v1/calligraphy/artworks/drafts/{artwork_id}` | 删除单个草稿 |
| `POST /api/v1/calligraphy/artworks/drafts/{artwork_id}/exports` | 导出 MVP 内联 PNG 图片或 SVG 矢量参考产物 |
| `GET /api/v1/calligraphy/users/{owner_user_id}/learning` | 读取收藏字、近期练习和学习统计 |
| `POST /api/v1/calligraphy/users/{owner_user_id}/favorites` | 收藏一个字形 |
| `DELETE /api/v1/calligraphy/users/{owner_user_id}/favorites/{glyph_id}` | 移除一个收藏字 |
| `POST /api/v1/calligraphy/users/{owner_user_id}/practice` | 记录一次单字和模板练习 |
| `GET /artifacts/{storage_key}` | 配置 `CALLIGRAPHY_EXPORT_DIR` 后下载本地试用导出产物 |

用户草稿、收藏、练习和学习档案接口都要求 `Authorization: Bearer <token>`。服务端会从会话推导有效所属用户，并拒绝 `owner_user_id` 不匹配的请求，返回 `403`。本地密码使用 Argon2id 保存，本地会话默认 24 小时过期，可通过 `CALLIGRAPHY_SESSION_TTL` 调整。

MVP 服务内置 120+ 个常用学习字，覆盖欧体、颜体、柳体、赵体和瘦金体五种书体。认证、草稿和学习记录默认存内存；配置 `CALLIGRAPHY_AUTH_FILE`、`CALLIGRAPHY_DATA_FILE` 和 `CALLIGRAPHY_LEARNING_FILE` 后会落到本地 JSON 文件；`CALLIGRAPHY_AUDIT_FILE` 写入 JSONL 审计事件；`CALLIGRAPHY_EXPORT_DIR` 写入 PNG/SVG 导出产物；`CALLIGRAPHY_WEB_DIR` 通过同一个 Go 服务托管静态试用工作台；`CALLIGRAPHY_GLYPH_MANIFEST_FILE` 加载真实碑帖范字 manifest；`CALLIGRAPHY_RENDER_FONT_FILE` 和 `CALLIGRAPHY_RENDER_CACHE_DIR` 控制服务端字图渲染。生产身份应接入 Nebula Identity 和 PostgreSQL 用户体系；公开商业生产还需要授权碑帖入库、专家审核发布流程、PostgreSQL 持久化和对象存储导出。

Go 服务默认设置保守的 HTTP 超时和安全响应头：`X-Content-Type-Options`、`X-Frame-Options`、`Referrer-Policy` 和 `Content-Security-Policy`。

## 生产试用模式

对外可访问部署请使用 `CALLIGRAPHY_RUNTIME_PROFILE=production`。该模式下如果缺少以下配置，服务会拒绝启动：

- `CALLIGRAPHY_AUTH_FILE`
- `CALLIGRAPHY_DATA_FILE`
- `CALLIGRAPHY_LEARNING_FILE`
- `CALLIGRAPHY_AUDIT_FILE`
- `CALLIGRAPHY_EXPORT_DIR`
- `CALLIGRAPHY_WEB_DIR`
- `CALLIGRAPHY_GLYPH_MANIFEST_FILE`

HTTPS 应在反向代理或负载均衡处终止，只把私有端口上的 HTTP 转发给 Go 进程。公网代理应设置 HSTS、请求体大小限制，并暴露 `/health` 和 `/ready` 给监控系统。
生产试用模式的 `/ready` 会真实验证持久化目录可写、审计文件可追加以及 Web
入口文件可读取，目录权限或磁盘写入故障会返回 `503`。

## 托管底座模式

只有在部署已接入外部平台底座时，才使用 `CALLIGRAPHY_RUNTIME_PROFILE=managed`。服务会校验必要的底座配置，并在 `/ready` 返回 `foundation_mode=managed`。

必填配置：

- `CALLIGRAPHY_DATABASE_URL`
- `CALLIGRAPHY_IDENTITY_ISSUER`
- `CALLIGRAPHY_IDENTITY_AUDIENCE`
- `CALLIGRAPHY_IDENTITY_BASE_URL`
- `CALLIGRAPHY_IDENTITY_JWKS_URL` 或 `CALLIGRAPHY_IDENTITY_HS256_SECRET`
- `CALLIGRAPHY_OBJECT_STORAGE_ENDPOINT`
- `CALLIGRAPHY_OBJECT_STORAGE_BUCKET`
- `CALLIGRAPHY_OBJECT_STORAGE_REGION`
- `CALLIGRAPHY_OBJECT_STORAGE_ACCESS_KEY`
- `CALLIGRAPHY_OBJECT_STORAGE_SECRET_KEY`
- `CALLIGRAPHY_OBJECT_STORAGE_SESSION_TOKEN`（使用云厂商临时凭证时）
- `CALLIGRAPHY_AUDIT_SINK`
- `CALLIGRAPHY_WEB_DIR`
- `CALLIGRAPHY_GLYPH_MANIFEST_FILE`

托管模式使用 PostgreSQL 存储用户、会话、草稿、收藏和练习记录；使用 S3 兼容对象存储保存导出产物；使用 JWKS/RS256 或 Nebula HS256 校验 Identity 令牌的签名、issuer、audience、exp 和 nbf；使用 HTTP/HTTPS 审计接收端接收 JSON 审计事件。如果审计服务要求 bearer 令牌，配置 `CALLIGRAPHY_AUDIT_TOKEN`。托管模式会关闭本地注册、登录和退出端点。

浏览器登录由 `GET /api/v1/calligraphy/runtime-config` 驱动。该接口只返回公开配置，不返回校验密钥、对象存储凭据、数据库 URL 或审计令牌。

支持的托管认证模式：

- `CALLIGRAPHY_AUTH_MODE=nebula-direct`：当前 Flutter Web、Android 和 iOS 客户端的默认托管模式。客户端通过 HTTPS 把用户名和密码提交到 `CALLIGRAPHY_IDENTITY_LOGIN_ENDPOINT`，默认是 `${CALLIGRAPHY_IDENTITY_BASE_URL}/api/v1/auth/login`，再使用访问令牌调用 Calligraphy API。
- `CALLIGRAPHY_AUTH_MODE=oidc-pkce`：轻量 Web 客户端 `web/app` 已支持的标准浏览器 SSO。只有在 Identity 已提供标准 authorize/token 端点且部署入口使用该客户端时才显式启用。授权和 token 端点可分别用 `CALLIGRAPHY_IDENTITY_AUTHORIZATION_ENDPOINT`、`CALLIGRAPHY_IDENTITY_TOKEN_ENDPOINT` 覆盖。

未显式设置 `CALLIGRAPHY_AUTH_MODE` 时，托管模式固定使用 `nebula-direct`，不会因存在 client ID 自动切换。Flutter 客户端收到 `oidc-pkce` 配置时会拒绝账号密码登录，避免错误地把凭据提交给 Calligraphy。服务 CSP 会自动只放行 Calligraphy 源和配置的 Identity 源。

生产建议把 Calligraphy 和 Identity 放在同一个 HTTPS 网关源下。如果必须跨源部署，Identity 的 CORS 只能放行 Calligraphy 源。

轻量 Web 客户端启用 `oidc-pkce` 前，Identity 必须以精确匹配方式登记 Calligraphy OIDC 公共客户端：

```bash
OIDC_ISSUER=https://identity.example.com
OIDC_PUBLIC_CLIENTS=nebula-calligraphy-web=https://calligraphy.example.com/
CORS_ORIGINS=https://calligraphy.example.com
```

然后用同一个客户端和 Identity 公开地址配置 Calligraphy：

```bash
CALLIGRAPHY_AUTH_MODE=oidc-pkce
CALLIGRAPHY_IDENTITY_BASE_URL=https://identity.example.com
CALLIGRAPHY_IDENTITY_CLIENT_ID=nebula-calligraphy-web
CALLIGRAPHY_IDENTITY_AUDIENCE=nebula-calligraphy
```

启动托管模式前，先执行 PostgreSQL 迁移：

```bash
cd services/calligraphy
CALLIGRAPHY_DATABASE_URL=postgres://calligraphy:password@postgres:5432/calligraphy \
go run ./cmd/calligraphy-migrate
```

随后用上述托管配置启动服务。`/ready` 只有在 PostgreSQL 已迁移、对象存储桶可访问且 Identity JWKS 可读取时才返回 `200`。

## 部署方式

| 方式 | 入口 | 适用场景 |
|------|------|----------|
| 裸机 | `scripts/build-ip-release.sh`、`deploy/ip/` | 单台 2C4G Ubuntu 试用或小规模生产 |
| Docker | `scripts/build-docker-image.sh`、`deploy/docker/compose.yaml` | 单机容器部署，无需外部镜像仓库 |
| Kubernetes | `deploy/helm/nebula-calligraphy/` | 托管底座模式，外接 PostgreSQL、对象存储、Identity 和审计 |

Docker 启动前必须指定已授权、已审校的生产字形清单；sample 草稿不能通过生产门禁：

```bash
CALLIGRAPHY_GLYPH_MANIFEST_FILE_HOST=/absolute/path/to/manifest.json \
docker compose -f deploy/docker/compose.yaml up -d
```

Android 正式包必须配置 `CALLIGRAPHY_ANDROID_KEYSTORE`、`CALLIGRAPHY_ANDROID_KEYSTORE_PASSWORD`、`CALLIGRAPHY_ANDROID_KEY_ALIAS` 和 `CALLIGRAPHY_ANDROID_KEY_PASSWORD`。release 构建不会回退到 debug 签名。
Android 正式包禁用明文 HTTP；本地或 IP 联调请使用 debug 包，生产 Identity 和 Calligraphy 必须通过受信任的 HTTPS 地址提供。

备份本地生产试用状态：

```bash
CALLIGRAPHY_AUTH_FILE=/srv/calligraphy/auth.json \
CALLIGRAPHY_DATA_FILE=/srv/calligraphy/drafts.json \
CALLIGRAPHY_LEARNING_FILE=/srv/calligraphy/learning.json \
CALLIGRAPHY_AUDIT_FILE=/srv/calligraphy/audit.jsonl \
CALLIGRAPHY_EXPORT_DIR=/srv/calligraphy/artifacts \
scripts/calligraphy-backup.sh /srv/backups/calligraphy-$(date +%Y%m%d%H%M%S)
```

恢复时使用同一组环境变量并执行：

```bash
scripts/calligraphy-restore.sh /srv/backups/<backup-dir>
```

## 本地检查

```bash
make -f Makefile.split split-check
make -f Makefile.split split-deploy-test
```
