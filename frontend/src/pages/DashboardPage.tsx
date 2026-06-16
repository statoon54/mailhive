import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  PieChart, Pie, Cell, BarChart, Bar, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, Legend,
} from 'recharts';
import api, { type MailStats, type Tenant, type PaginatedList, type Mail, type TenantMailStats, type QueueInfo } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import { renderSubject } from '../utils/renderSubject';

const TENANT_COLORS = ['#6366f1', '#8b5cf6', '#ec4899', '#f43f5e', '#f97316', '#eab308', '#22c55e', '#14b8a6', '#06b6d4', '#3b82f6'];

export default function DashboardPage() {
  const { t, i18n } = useTranslation();
  const { isAdmin } = useAuth();
  const [stats, setStats] = useState<MailStats | null>(null);
  const [tenantCount, setTenantCount] = useState<number>(0);
  const [recentMails, setRecentMails] = useState<Mail[]>([]);
  const [tenantStats, setTenantStats] = useState<TenantMailStats[]>([]);
  const [queueStats, setQueueStats] = useState<QueueInfo[]>([]);
  const [loading, setLoading] = useState(true);

  const locale = i18n.language === 'fr' ? 'fr-FR' : 'en-US';

  const fetchDashboard = () => {
    const requests: Promise<void>[] = [
      api.get('/mails/stats').then((res) => setStats(res.data.data)).catch(() => {}),
      api.get('/mails?limit=5').then((res) => {
        const data = res.data.data as PaginatedList<Mail>;
        setRecentMails(data.items || []);
      }).catch(() => {}),
    ];
    if (isAdmin) {
      requests.push(
        api.get('/admin/tenants?limit=1').then((res) => {
          const data = res.data.data as PaginatedList<Tenant>;
          setTenantCount(data.total);
        }).catch(() => {}),
        api.get('/admin/stats/by-tenant').then((res) => {
          setTenantStats(res.data.data || []);
        }).catch(() => {}),
      );
    }
    Promise.all(requests).finally(() => setLoading(false));
  };

  const fetchQueues = () => {
    const url = isAdmin ? '/admin/queues' : '/queues';
    api.get(url).then((res) => {
      setQueueStats(res.data.data || []);
    }).catch(() => {});
  };

  const handleCancelMail = async (mailId: string) => {
    await api.post(`/mails/${mailId}/cancel`);
    fetchDashboard();
  };

  // Activité en cours : tant qu'il reste des mails non terminaux, le tableau de
  // bord évolue encore ; une fois tout au repos, on espace le rafraîchissement.
  const hasActiveMails = !!stats && stats.pending + stats.queued + stats.sending > 0;
  const refreshSeconds = hasActiveMails ? 2 : 15;

  // Chargement initial.
  useEffect(() => {
    fetchDashboard();
    fetchQueues();
  }, []);

  // Rafraîchissement adaptatif (rapide si activité, lent sinon), suspendu quand
  // l'onglet est masqué et relancé immédiatement au retour.
  useEffect(() => {
    const tick = () => {
      if (document.visibilityState !== 'visible') return;
      fetchDashboard();
      fetchQueues();
    };
    const interval = setInterval(tick, refreshSeconds * 1_000);
    document.addEventListener('visibilitychange', tick);
    return () => {
      clearInterval(interval);
      document.removeEventListener('visibilitychange', tick);
    };
  }, [refreshSeconds]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
      </div>
    );
  }

  const barData = stats ? [
    { name: t('dashboard.sent'), valeur: stats.sent, fill: '#10b981' },
    { name: t('dashboard.pending'), valeur: stats.pending + stats.queued, fill: '#f59e0b' },
    { name: t('dashboard.sending'), valeur: stats.sending, fill: '#3b82f6' },
    { name: t('dashboard.failed'), valeur: stats.failed, fill: '#ef4444' },
    { name: t('dashboard.cancelled'), valeur: stats.cancelled, fill: '#6b7280' },
    { name: t('dashboard.rejected'), valeur: stats.rejected, fill: '#f97316' },
  ] : [];

  const pieData = barData.filter((d) => d.valeur > 0);

  const summaryCards = [
    { label: t('dashboard.total_mails'), value: stats?.total ?? 0, icon: '✉', color: 'bg-indigo-50 text-indigo-700 border-indigo-200' },
    { label: t('dashboard.sent'), value: stats?.sent ?? 0, icon: '✓', color: 'bg-emerald-50 text-emerald-700 border-emerald-200' },
    { label: t('dashboard.failed'), value: stats?.failed ?? 0, icon: '✕', color: 'bg-red-50 text-red-700 border-red-200' },
    { label: t('dashboard.rejected'), value: stats?.rejected ?? 0, icon: '⊘', color: 'bg-orange-50 text-orange-700 border-orange-200' },
    ...(isAdmin ? [{ label: t('dashboard.tenants'), value: tenantCount, icon: '⊞', color: 'bg-violet-50 text-violet-700 border-violet-200' }] : []),
  ];

  const successRate = stats && stats.total > 0
    ? ((stats.sent / stats.total) * 100).toFixed(1)
    : '0.0';

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900">{t('dashboard.title')}</h2>
        <span className="text-xs text-gray-400">{t('dashboard.auto_refresh', { seconds: refreshSeconds })}</span>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4 mb-8">
        {summaryCards.map((card) => (
          <div key={card.label} className={`rounded-xl p-5 border ${card.color}`}>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium opacity-75">{card.label}</p>
                <p className="text-3xl font-bold mt-1">{card.value}</p>
              </div>
              <span className="text-3xl opacity-30">{card.icon}</span>
            </div>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        <div className="bg-white rounded-xl shadow-sm p-6">
          <h3 className="font-semibold text-gray-900 mb-4">{t('dashboard.status_distribution')}</h3>
          {stats && stats.total > 0 ? (
            <ResponsiveContainer width="100%" height={280}>
              <BarChart data={barData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="name" tick={{ fontSize: 12 }} />
                <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
                <Tooltip
                  contentStyle={{ borderRadius: '8px', border: '1px solid #e5e7eb' }}
                  formatter={(value) => [value, 'Mails']}
                />
                <Bar dataKey="valeur" radius={[6, 6, 0, 0]}>
                  {barData.map((entry, i) => (
                    <Cell key={i} fill={entry.fill} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-64 text-gray-400">
              {t('common.noData')}
            </div>
          )}
        </div>

        <div className="bg-white rounded-xl shadow-sm p-6">
          <h3 className="font-semibold text-gray-900 mb-4">{t('dashboard.overview')}</h3>
          {pieData.length > 0 ? (
            <ResponsiveContainer width="100%" height={280}>
              <PieChart>
                <Pie
                  data={pieData}
                  cx="35%"
                  cy="50%"
                  innerRadius={60}
                  outerRadius={100}
                  paddingAngle={3}
                  dataKey="valeur"
                  nameKey="name"
                >
                  {pieData.map((entry, i) => (
                    <Cell key={i} fill={entry.fill} stroke="white" strokeWidth={2} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{ borderRadius: '8px', border: '1px solid #e5e7eb' }}
                  formatter={(value) => [value, 'Mails']}
                />
                <Legend
                  layout="vertical"
                  align="right"
                  verticalAlign="middle"
                  iconType="circle"
                  formatter={(value: string) => <span className="text-sm text-gray-600">{value}</span>}
                />
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-64 text-gray-400">
              {t('common.noData')}
            </div>
          )}
          {stats && stats.total > 0 && (
            <div className="text-center -mt-4">
              <p className="text-sm text-gray-500">{t('dashboard.success_rate')}</p>
              <p className="text-2xl font-bold text-emerald-600">{successRate}%</p>
            </div>
          )}
        </div>
      </div>

      {isAdmin && tenantStats.length > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
          <div className="bg-white rounded-xl shadow-sm p-6">
            <h3 className="font-semibold text-gray-900 mb-4">{t('dashboard.mails_by_tenant')}</h3>
            <ResponsiveContainer width="100%" height={280}>
              <BarChart data={tenantStats} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="tenant_name" tick={{ fontSize: 12 }} />
                <YAxis allowDecimals={false} tick={{ fontSize: 12 }} />
                <Tooltip contentStyle={{ borderRadius: '8px', border: '1px solid #e5e7eb' }} />
                <Legend verticalAlign="bottom" iconType="circle" />
                <Bar dataKey="sent" name={t('dashboard.sent')} stackId="a" fill="#10b981" radius={[0, 0, 0, 0]} />
                <Bar dataKey="pending" name={t('dashboard.pending')} stackId="a" fill="#f59e0b" />
                <Bar dataKey="failed" name={t('dashboard.failed')} stackId="a" fill="#ef4444" radius={[6, 6, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>

          <div className="bg-white rounded-xl shadow-sm p-6">
            <h3 className="font-semibold text-gray-900 mb-4">{t('dashboard.distribution_by_tenant')}</h3>
            {tenantStats.filter((ts) => ts.total > 0).length > 0 ? (
              <ResponsiveContainer width="100%" height={280}>
                <PieChart>
                  <Pie
                    data={tenantStats.filter((ts) => ts.total > 0)}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={100}
                    paddingAngle={3}
                    dataKey="total"
                    nameKey="tenant_name"
                  >
                    {tenantStats.filter((ts) => ts.total > 0).map((_entry, i) => (
                      <Cell key={i} fill={TENANT_COLORS[i % TENANT_COLORS.length]} stroke="white" strokeWidth={2} />
                    ))}
                  </Pie>
                  <Tooltip
                    contentStyle={{ borderRadius: '8px', border: '1px solid #e5e7eb' }}
                    formatter={(value) => [value, 'Mails']}
                  />
                  <Legend
                    verticalAlign="bottom"
                    iconType="circle"
                    formatter={(value: string) => <span className="text-sm text-gray-600">{value}</span>}
                  />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex items-center justify-center h-64 text-gray-400">
                {t('dashboard.no_mail_sent')}
              </div>
            )}
          </div>
        </div>
      )}

      <div className="bg-white rounded-xl shadow-sm p-6 mb-8">
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-semibold text-gray-900">{t('dashboard.asynq_queues')}</h3>
          <span className="text-xs text-gray-400">{t('dashboard.auto_refresh', { seconds: refreshSeconds })}</span>
        </div>
        {queueStats.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-gray-500 border-b">
                <tr>
                  <th className="text-left py-2 font-medium">{t('dashboard.queue.name')}</th>
                  <th className="text-right py-2 font-medium">{t('dashboard.queue.active')}</th>
                  <th className="text-right py-2 font-medium">{t('dashboard.queue.pending')}</th>
                  <th className="text-right py-2 font-medium">{t('dashboard.queue.scheduled')}</th>
                  <th className="text-right py-2 font-medium">{t('dashboard.queue.retry')}</th>
                  <th className="text-right py-2 font-medium">{t('dashboard.queue.archived')}</th>
                  <th className="text-right py-2 font-medium">{t('dashboard.queue.processed')}</th>
                  <th className="text-right py-2 font-medium">{t('dashboard.queue.failed')}</th>
                  <th className="text-right py-2 font-medium">{t('dashboard.queue.latency')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-50">
                {queueStats.map((q) => {
                  const total = q.active + q.pending + q.retry + q.archived;
                  return (
                    <tr key={q.name} className="hover:bg-gray-50">
                      <td className="py-2.5 font-medium flex items-center gap-2">
                        <span className={`inline-block w-2 h-2 rounded-full ${q.paused ? 'bg-red-500' : 'bg-emerald-500'}`} />
                        {q.name}
                      </td>
                      <td className="py-2.5 text-right">{q.active}</td>
                      <td className="py-2.5 text-right">{q.pending}</td>
                      <td className="py-2.5 text-right">{q.scheduled}</td>
                      <td className="py-2.5 text-right">{q.retry}</td>
                      <td className="py-2.5 text-right">{q.archived}</td>
                      <td className="py-2.5 text-right">{q.processed}</td>
                      <td className="py-2.5 text-right">{q.failed}</td>
                      <td className="py-2.5 text-right">{q.latency_ms} ms</td>
                      {total > 0 && (
                        <td className="py-2.5 pl-4 w-32">
                          <div className="flex h-2 rounded-full overflow-hidden bg-gray-100">
                            {q.active > 0 && <div className="bg-blue-500" style={{ width: `${(q.active / total) * 100}%` }} />}
                            {q.pending > 0 && <div className="bg-amber-400" style={{ width: `${(q.pending / total) * 100}%` }} />}
                            {q.retry > 0 && <div className="bg-orange-500" style={{ width: `${(q.retry / total) * 100}%` }} />}
                            {q.archived > 0 && <div className="bg-red-500" style={{ width: `${(q.archived / total) * 100}%` }} />}
                          </div>
                        </td>
                      )}
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        ) : (
          <p className="text-gray-400 text-center py-6">{t('dashboard.no_queue')}</p>
        )}
      </div>

      <div className="bg-white rounded-xl shadow-sm p-6">
        <h3 className="font-semibold text-gray-900 mb-4">{t('dashboard.recent_mails')}</h3>
        {recentMails.length > 0 ? (
          <table className="w-full text-sm">
            <thead className="text-gray-500 border-b">
              <tr>
                <th className="text-left py-2 font-medium">{t('dashboard.table.subject')}</th>
                <th className="text-left py-2 font-medium">{t('dashboard.table.recipient')}</th>
                <th className="text-left py-2 font-medium">{t('dashboard.table.status')}</th>
                <th className="text-left py-2 font-medium">{t('dashboard.table.date')}</th>
                <th className="py-2 font-medium"></th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {recentMails.map((mail) => {
                const toRecipients = mail.recipients?.filter((r) => r.type === 'to') || [];
                const firstTo = toRecipients[0];
                const recipientLabel = firstTo
                  ? (firstTo.name ? `${firstTo.name} <${firstTo.email}>` : firstTo.email)
                  : '-';
                const extraCount = toRecipients.length - 1;

                return (
                  <tr key={mail.id} className="hover:bg-gray-50">
                    <td className="py-2.5">
                      <Link to={`/mails/${mail.id}`} className="text-indigo-600 hover:underline">
                        {renderSubject(mail.subject, mail.template_data) || t('dashboard.no_subject')}
                      </Link>
                    </td>
                    <td className="py-2.5 text-gray-500">
                      {recipientLabel}
                      {extraCount > 0 && (
                        <span className="ml-1 text-xs text-gray-400">+{extraCount}</span>
                      )}
                    </td>
                    <td className="py-2.5">
                      <StatusBadge status={mail.status} />
                    </td>
                    <td className="py-2.5 text-gray-400">
                      {new Date(mail.created_at).toLocaleString(locale, {
                        day: '2-digit', month: '2-digit', year: 'numeric',
                        hour: '2-digit', minute: '2-digit', second: '2-digit',
                      })}
                    </td>
                    <td className="py-2.5 text-right">
                      {(mail.status === 'pending' || mail.status === 'queued') && (
                        <button
                          onClick={() => handleCancelMail(mail.id)}
                          className="px-2 py-1 text-xs bg-red-100 text-red-700 rounded-lg hover:bg-red-200 cursor-pointer"
                        >
                          {t('common.cancel')}
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        ) : (
          <p className="text-gray-400 text-center py-6">{t('dashboard.no_recent_mail')}</p>
        )}
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const { t } = useTranslation();
  const styles: Record<string, string> = {
    pending: 'bg-gray-100 text-gray-700',
    queued: 'bg-amber-100 text-amber-700',
    sending: 'bg-blue-100 text-blue-700',
    sent: 'bg-emerald-100 text-emerald-700',
    failed: 'bg-red-100 text-red-700',
    cancelled: 'bg-gray-200 text-gray-600',
  };

  return (
    <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${styles[status] || 'bg-gray-100'}`}>
      {t(`status.${status}`, status)}
    </span>
  );
}
