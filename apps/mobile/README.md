# Nebula Calligraphy Flutter 客户端

这是书法 C 端应用的 Flutter 入口，面向手机、平板和 Web 试用场景。当前版本已接入 `services/calligraphy` API，覆盖登录/注册、查字、单字学习、集字章法、草稿保存和 PNG/SVG 导出主流程。

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
Web 正式构建不指定该参数时会自动使用页面当前源，因而同一产物可以放到裸机、Docker 或 Ingress 后面。

trial 模式默认放行 `http://localhost:8088` 和 `http://127.0.0.1:8088`，方便 Flutter Web 调试。生产或托管环境如需跨源访问，应显式设置：

```bash
CALLIGRAPHY_ALLOWED_ORIGINS=https://calligraphy.example
```

## 已实现能力

- 试用环境支持本地账号登录和注册；客户端启动时会清理过期会话。
- 托管环境使用系统浏览器完成 OIDC Authorization Code + PKCE 登录，不在应用内收集或转发 Identity 密码。
- Android 使用系统安全存储，iOS 使用 Keychain 保存短期会话；损坏或过期会话会自动清理。
- 常用字速查和碑帖字形检索。
- 单字详情展示，包括结构要点、笔法要点和临摹记录入口。
- 集字创作表单，支持书体和幅式选择。
- 章法预览画布，按后端返回坐标绘制竖排布局、落款和印章。
- 保存作品草稿并导出 PNG 图片或 SVG 矢量参考。

## 字体资产

- UI 中文字体使用设备系统字体，不在 Web 包内捆绑完整 CJK 字体，避免首访下载 20MB+ 字体资产。
- 书法参考字和章法预览优先使用服务端返回的 `render_asset.url` PNG 字图。前端不再打包书法字体。
- 本地视觉测试会从仓库根目录 `assets/fonts/` 加载测试字体，保证无中文字体的 Linux 环境也能生成可读截图；该字体不进入 Flutter Web 产物。
- Web、Android 和 iOS 使用 `assets/brand/calligraphy-app-icon-master.png` 派生的星云书法品牌图标，不再使用 Flutter 默认图标。

## 验证命令

```bash
flutter analyze
flutter test
flutter test test/visual_capture_test.dart --update-goldens
flutter build web --dart-define=CALLIGRAPHY_API_BASE_URL=http://localhost:8090
```

Android release 构建必须提供独立签名：

```bash
export CALLIGRAPHY_ANDROID_KEYSTORE=/secure/calligraphy.jks
export CALLIGRAPHY_ANDROID_KEYSTORE_PASSWORD='...'
export CALLIGRAPHY_ANDROID_KEY_ALIAS=calligraphy
export CALLIGRAPHY_ANDROID_KEY_PASSWORD='...'
flutter build appbundle --release \
  --dart-define=CALLIGRAPHY_API_BASE_URL=https://calligraphy.example.com \
  --dart-define=CALLIGRAPHY_OIDC_CLIENT_ID=nebula-calligraphy-mobile \
  --dart-define=CALLIGRAPHY_OIDC_REDIRECT_URI=com.nebula.calligraphy:/oauthredirect
```

Identity 必须登记 `nebula-calligraphy-mobile=com.nebula.calligraphy:/oauthredirect`，并把该 client_id 绑定到 `nebula-calligraphy` audience。release 包禁用明文 HTTP，debug 包保留本机或 IP 联调能力。推送 `android-v<semver>` 标签或手动运行 `Release Android` 工作流，会从受保护的 `production` 环境读取签名密钥、`CALLIGRAPHY_API_BASE_URL` 和 `CALLIGRAPHY_OIDC_CLIENT_ID`，校验 AAB 签名并生成 SHA-256。
正式 AAB 的发布预算为 60 MiB；当前签名构建实测 50.4 MB，工作流会在包体异常膨胀时阻断发布。

iOS 需要在 macOS 上使用 Apple Distribution 证书和匹配 `com.nebula.calligraphy` 的 provisioning profile 完成签名。CI 会先执行无签名 release 编译，阻止 CocoaPods、AppAuth、Keychain entitlement 或回调 scheme 集成错误进入主分支：

```bash
flutter build ios --release --no-codesign \
  --dart-define=CALLIGRAPHY_API_BASE_URL=https://calligraphy.example.com \
  --dart-define=CALLIGRAPHY_OIDC_CLIENT_ID=nebula-calligraphy-mobile \
  --dart-define=CALLIGRAPHY_OIDC_REDIRECT_URI=com.nebula.calligraphy:/oauthredirect
```

## 目录说明

```text
lib/main.dart                 # 应用入口，读取 CALLIGRAPHY_API_BASE_URL
lib/src/app.dart              # Flutter 页面与交互
lib/src/app_controller.dart   # 学习工作台状态和主流程编排
lib/src/calligraphy_api.dart  # 后端 HTTP API 客户端
lib/src/models.dart           # 与服务端契约对应的数据模型
lib/src/session_store.dart    # Keychain/Android 安全会话存储
lib/src/oidc_client*.dart     # 原生系统浏览器 OIDC PKCE 适配器
test/                         # API、控制器和页面 smoke tests
```
