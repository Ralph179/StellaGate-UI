# StellaGate

StellaGate 是面向“自己买了一台 VPS”的极简代理控制台。它把成熟的
3x-ui / Xray 引擎留在底层，把日常使用收敛为一套普通用户能理解的流程：

- 我的 VPS 状态；
- 订阅导入；
- 节点重置；
- VLESS Reality / Hysteria2 协议切换；
- 流量统计。

默认首页不会展示 inbound、outbound、routing、JSON 或 Reality 参数。
普通用户只看到 StellaGate 单 VPS 控制台。

## 一键安装

完整一键安装命令：

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash
```

协议不需要在安装时选择。安装后，VLESS Reality 和 Hysteria2 都可以在
StellaGate 首页直接切换使用。

安装完成后，终端会输出：

- 面板访问地址；
- 登录用户名；
- 登录密码；
- 自动生成的订阅链接。

无需连接 Cloud、注册账号或输入邀请码。安装完成即可登录使用。

打开面板地址登录后，默认首页就是 StellaGate 控制台；手机或电脑客户端复制
订阅链接 / 扫二维码即可导入节点。

## 产品边界

StellaGate 不是机场系统：不做支付、套餐、邀请返佣、多租户营销、自动购买
VPS，也不做移动 App。第一版只管理用户自有的一台 VPS，可以完全独立部署使用。

## 本地运行

```sh
# 后端
go run .

# 前端开发
cd frontend
npm ci
npm run dev

# 生产构建（输出到 internal/web/dist）
npm run build
```

访问 `/panel/` 进入 StellaGate。

## Stella API

所有接口沿用现有面板 Session 或 API Bearer Token 鉴权。

| 方法 | 接口 | 用途 |
| --- | --- | --- |
| GET | `/panel/api/stella/vps/status` | VPS、协议与服务状态 |
| GET | `/panel/api/stella/subscription` | 订阅链接与二维码数据 |
| POST | `/panel/api/stella/subscription/reset` | 重置订阅 Token |
| POST | `/panel/api/stella/node/restart` | 重启代理服务 |
| POST | `/panel/api/stella/node/random-port` | 随机更换可用端口 |
| POST | `/panel/api/stella/node/reset` | 轻度、普通或深度重置 |
| POST | `/panel/api/stella/protocol/switch` | 切换 VLESS Reality / Hysteria2 |
| GET | `/panel/api/stella/traffic/summary` | 今日、本月与总流量 |

## 基础引擎与许可证

StellaGate 是基于 [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui)
修改而来；底层引擎能力由 StellaGate 简化产品层封装，不作为默认产品体验展示。
本仓库继续以 **GPL-3.0-or-later** 发布，保留上游版权与许可证声明。

后续一键安装的参数约定见
[`deploy/stellagate/README.md`](deploy/stellagate/README.md)。
