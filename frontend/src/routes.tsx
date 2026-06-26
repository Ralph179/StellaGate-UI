import { lazy, Suspense } from 'react';
import { createBrowserRouter, Navigate, type RouteObject } from 'react-router-dom';

import PanelLayout from '@/layouts/PanelLayout';

const ActivationGate = lazy(() => import('@/pages/activation/ActivationGate'));
const IndexPage = lazy(() => import('@/pages/index/IndexPage'));
const InboundsPage = lazy(() => import('@/pages/inbounds/InboundsPage'));
const ClientsPage = lazy(() => import('@/pages/clients/ClientsPage'));
const GroupsPage = lazy(() => import('@/pages/groups/GroupsPage'));
const NodesPage = lazy(() => import('@/pages/nodes/NodesPage'));
const HostsPage = lazy(() => import('@/pages/hosts/HostsPage'));
const SettingsPage = lazy(() => import('@/pages/settings/SettingsPage'));
const XrayPage = lazy(() => import('@/pages/xray/XrayPage'));
const ApiDocsPage = lazy(() => import('@/pages/api-docs/ApiDocsPage'));

function withSuspense(node: React.ReactNode) {
  return <Suspense fallback={null}>{node}</Suspense>;
}

function gated(node: React.ReactNode) {
  return withSuspense(<ActivationGate>{node}</ActivationGate>);
}

const routes: RouteObject[] = [
  {
    path: '/',
    element: <PanelLayout />,
    children: [
      { index: true, element: withSuspense(<ActivationGate />) },
      { path: 'advanced', element: gated(<IndexPage />) },
      { path: 'inbounds', element: gated(<InboundsPage />) },
      { path: 'clients', element: gated(<ClientsPage />) },
      { path: 'groups', element: gated(<GroupsPage />) },
      { path: 'nodes', element: gated(<NodesPage />) },
      { path: 'hosts', element: gated(<HostsPage />) },
      { path: 'settings', element: gated(<SettingsPage />) },
      { path: 'xray', element: gated(<XrayPage />) },
      { path: 'outbound', element: gated(<XrayPage />) },
      { path: 'routing', element: gated(<XrayPage />) },
      { path: 'api-docs', element: gated(<ApiDocsPage />) },
      { path: '*', element: <Navigate to="/" replace /> },
    ],
  },
];

function computeBasename() {
  const raw = (typeof window !== 'undefined' && window.X_UI_BASE_PATH) || '/';
  const trimmed = raw.replace(/\/+$/, '');
  return `${trimmed}/panel`;
}

export const router = createBrowserRouter(routes, {
  basename: computeBasename(),
});
