# StellaGate

StellaGate 是面向“自己买了一台 VPS”的极简代理控制台。它把成熟的
3x-ui / Xray 引擎留在底层，把日常使用收敛为一套普通用户能理解的流程：

- 我的 VPS 状态；
- 订阅导入；
- 节点重置；
- VLESS Reality / Hysteria2 协议切换；
- 流量统计。

默认首页不会展示 inbound、outbound、routing、JSON 或 Reality 参数。
完整引擎能力仍保留在“高级设置”，用于故障恢复和专家排查，但不再是产品的
默认入口。

## 一键安装

StellaGate-UI 必须通过 StellaGate-Cloud 邀请激活后才能使用。安装时请写入
官方 Cloud 地址：

安装脚本真实地址：

```text
https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh
```

完整一键安装命令：

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash -s -- --cloud https://stellagate.simuse.uk
```

不传邀请码时，安装后首次打开面板会显示激活页。输入作者发放的有效邀请码后，
本地面板才会解锁。

如果安装时已经拿到邀请码，也可以一次完成激活：

```sh
curl -fsSL https://raw.githubusercontent.com/Ralph179/StellaGate-UI/codex/stellagate/install.sh | bash -s -- --cloud https://stellagate.simuse.uk --invite SGC-XXXX-XXXX-XXXX
```

协议不需要在安装时选择。激活后，VLESS Reality 和 Hysteria2 都可以在
StellaGate 首页直接切换使用。

安装完成后，终端会输出：

- 面板访问地址；
- 登录用户名；
- 登录密码；
- 自动生成的订阅链接。

打开面板地址登录后，默认首页就是 StellaGate 控制台；手机或电脑客户端复制
订阅链接 / 扫二维码即可导入节点。

如果 Cloud 后台删除或撤销该激活，本地 StellaGate-UI 会复核 Cloud 状态，
重新锁定面板，禁用 StellaGate 管理节点，并重启代理服务。临时网络不可达不会
立刻误伤已激活 VPS，但 Cloud 明确判定失效时不会继续放行。

## 产品边界

StellaGate 不是机场系统：不做支付、套餐、邀请返佣、多租户营销、自动购买
VPS，也不做移动 App。第一版只管理用户自有的一台 VPS。StellaGate-UI 不能
作为无邀请的独立面板使用，必须由 StellaGate-Cloud 邀请激活。

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

访问 `/panel/` 进入 StellaGate；`/panel/advanced` 是保留的高级引擎界面。

## Stella API

所有接口沿用现有面板 Session 或 API Bearer Token 鉴权。

| 方法 | 接口 | 用途 |
| --- | --- | --- |
| GET | `/panel/api/stella/activation/status` | 本地激活状态 |
| POST | `/panel/api/stella/activation/claim` | 使用 Cloud 邀请码激活 |
| POST | `/panel/api/stella/activation/check` | 复核激活是否仍有效 |
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
修改而来；为兼容性保留其高级引擎能力，但不会把它当作 StellaGate 的默认产品
体验。本仓库继续以 **GPL-3.0-or-later** 发布，保留上游版权与许可证声明。

后续一键安装的参数约定见
[`deploy/stellagate/README.md`](deploy/stellagate/README.md)。
