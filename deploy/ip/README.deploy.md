# Nebula Calligraphy 裸机部署包

适用于 Ubuntu 服务器、systemd 和 Nginx，默认安装到 `/data/nebula-calligraphy`，服务用户为 `ubuntu`。

```bash
tar -xzf nebula-calligraphy-linux-amd64.tar.gz
cd calligraphy-ip-release
./deploy/update-on-server.sh
```

安装脚本会部署后端、Flutter Web、服务端字体、systemd 单元和 Nginx 配置，并执行就绪及字图渲染检查。Nginx 将 API 请求体限制为 2MB，并对本地登录和注册按来源 IP 限流；过量请求返回 `429`。
就绪检查还会验证状态目录可写、审计文件可追加和 Web 入口可读取，权限错误不会等到用户保存数据时才暴露。

默认入口为 `http://1.14.208.189/`。HTTP 只适合受控试用；公开生产必须在 Nginx 或云负载均衡上启用 HTTPS。

验收：

```bash
curl -fsS http://127.0.0.1:8090/ready
curl -fsS 'http://127.0.0.1:8090/api/v1/calligraphy/glyphs/search?character=山&style=ou'
curl -fsS http://127.0.0.1:8090/api/v1/calligraphy/glyphs/ou-jiuchenggong-shan/render.png -o /tmp/shan.png
sudo systemctl status nebula-calligraphy --no-pager
```

数据位于 `/data/nebula-calligraphy/state`。升级前应执行仓库中的 `scripts/calligraphy-backup.sh`。
