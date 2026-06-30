# 星云书法体验审查记录（2026-06-30）

## 审查范围

- 访问目标：`http://1.14.208.189`
- 实际可达性：HTTP 首页返回 `200 OK`，`/health` 返回 `{"service":"calligraphy","status":"ok"}`
- 截图方式：当前环境没有 Chrome/Chromium/Firefox/Playwright runtime，使用 Flutter golden 渲染导出移动端与桌面端截图。
- 截图基线：
  - `apps/mobile/test/goldens/daily_practice_mobile.png`
  - `apps/mobile/test/goldens/daily_practice_desktop.png`
- 线上截图证据：
  - `docs/products/audits/2026-06-30-calligraphy-live/live_ip_mobile.png`
  - `docs/products/audits/2026-06-30-calligraphy-live/live_ip_desktop.png`
  - `docs/products/audits/2026-06-30-calligraphy-live/live_ip_auth_mobile.png`
  - `docs/products/audits/2026-06-30-calligraphy-live/live_ip_auth_desktop.png`

## 线上现状判断

`http://1.14.208.189` 当前仍是旧包：登录后的今日页保留“今日任务 / 选今日字 / 看临摹参考 / 我已临摹”的旧结构，临摹参考字只在小卡片中展示，没有成为首屏中心。因此用户看到“效果很差”是准确反馈。新版本已经重新打包，需要部署 `dist/nebula-calligraphy-1.14.208.189-linux-amd64.tar.gz` 后线上才会切换到新的大范字主面板。

## 发现的问题

1. 中文字体没有稳定资产时，测试和部分运行环境会退化成方框字，直接破坏书法学习场景。
2. 首页原先顶部动作和说明过多，临摹参考字没有成为首屏视觉中心。
3. 书体字段存在内部 ID 外露风险，例如 `regular_ou`。
4. 米字格、九宫格、双钩练习虽然存在，但切换后的视觉反馈不够明确。
5. 基本笔画、单字结构、多字章法需要在“临摹参考”之后形成清晰学习路径，而不是分散的小工具卡。

## 本次收口

1. 增加 Noto Serif CJK 字体资产，保证中文和参考字在 Web、移动端、测试环境中都有稳定渲染。
2. 将今日页重构为“今日临摹”主面板：大范字、米字格、碑帖来源、练习次数和主操作集中展示。
3. 练习模式切换联动参考字画布：米字格、九宫格、双钩练习各有不同显示。
4. 统一书体显示映射，避免 `regular_ou` 等内部字段暴露给用户。
5. 增加移动端和桌面端视觉回归截图测试，后续 UI 变更必须能看到真实画面。

## 当前判断

当前版本相比上一版已经从“通用后台感”收敛为“书法学习工具感”：首屏重点是范字和临摹动作，下面承接基本笔画、结构、章法。后续继续提升应优先接入真实碑帖裁图和笔画分段高亮，而不是继续堆入口。

## 二次收口

收到“手机首页内容太多、PC 浏览器样式没有适配”的反馈后，首页进一步拆成两套响应式结构：手机端仅保留今日临摹主卡，笔画、结构、章法和进度默认折叠；PC 端改为左侧导航栏、左侧任务信息、中间大字帖、右侧学习重点的工作台布局，移除桌面底部导航。
