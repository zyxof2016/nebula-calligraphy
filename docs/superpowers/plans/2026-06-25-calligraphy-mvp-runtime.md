# Nebula Calligraphy MVP 运行时实施计划

> **给自动化执行代理的要求：** 按任务逐项实施时，必须使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans`。步骤使用复选框（`- [ ]`）语法跟踪进度。

**目标：** 为独立的 `nebula-calligraphy` C 端产品线构建一个可编译的 MVP 后端运行时。

**架构：** 第一阶段运行单元是 `services/calligraphy` 下的小型 Go HTTP 服务。它提供单字字形查询和章法预览 API，本地验证阶段使用内存种子字库，并通过文档明确身份、存储、AI 等星云底座边界。

**技术栈：** Go 1.23、`net/http`、`go-chi/chi/v5`、表驱动 Go 测试、`docs/contracts` 下的 JSON 契约。

---

### 任务 1：领域模型

**文件：**
- 新建：`services/calligraphy/internal/model/model.go`
- 通过 service 和 handler 测试覆盖。

- [ ] 定义面向 JSON 的字形元数据、章法请求、纸张规格、章法结果和定位槽位类型。
- [ ] 字段名与 `docs/contracts/glyph-v1.json` 和 `docs/contracts/layout-v1.json` 保持一致。

### 任务 2：章法引擎

**文件：**
- 新建：`services/calligraphy/internal/service/layout.go`
- 新建：`services/calligraphy/internal/service/layout_test.go`

- [ ] 编写测试覆盖标点过滤、竖排从右到左槽位顺序、边距校验和空文本拒绝。
- [ ] 在 `services/calligraphy` 下运行 `go test ./internal/service -run Layout`，确认实现前测试失败。
- [ ] 实现 MVP 预览所需的最小确定性章法算法。
- [ ] 重新运行 service 测试并保持通过。

### 任务 3：字形目录

**文件：**
- 新建：`services/calligraphy/internal/service/glyph_catalog.go`
- 新建：`services/calligraphy/internal/service/glyph_catalog_test.go`

- [ ] 编写测试覆盖按字符、书体、碑帖过滤，以及排除受限授权字形。
- [ ] 在 `services/calligraphy` 下运行 `go test ./internal/service -run Glyph`，确认实现前测试失败。
- [ ] 实现内存种子字库和查询过滤逻辑。
- [ ] 重新运行 service 测试。

### 任务 4：HTTP API

**文件：**
- 新建：`services/calligraphy/internal/handler/handler.go`
- 新建：`services/calligraphy/internal/handler/handler_test.go`
- 新建：`services/calligraphy/cmd/calligraphy/main.go`
- 新建：`services/calligraphy/go.mod`

- [ ] 编写 handler 测试覆盖 `/health`、`/api/v1/calligraphy/glyphs/search` 和 `/api/v1/calligraphy/layouts/preview`。
- [ ] 运行 `go test ./internal/handler`，确认实现前测试失败。
- [ ] 实现 JSON 辅助函数、路由、启动流程和优雅退出。
- [ ] 重新运行 `go test ./...`。

### 任务 5：拆仓检查和文档

**文件：**
- 修改：`Makefile.split`
- 修改：`README.md`
- 修改：`SPLIT_READINESS.md`
- 修改：`docs/products/calligraphy.md`

- [ ] 让 `split-check` 校验 JSON 契约并运行 Go 测试。
- [ ] 让 `split-deploy-test` 运行运行时测试套件。
- [ ] 记录本地运行命令和 MVP API 边界。
- [ ] 在 `nebula-calligraphy` 仓库根目录运行 `make -f Makefile.split split-check`。
