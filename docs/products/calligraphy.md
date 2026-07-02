# Nebula Calligraphy 产品说明

## 产品定位

Nebula Calligraphy 是面向 C 端的 AI 书法学习与集字创作应用。它为学习者提供一个随身书法导师，覆盖单字查询、书体对比、章法规划和可导出的临摹练习模板。

## MVP 范围

| 模块 | 已纳入 | 暂缓 |
|------|--------|------|
| 智能书法字典 | 拼音/部首/笔画检索、字形对比、来源碑帖、基础书写说明 | 拍照检索、手写检索、语音讲解 |
| 碑帖字库 | `Jiuchenggong`、`Duobaota` 和高频字样本；支持从可追溯 manifest 加载真实碑帖裁切字 | 大规模碑帖扩展、自动切字和专家工作台 |
| 集字创作 | 文本输入、书体/碑帖选择、幅式选择、自动章法排版、落款和印章预览、单字替换 | 偏旁合成、跨书家自动混排 |
| 导出 | PNG/PDF、临摹模板、作品参考图 | AR 临摹和视频卡片 |
| 用户资产 | 收藏、每日练习记录、学习档案、作品草稿、近期历史 | 社区和课堂工作流 |

## 当前运行单元

仓库已包含 `services/calligraphy`，这是一个 Go MVP API 服务，用于在移动端和管理后台开发前先验证 C 端核心闭环。

| API | 状态 | 说明 |
|-----|------|------|
| `GET /health` | 已实现 | 容器和进程健康探针 |
| `GET /ready` | 已实现 | 生产配置就绪探针，会校验持久化配置 |
| `GET /metrics` | 已实现 | Prometheus 文本指标，包含请求数和进程运行时长 |
| `POST /api/v1/calligraphy/auth/register` | 已实现 | 创建本地 MVP 学习者账号并返回会话令牌 |
| `POST /api/v1/calligraphy/auth/login` | 已实现 | 校验用户名密码并返回会话令牌 |
| `POST /api/v1/calligraphy/auth/logout` | 已实现 | 吊销当前本地会话令牌 |
| `GET /api/v1/calligraphy/auth/me` | 已实现 | 通过 `Authorization: Bearer <token>` 解析当前学习者 |
| `GET /api/v1/calligraphy/glyphs/search` | 已实现 | 只查询已授权且已发布的种子字形；配置 `CALLIGRAPHY_GLYPH_MANIFEST_FILE` 后优先返回 manifest 中已发布的真实裁切字 |
| `GET /api/v1/calligraphy/glyphs/presets` | 已实现 | 每种书体返回 120+ 个预置常用学习字，并按练习目的分组 |
| `GET /api/v1/calligraphy/glyphs/{id}` | 已实现 | 返回字形详情、结构说明、笔法说明和练习模板 |
| `POST /api/v1/calligraphy/layouts/preview` | 已实现 | 传统 `vertical_rtl` 章法预览，支持边距、落款和印章位；斗方场景优先保持竖排作品感，例如 4 字 2x2、8 字 2 列 4 行 |
| `POST /api/v1/calligraphy/artworks/drafts` | 已实现 | 根据排版请求保存作品草稿；默认内存存储，配置后写入 JSON 文件 |
| `GET /api/v1/calligraphy/artworks/drafts` | 已实现 | 查询认证用户的草稿列表；所属用户不匹配会被拒绝 |
| `DELETE /api/v1/calligraphy/artworks/drafts/{id}` | 已实现 | 删除一个试用草稿 |
| `POST /api/v1/calligraphy/artworks/drafts/{id}/exports` | 已实现 | 生成 SVG 参考导出并计算 SHA256；默认内联返回，配置后写入本地产物文件 |
| `GET /api/v1/calligraphy/users/{id}/learning` | 已实现 | 返回收藏字、近期练习记录和学习统计 |
| `POST /api/v1/calligraphy/users/{id}/favorites` | 已实现 | 将已发布字形收藏到学习者账号 |
| `DELETE /api/v1/calligraphy/users/{id}/favorites/{glyph_id}` | 已实现 | 移除一个收藏字 |
| `POST /api/v1/calligraphy/users/{id}/practice` | 已实现 | 记录一次单字练习动作，包含模板类型和格线类型 |
| `GET /artifacts/{storage_key}` | 已实现 | 配置 `CALLIGRAPHY_EXPORT_DIR` 后提供本地 SVG 导出下载 |
| 静态试用工作台 | 已实现 | 通过 `CALLIGRAPHY_WEB_DIR` 托管 `web/app`；支持本地注册/登录、常用字、查字/详情、练习模板预览/下载、收藏、练习记录、学习档案、章法预览、保存、列表、载入、删除、导出历史和 SVG 下载 |

