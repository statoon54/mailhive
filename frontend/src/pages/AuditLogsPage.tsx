import { useEffect, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import api, { type AuditLog, type PaginatedList } from '../api/client';
import { useAuth } from '../contexts/AuthContext';

interface MailAuditRecipient {
  email: string;
  name?: string;
  type: string;
  sujet?: string;
}

interface MailAuditDetails {
  destinataires?: MailAuditRecipient[];
  total_destinataires?: number;
  total_mails?: number;
  sujet?: string;
  text_body?: string;
  html_body?: string;
}

function parseDetails(details: string): MailAuditDetails | null {
  if (!details) return null;
  try {
    return JSON.parse(details) as MailAuditDetails;
  } catch {
    return null;
  }
}

function RecipientsCell({ details }: { details: string }) {
  const { t } = useTranslation();
  const parsed = parseDetails(details);
  if (!parsed?.destinataires) return <span className="text-gray-400">—</span>;

  const all = parsed.destinataires;
  const total = parsed.total_destinataires ?? all.length;
  const maxVisible = 2;
  const visible = all.slice(0, maxVisible);
  const hiddenCount = total - maxVisible;

  const formatRecipient = (d: MailAuditRecipient) => {
    const label = d.name ? `${d.name} <${d.email}>` : d.email;
    const suffix = d.type !== 'to' ? ` (${d.type})` : '';
    return label + suffix;
  };

  const tooltipText = all.map(formatRecipient).join('\n')
    + (total > all.length ? `\n+${total - all.length} ${t(total - all.length > 1 ? 'audit.others_plural' : 'audit.others', { count: total - all.length })}` : '');

  return (
    <Tooltip content={tooltipText}>
      <div className="space-y-0.5">
        {visible.map((d, i) => (
          <div key={i} className="truncate">
            {d.name ? (
              <><span className="font-medium text-gray-700">{d.name}</span>{' '}<span className="text-gray-400">&lt;{d.email}&gt;</span></>
            ) : (
              <span className="text-gray-600">{d.email}</span>
            )}
            {d.type !== 'to' && <span className="ml-1 text-xs text-gray-400 uppercase">{d.type}</span>}
          </div>
        ))}
        {hiddenCount > 0 && (
          <div className="text-gray-400 text-xs">
            {t(hiddenCount > 1 ? 'audit.others_plural' : 'audit.others', { count: hiddenCount })}
          </div>
        )}
      </div>
    </Tooltip>
  );
}

function ContentCell({ details }: { details: string }) {
  const { t } = useTranslation();
  const parsed = parseDetails(details);
  if (!parsed) return <span className="text-gray-400">—</span>;

  const parts: string[] = [];
  if (parsed.sujet) parts.push(`${t('audit.subject_label')}: ${parsed.sujet}`);
  if (parsed.total_mails && parsed.total_mails > 1) parts.push(t('audit.mails_count', { count: parsed.total_mails }));

  if (parts.length === 0) return <span className="text-gray-400">—</span>;

  const fullText = parts.join('\n');

  return (
    <Tooltip content={fullText}>
      <div className="truncate">
        <span className="text-gray-600">{parts.join(' | ')}</span>
      </div>
    </Tooltip>
  );
}

function Tooltip({ content, children }: { content: string; children: ReactNode }) {
  const [visible, setVisible] = useState(false);

  return (
    <span
      className="relative"
      onMouseEnter={() => setVisible(true)}
      onMouseLeave={() => setVisible(false)}
    >
      {children}
      {visible && (
        <span className="absolute z-50 bottom-full left-0 mb-2 px-3 py-2 text-xs text-white bg-gray-900 rounded-lg shadow-lg whitespace-pre-wrap max-w-sm max-h-48 overflow-y-auto wrap-break-word">
          {content}
        </span>
      )}
    </span>
  );
}

export default function AuditLogsPage() {
  const { t, i18n } = useTranslation();
  const { isAdmin } = useAuth();
  const [logs, setLogs] = useState<PaginatedList<AuditLog> | null>(null);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('');
  const [resourceTypeFilter, setResourceTypeFilter] = useState('');
  const [loading, setLoading] = useState(true);

  const locale = i18n.language === 'fr' ? 'fr-FR' : 'en-US';
  const resourceTypes = ['template', 'smtp_config', 'mail', 'branding', 'tenant'] as const;

  useEffect(() => {
    setLoading(true);
    const params = new URLSearchParams({ page: String(page), limit: '20' });
    if (statusFilter) params.set('status', statusFilter);
    if (resourceTypeFilter) params.set('resource_type', resourceTypeFilter);

    const url = isAdmin ? '/admin/audit-logs' : '/audit-logs';
    api.get(`${url}?${params}`)
      .then((res) => setLogs(res.data.data))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [page, statusFilter, resourceTypeFilter]);

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900">{t('audit.title')}</h2>
        <div className="flex gap-2">
          <select
            value={statusFilter}
            onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm"
          >
            <option value="">{t('common.all_statuses')}</option>
            <option value="success">{t('audit.status.success')}</option>
            <option value="error">{t('audit.status.error')}</option>
          </select>
          <select
            value={resourceTypeFilter}
            onChange={(e) => { setResourceTypeFilter(e.target.value); setPage(1); }}
            className="px-3 py-2 border border-gray-300 rounded-lg text-sm"
          >
            <option value="">{t('audit.all_types')}</option>
            {resourceTypes.map((key) => (
              <option key={key} value={key}>{t(`audit.resource_type.${key}`)}</option>
            ))}
          </select>
        </div>
      </div>

      {loading ? (
        <p className="text-gray-500">{t('common.loading')}</p>
      ) : (
        <>
          <div className="bg-white rounded-xl shadow-sm overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-gray-600">
                <tr>
                  <th className="text-left px-4 py-3 font-medium">{t('audit.table.date')}</th>
                  {isAdmin && <th className="text-left px-4 py-3 font-medium">{t('audit.table.tenant')}</th>}
                  <th className="text-left px-4 py-3 font-medium">{t('audit.table.action')}</th>
                  <th className="text-left px-4 py-3 font-medium">{t('audit.table.type')}</th>
                  <th className="text-left px-4 py-3 font-medium">{t('audit.table.status')}</th>
                  <th className="text-left px-4 py-3 font-medium">{t('audit.table.recipients')}</th>
                  <th className="text-left px-4 py-3 font-medium">{t('audit.table.content')}</th>
                  <th className="text-left px-4 py-3 font-medium">{t('audit.table.error')}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {logs?.items?.map((log) => {
                  const statusStyle = log.status === 'success'
                    ? 'bg-emerald-100 text-emerald-700'
                    : log.status === 'error'
                      ? 'bg-red-100 text-red-700'
                      : 'bg-gray-100';
                  return (
                    <tr key={log.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 text-gray-500 whitespace-nowrap">
                        {new Date(log.created_at).toLocaleDateString(locale, {
                          day: '2-digit', month: '2-digit', year: 'numeric',
                          hour: '2-digit', minute: '2-digit',
                        })}
                      </td>
                      {isAdmin && <td className="px-4 py-3 text-gray-700 font-medium">{log.tenant_name}</td>}
                      <td className="px-4 py-3 text-gray-600">{t(`audit.action.${log.action}`, log.action)}</td>
                      <td className="px-4 py-3 text-gray-600">{t(`audit.resource_type.${log.resource_type}`, log.resource_type)}</td>
                      <td className="px-4 py-3">
                        <span className={`px-2 py-1 rounded-full text-xs font-medium ${statusStyle}`}>
                          {t(`audit.status.${log.status}`, log.status)}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-xs max-w-56">
                        <RecipientsCell details={log.details} />
                      </td>
                      <td className="px-4 py-3 text-xs max-w-64">
                        <ContentCell details={log.details} />
                      </td>
                      <td className="px-4 py-3 text-xs max-w-48">
                        {log.error_message ? (
                          <Tooltip content={log.error_message}>
                            <span className="text-red-600 truncate block">{log.error_message}</span>
                          </Tooltip>
                        ) : (
                          <span className="text-gray-400">—</span>
                        )}
                      </td>
                    </tr>
                  );
                })}
                {(!logs?.items || logs.items.length === 0) && (
                  <tr><td colSpan={isAdmin ? 8 : 7} className="px-4 py-8 text-center text-gray-400">{t('audit.no_log')}</td></tr>
                )}
              </tbody>
            </table>
          </div>

          {logs && logs.total_pages > 1 && (
            <div className="flex justify-center gap-2 mt-4">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-3 py-1 border rounded disabled:opacity-40 cursor-pointer"
              >
                {t('common.previous')}
              </button>
              <span className="px-3 py-1 text-sm text-gray-600">
                {t('common.page', { current: page, total: logs.total_pages })}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(logs.total_pages, p + 1))}
                disabled={page === logs.total_pages}
                className="px-3 py-1 border rounded disabled:opacity-40 cursor-pointer"
              >
                {t('common.next')}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
