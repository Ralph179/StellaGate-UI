import { type ReactNode, useCallback, useEffect, useState } from 'react';
import { Spin } from 'antd';
import { HttpUtil } from '@/utils';
import ActivationPage from './ActivationPage';
import StellaGatePage from '@/pages/stella/StellaGatePage';

type ActivationStatus = {
  activated: boolean;
  cloud_url: string;
  server_id?: string;
  checked_at?: number;
  reason?: string;
};

type Props = {
  children?: ReactNode;
};

export default function ActivationGate({ children }: Props) {
  const [status, setStatus] = useState<ActivationStatus | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    const resp = await HttpUtil.get<ActivationStatus>('/panel/api/stella/activation/status', undefined, { silent: true });
    if (resp.success && resp.obj) {
      setStatus(resp.obj);
    } else {
      setStatus({ activated: false, cloud_url: '', reason: resp.msg || 'cloud_unreachable' });
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => void load(), 60_000);
    return () => window.clearInterval(timer);
  }, [load]);

  if (loading) {
    return (
      <main className="activation-page">
        <Spin size="large" />
      </main>
    );
  }

  if (!status?.activated) {
    return <ActivationPage cloudUrl={status?.cloud_url} reason={status?.reason} onActivated={() => void load()} />;
  }

  return children ? <>{children}</> : <StellaGatePage />;
}