该服务可将本地用户、草稿、学习记录、审计日志和 SVG 导出保存到本地文件，用于受控生产试用。用户草稿、收藏、练习和学习档案接口都要求 Bearer 令牌，并拒绝 所属用户不匹配的请求；连续登录失败会触发临时锁定。托管底座模式会在启动前校验 PostgreSQL、Identity、对象存储和审计接收端 配置，使用 PostgreSQL 保存用户/会话和学习资产，校验 Nebula Identity 兼容的 JWKS/RS256 或 HS256 Bearer 令牌，向浏览器暴露安全的运行时认证配置，托管 Web 登录优先使用 OIDC Authorization Code + PKCE，Nebula Identity 直连登录保留为兼容回退，并通过 S3 兼容对象存储写入导出产物。面向大规模商业生产仍需要授权碑帖入库，以及选定云服务的运维运行手册。

## 视觉和字体策略

当前试用版采用系统 UI 字体加书法展示字体：

- 中文 UI：使用设备系统字体，不在 Web 包内捆绑完整 CJK 字体，降低首访下载体积。
- Ma Shan Zheng：OFL 许可的书法展示字体，用于临摹参考字和章法预览，避免生产 Web 依赖设备系统楷体。

Ma Shan Zheng 只是当前无授权碑帖裁切图时的视觉兜底。正式内容生产仍应以授权碑帖高清图、单字裁切、书体来源标注和专家审核为准，不能把通用展示字体当作真实欧体、颜体、柳体或赵体字库。

## 碑帖范字流水线

当前已落地 V1 范字库 manifest 流水线：

- `assets/copybooks/jiuchenggong/manifest.sample.json`：九成宫 manifest 样例，坐标为格式示例，全部保持 `draft`，不会作为已发布字库返回。
- `docs/contracts/glyph-manifest-v1.json`：机器可读契约，定义碑帖来源、授权、单字裁切框和审校状态。
- `services/calligraphy/cmd/calligraphy-glyph-manifest`：导入前校验工具。
- `CALLIGRAPHY_GLYPH_MANIFEST_FILE`：运行时加载 manifest 字库，优先于内置兜底字库。

manifest 必须包含 `source_url`、`license_status`、`attribution`、`source_image` 和正数裁切框。只有 `review_status=published` 且非 `restricted` 的字会对外服务。AI 补字、部件合成字和人工重绘字后续必须单独标注，不能混入原碑裁切字。

## 底座集成

| 底座能力 | 用途 |
|----------|------|
| Identity | 用户账号、会员关系、个人工作空间 |
| 对象存储 | 碑帖图片、单字裁切、导出产物、用户作品 |
| AI 网关 | OCR、相似度、点评、字形生成，全部通过 功能开关 控制 |
| 审计 | 导出、AI 生成、管理端发布、涉及版权授权的操作 |
| 开放平台 | 后续字库、排版和导出 API |

## 非目标

- 不属于 Signage 排程、Player 播放、Device Hub OTA 或 RemoteOps 流程。
- MVP 不发布社区功能。
- 在专家验证评价体系可用前，不宣称 AI 评分具有权威性。
