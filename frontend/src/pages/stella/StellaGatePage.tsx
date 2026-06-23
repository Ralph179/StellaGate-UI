import { useCallback, useEffect, useState } from 'react';
import { Button, Card, Col, Descriptions, Divider, Modal, Row, Select, Space, Statistic, Tag, Typography, message } from 'antd';
import { CopyOutlined, ReloadOutlined, PoweroffOutlined, SafetyCertificateOutlined, SwapOutlined, BarChartOutlined } from '@ant-design/icons';
import { HttpUtil, ClipboardManager, SizeFormatter } from '@/utils';
import './StellaGatePage.css';

type Access = { panelUrl:string; username:string; password:string; subscriptionLink:string };
type Status = { name:string; ip:string; system:string; online:boolean; protocol:string; port:number; xrayStatus:string; singBoxStatus:string; monthTraffic:{up:number;down:number;total:number}; access?:Access; checkedAt:number };
type Subscription = { link:string; token:string; qrData:string };
type Traffic = { today:{up:number;down:number;total:number}; month:{up:number;down:number;total:number}; total:{up:number;down:number;total:number}; onlineClients:number };
const bytes = (n:number) => SizeFormatter.sizeFormat(n || 0);

export default function StellaGatePage() {
  const [status, setStatus] = useState<Status | null>(null);
  const [sub, setSub] = useState<Subscription | null>(null);
  const [traffic, setTraffic] = useState<Traffic | null>(null);
  const [busy, setBusy] = useState(false);
  const [resetOpen, setResetOpen] = useState(false);
  const [protocol, setProtocol] = useState('vless-reality');
  const [messageApi, contextHolder] = message.useMessage();
  const load = useCallback(async () => {
    const [s, su, t] = await Promise.all([
      HttpUtil.get<Status>('/panel/api/stella/vps/status', undefined, { silent: true }),
      HttpUtil.get<Subscription>('/panel/api/stella/subscription', undefined, { silent: true }),
      HttpUtil.get<Traffic>('/panel/api/stella/traffic/summary', undefined, { silent: true }),
    ]);
    if (s.success) { setStatus(s.obj); setProtocol(s.obj?.protocol || 'vless-reality'); }
    if (su.success) setSub(su.obj);
    if (t.success) setTraffic(t.obj);
  }, []);
  useEffect(() => { void load(); }, [load]);
  const run = async (url:string, data?:unknown) => { setBusy(true); try { const r = await HttpUtil.post(url, data, { silent:true }); if (!r.success) throw new Error(r.msg || '操作失败'); messageApi.success('已完成'); await load(); } catch (e) { messageApi.error(e instanceof Error ? e.message : '操作失败'); } finally { setBusy(false); } };
  const copy = async () => { if (sub && await ClipboardManager.copyText(sub.link)) messageApi.success('订阅链接已复制'); };
  const current = status?.protocol === 'hysteria2' ? 'Hysteria2' : status?.protocol === 'vless-reality' ? 'VLESS Reality' : '尚未创建节点';
  return <main className="stella-page">{contextHolder}
    <header className="stella-header"><div><Typography.Title level={2}>StellaGate</Typography.Title><Typography.Text type="secondary">自建 VPS 一键代理控制台</Typography.Text></div><Button href="./advanced">高级设置</Button></header>
    <Card className="vps-card" title="我的 VPS" extra={<Space><Button icon={<ReloadOutlined />} onClick={() => void load()}>重新检测</Button><Button type="primary" loading={busy} icon={<PoweroffOutlined />} onClick={() => void run('/panel/api/stella/node/restart')}>重启服务</Button></Space>}>
      <Descriptions column={{ xs: 1, sm: 2, lg: 4 }} size="small">
        <Descriptions.Item label="VPS 名称">{status?.name || 'StellaGate VPS'}</Descriptions.Item><Descriptions.Item label="IP 地址">{status?.ip || '检测中'}</Descriptions.Item><Descriptions.Item label="系统版本">{status?.system || '—'}</Descriptions.Item><Descriptions.Item label="在线状态"><Tag color={status?.online ? 'success' : 'error'}>{status?.online ? '在线' : '服务未运行'}</Tag></Descriptions.Item>
        <Descriptions.Item label="当前协议">{current}</Descriptions.Item><Descriptions.Item label="当前端口">{status?.port || '—'}</Descriptions.Item><Descriptions.Item label="Xray 状态">{status?.xrayStatus || '未知'}</Descriptions.Item><Descriptions.Item label="本月流量">{bytes(status?.monthTraffic.total || 0)}</Descriptions.Item>
      </Descriptions>
    </Card>
    <Card className="vps-card" title="安装信息">
      <Descriptions column={{ xs: 1, sm: 2, lg: 4 }} size="small">
        <Descriptions.Item label="面板网址"><Typography.Text copyable={{ text: status?.access?.panelUrl }}>{status?.access?.panelUrl || '—'}</Typography.Text></Descriptions.Item>
        <Descriptions.Item label="用户名"><Typography.Text copyable={{ text: status?.access?.username }}>{status?.access?.username || '—'}</Typography.Text></Descriptions.Item>
        <Descriptions.Item label="初始密码"><Typography.Text copyable={!!status?.access?.password}>{status?.access?.password || '安装信息不可用'}</Typography.Text></Descriptions.Item>
        <Descriptions.Item label="自动订阅链接"><Typography.Text copyable={{ text: status?.access?.subscriptionLink || sub?.link }}>{status?.access?.subscriptionLink || sub?.link || '请先创建节点'}</Typography.Text></Descriptions.Item>
      </Descriptions>
    </Card>
    <section><Typography.Title level={3}>节点控制中心</Typography.Title><Typography.Text type="secondary">导入、重置、切换与流量，所有日常操作都在这里。</Typography.Text>
      <Row gutter={[16,16]} className="stella-grid">
        <Col xs={24} md={12}><Card title={<><CopyOutlined /> 订阅导入</>}><Typography.Paragraph type="secondary">复制链接或扫码导入 Shadowrocket、Hiddify、v2rayN、Clash Verge 与 v2rayNG。</Typography.Paragraph><div className="sub-link">{sub?.link || '请先在协议切换中创建节点'}</div><Space wrap><Button type="primary" icon={<CopyOutlined />} disabled={!sub} onClick={() => void copy()}>复制订阅链接</Button><Button loading={busy} onClick={() => void run('/panel/api/stella/subscription/reset')}>重置订阅 Token</Button></Space>{sub && <img className="stella-qr" alt="订阅二维码" src={`https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=${encodeURIComponent(sub.qrData)}`} />}</Card></Col>
        <Col xs={24} md={12}><Card title={<><SafetyCertificateOutlined /> 节点重置</>}><Typography.Paragraph type="secondary">节点失效、密钥泄露或配置异常时快速恢复。</Typography.Paragraph><Space wrap><Button loading={busy} onClick={() => void run('/panel/api/stella/node/restart')}>轻度：重启服务</Button><Button loading={busy} onClick={() => setResetOpen(true)}>普通 / 深度重置</Button></Space></Card></Col>
        <Col xs={24} md={12}><Card title={<><SwapOutlined /> 协议切换</>}><Typography.Paragraph type="secondary">当前：<b>{current}</b>。复杂参数由系统自动生成，切换后订阅自动更新。</Typography.Paragraph><Space wrap><Select value={protocol} onChange={setProtocol} options={[{value:'vless-reality',label:'VLESS Reality'},{value:'hysteria2',label:'Hysteria2'}]} /><Button type="primary" loading={busy} onClick={() => void run('/panel/api/stella/protocol/switch', { protocol })}>切换协议</Button></Space></Card></Col>
        <Col xs={24} md={12}><Card title={<><BarChartOutlined /> 流量统计</>}><Row gutter={8}><Col span={8}><Statistic title="今日总流量" value={bytes(traffic?.today.total || 0)} /></Col><Col span={8}><Statistic title="本月总流量" value={bytes(traffic?.month.total || 0)} /></Col><Col span={8}><Statistic title="在线客户端" value={traffic?.onlineClients || 0} /></Col></Row><Divider /><Typography.Text type="secondary">总上传 {bytes(traffic?.total.up || 0)} · 总下载 {bytes(traffic?.total.down || 0)}</Typography.Text></Card></Col>
      </Row>
    </section>
    <Modal title="节点重置" open={resetOpen} onCancel={() => setResetOpen(false)} footer={null}><Typography.Paragraph>普通重置会更换 UUID / Hysteria 密码；深度重置会重建当前协议的入站与订阅。</Typography.Paragraph><Space><Button onClick={() => { setResetOpen(false); void run('/panel/api/stella/node/reset',{resetType:'normal'}); }}>普通重置</Button><Button danger onClick={() => { setResetOpen(false); void run('/panel/api/stella/node/reset',{resetType:'deep'}); }}>深度重置</Button></Space></Modal>
  </main>;
}
