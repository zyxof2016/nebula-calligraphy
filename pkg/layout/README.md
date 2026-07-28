# 章法排版包

多个运行时需要共享章法算法时，将公共实现迁移到这里。

当前可运行且有测试覆盖的章法实现位于
`services/calligraphy/internal/service`，由 Go API 统一提供竖排、幅式、边距、
落款和印章位置计算。
