import { useState } from 'react';
import { Alert, Button, Card, Form, Input, Space, Typography, message } from 'antd';
import { SafetyCertificateOutlined } from '@ant-design/icons';
import { HttpUtil } from '@/utils';
import './ActivationPage.css';

const errorText: Record<string, string> = {
  invite_invalid: '邀请码无效',
  invite_expired: '邀请码已过期',
  invite_disabled: '邀请码已被禁用',
  invite_used_up: '邀请码已用完',
  device_already_bound: '当前设备已绑定',
  rate_limited: '请求过于频繁，请稍后再试',
  cloud_unreachable: '无法连接 Cloud',
  cloud_not_configured: '未配置 Cloud 地址',
  cloud_url_must_be_https: 'Cloud 地址必须使用 HTTPS',
  invalid_cloud_url: 'Cloud 地址无效',
};

type Props = {
  cloudUrl?: string;
  reason?: string;
  onActivated: () => void;
};

export default function ActivationPage({ cloudUrl, reason, onActivated }: Props) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(reason || '');
  const [messageApi, contextHolder] = message.useMessage();

  const submit = async (values: { invite_code: string }) => {
    setLoading(true);
    setError('');
    try {
      const resp = await HttpUtil.post<{ activated?: boolean; server_id?: string; error?: string }>(
        '/panel/api/stella/activation/claim',
        { invite_code: values.invite_code },
        { silent: true, headers: { 'Content-Type': 'application/json' } },
      );
      if (!resp.success || !resp.obj?.activated) {
        const code = resp.obj?.error || resp.msg || 'cloud_unreachable';
        setError(code);
        messageApi.error(errorText[code] || code);
        return;
      }
      messageApi.success('激活成功');
      onActivated();
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="activation-page">
      {contextHolder}
      <Card className="activation-card">
        <Space direction="vertical" size={18} className="activation-stack">
          <div className="activation-icon"><SafetyCertificateOutlined /></div>
          <div>
            <Typography.Title level={2}>激活 StellaGate-UI</Typography.Title>
            <Typography.Paragraph type="secondary">
              请输入邀请码以开启当前 VPS 上的 StellaGate-UI。
            </Typography.Paragraph>
          </div>
          {cloudUrl ? (
            <Alert type="info" showIcon message={`Cloud 地址：${cloudUrl}`} />
          ) : (
            <Alert type="warning" showIcon message="未配置 Cloud 地址" />
          )}
          {error && <Alert type="error" showIcon message={errorText[error] || error} />}
          <Form layout="vertical" onFinish={submit} className="activation-form">
            <Form.Item
              name="invite_code"
              label="邀请码"
              rules={[{ required: true, message: '请输入邀请码' }]}
            >
              <Input size="large" placeholder="SGC-XXXX-XXXX-XXXX" autoComplete="off" />
            </Form.Item>
            <Button type="primary" size="large" block loading={loading} htmlType="submit" disabled={!cloudUrl}>
              激活
            </Button>
          </Form>
        </Space>
      </Card>
    </main>
  );
}
