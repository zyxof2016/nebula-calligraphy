# Nebula Calligraphy Flutter 客户端

这是书法 C 端应用的 Flutter 入口，面向手机、平板和 Web 试用场景。当前版本已接入 `services/calligraphy` API，覆盖登录/注册、查字、单字学习、集字章法、草稿保存和参考 SVG 导出主流程。

## 本地启动

先启动后端服务：

```bash
cd /home/administrator/projects/nebula/nebula-calligraphy/services/calligraphy
go run ./cmd/calligraphy
```

再启动 Flutter 客户端：

```bash
cd /home/administrator/projects/nebula/nebula-calligraphy/apps/mobile
flutter run -d web-server --web-hostname 0.0.0.0 --web-port 8088 --dart-define=CALLIGRAPHY_API_BASE_URL=http://localhost:8090
```

移动端真机或模拟器访问本机后端时，需要把 `CALLIGRAPHY_API_BASE_URL` 换成设备可访问的局域网地址。

trial 模式默认放行 `http://localhost:8088` 和 `http://127.0.0.1:8088`，方便 Flutter Web 调试。生产或托管环境如需跨源访问，应显式设置：

```bash
CALLIGRAPHY_ALLOWED_ORIGINS=https://calligraphy.example
```

## 已实现能力

- 本地账号登录和注册，令牌会保存在本地会话存储中。
- 常用字速查和碑帖字形检索。
- 单字详情展示，包括结构要点、笔法要点和临摹记录入口。
- 集字创作表单，支持书体和幅式选择。
- 章法预览画布，按后端返回坐标绘制竖排布局、落款和印章。
- 保存作品草稿并导出参考 SVG。

## 字体资产

- UI 中文字体使用设备系统字体，不再在 Web 包内捆绑完整 CJK 字体，避免首访下载 20MB+ 字体资产。
- `MaShanZheng-Regular.ttf`：OFL 许可的书法展示字体，用于临摹参考字和章法预览。它是当前试用版的视觉兜底，不替代后续真实碑帖裁切图、授权书体和专家审核字库。

## 验证命令

```bash
flutter analyze
flutter test
flutter test test/visual_capture_test.dart --update-goldens
flutter build web --dart-define=CALLIGRAPHY_API_BASE_URL=http://localhost:8090
```

## 目录说明

```text
lib/main.dart                 # 应用入口，读取 CALLIGRAPHY_API_BASE_URL
lib/src/app.dart              # Flutter 页面与交互
lib/src/app_controller.dart   # 学习工作台状态和主流程编排
lib/src/calligraphy_api.dart  # 后端 HTTP API 客户端
lib/src/models.dart           # 与服务端契约对应的数据模型
lib/src/session_store.dart    # 会话持久化抽象与实现
test/                         # API、控制器和页面 smoke tests
```
