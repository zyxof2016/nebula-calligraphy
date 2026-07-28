# 渲染包

跨服务复用的 PNG、SVG 和临摹模板导出能力后续放在这里。

当前 PNG/SVG 导出、SHA256 校验、对象存储写入和高风险审计实现在
`services/calligraphy/internal/service`。只有出现第二个运行时消费者时才迁移到共享包，
避免提前复制代码。PDF 尚未实现，不应作为当前产品能力对外承诺。
