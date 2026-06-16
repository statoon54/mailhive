import { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import api, { type Mail, type PaginatedList } from '../api/client';
import { renderSubject } from '../utils/renderSubject';

export default function MailsPage() {
  const { t, i18n } = useTranslation();
  const [mails, setMails] = useState<PaginatedList<Mail> | null>(null);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const initialLoad = useRef(true);

  const [tagsFilter, setTagsFilter] = useState('');
  const [tagMode, setTagMode] = useState<'and' | 'or'>('and');
  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');

  const locale = i18n.language === 'fr' ? 'fr-FR' : 'en-US';
  const statuses = ['pending', 'queued', 'sending', 'sent', 'failed', 'cancelled'] as const;

  // Debounce search query
  useEffect(() => {
    const timer = setTimeout(() => { setDebouncedQuery(searchQuery); setPage(1); }, 300);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  const loadMails = () => {
    if (initialLoad.current) {
      setLoading(true);
    } else {
      setRefreshing(true);
    }
    const params = new URLSearchParams({ page: String(page), limit: '20' });
    if (statusFilter) params.set('status', statusFilter);
    if (tagsFilter.trim()) {
      params.set('tags', tagsFilter.trim());
      params.set('tag_mode', tagMode);
    }
    if (debouncedQuery.trim()) params.set('q', debouncedQuery.trim());

    api.get(`/mails?${params}`)
      .then((res) => setMails(res.data.data))
      .catch(() => {})
      .finally(() => {
        setLoading(false);
        setRefreshing(false);
        initialLoad.current = false;
      });
  };

  useEffect(() => { loadMails(); }, [page, statusFilter, tagsFilter, tagMode, debouncedQuery]);

  // Un mail « actif » (pending/queued/sending) peut encore changer de statut ;
  // une fois tout terminal (sent/failed/cancelled), plus rien ne bouge côté serveur.
  const hasActiveMails = mails?.items?.some((m) =>
    m.status === 'pending' || m.status === 'queued' || m.status === 'sending',
  ) ?? false;
  const refreshSeconds = hasActiveMails ? 2 : 15;

  // Sondage adaptatif : rapide tant qu'un envoi est en cours, lent ensuite (pour
  // détecter l'arrivée de nouveaux mails). Suspendu quand l'onglet est masqué, et
  // relancé immédiatement au retour de l'onglet.
  useEffect(() => {
    const tick = () => { if (document.visibilityState === 'visible') loadMails(); };
    const interval = setInterval(tick, refreshSeconds * 1_000);
    const onVisible = () => { if (document.visibilityState === 'visible') loadMails(); };
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      clearInterval(interval);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [refreshSeconds, page, statusFilter, tagsFilter, tagMode, debouncedQuery]);

  const handleCancel = async (mailId: string) => {
    await api.post(`/mails/${mailId}/cancel`);
    loadMails();
  };

  const statusStyles: Record<string, string> = {
    pending: 'bg-gray-100 text-gray-700',
    queued: 'bg-amber-100 text-amber-700',
    sending: 'bg-blue-100 text-blue-700',
    sent: 'bg-emerald-100 text-emerald-700',
    failed: 'bg-red-100 text-red-700',
    cancelled: 'bg-gray-200 text-gray-600',
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <h2 className="text-2xl font-bold text-gray-900">{t('mails.title')}</h2>
          <span className="text-xs text-gray-400">{t('dashboard.auto_refresh', { seconds: refreshSeconds })}</span>
        </div>
        <select
          value={statusFilter}
          onChange={(e) => { setStatusFilter(e.target.value); setPage(1); }}
          className="px-3 py-2 border border-gray-300 rounded-lg text-sm"
        >
          <option value="">{t('common.all_statuses')}</option>
          {statuses.map((s) => (
            <option key={s} value={s}>{t(`status.${s}`)}</option>
          ))}
        </select>
      </div>

      <div className="flex items-center gap-3 mb-6">
        <input
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder={t('mails.search')}
          className="px-3 py-2 border border-gray-300 rounded-lg text-sm flex-1 max-w-xs"
        />
        <input
          value={tagsFilter}
          onChange={(e) => { setTagsFilter(e.target.value); setPage(1); }}
          placeholder={t('mails.filter_tags')}
          className="px-3 py-2 border border-gray-300 rounded-lg text-sm flex-1 max-w-xs"
        />
        <div className="flex items-center gap-1">
          {(['and', 'or'] as const).map((mode) => (
            <button
              key={mode}
              onClick={() => { setTagMode(mode); setPage(1); }}
              className={`px-2 py-1 text-xs rounded cursor-pointer ${tagMode === mode ? 'bg-indigo-100 text-indigo-700 font-medium' : 'bg-gray-100 text-gray-500 hover:bg-gray-200'}`}
            >
              {t(`mails.tag_mode_${mode}`)}
            </button>
          ))}
        </div>
      </div>

      {loading ? (
        <p className="text-gray-500">{t('common.loading')}</p>
      ) : (
        <>
          <div className={`bg-white rounded-xl shadow-sm overflow-x-auto transition-opacity duration-300 ${refreshing ? 'opacity-60' : 'opacity-100'}`}>
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-gray-600">
                <tr>
                  <th className="text-left px-4 py-3 font-medium">{t('mails.table.subject')}</th>
                  <th className="text-left px-4 py-3 font-medium">{t('mails.table.sender')}</th>
                  <th className="text-left px-4 py-3 font-medium">{t('mails.table.status')}</th>
                  <th className="text-left px-4 py-3 font-medium">{t('mails.table.spam')}</th>
                  <th className="text-left px-4 py-3 font-medium">{t('mails.table.tags')}</th>
                  <th className="text-left px-4 py-3 font-medium">{t('mails.table.date')}</th>
                  <th className="px-4 py-3 font-medium"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {mails?.items?.map((mail) => {
                  const style = statusStyles[mail.status] || 'bg-gray-100';
                  return (
                    <tr key={mail.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3">
                        <Link to={`/mails/${mail.id}`} className="text-indigo-600 hover:underline font-medium">
                          {renderSubject(mail.subject, mail.template_data) || t('mails.no_subject')}
                        </Link>
                      </td>
                      <td className="px-4 py-3 text-gray-600">{mail.from_email}</td>
                      <td className="px-4 py-3">
                        <span className={`px-2 py-1 rounded-full text-xs font-medium whitespace-nowrap ${style}`}>
                          {t(`status.${mail.status}`, mail.status)}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        {mail.spam_score != null && (
                          <span className={`px-1.5 py-0.5 text-xs rounded font-bold ${mail.spam_score <= 3 ? 'bg-emerald-100 text-emerald-700' : mail.spam_score <= 6 ? 'bg-amber-100 text-amber-700' : 'bg-red-100 text-red-700'}`}>
                            {mail.spam_score.toFixed(1)}
                          </span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {mail.tags && mail.tags.length > 0 && (
                          <div className="flex flex-wrap gap-1">
                            {mail.tags.map((tag) => (
                              <span key={tag} className="px-1.5 py-0.5 bg-gray-100 text-gray-600 text-xs rounded">{tag}</span>
                            ))}
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-3 text-gray-500">
                        {new Date(mail.created_at).toLocaleDateString(locale, {
                          day: '2-digit', month: '2-digit', year: 'numeric',
                          hour: '2-digit', minute: '2-digit',
                        })}
                      </td>
                      <td className="px-4 py-3 text-right">
                        {(mail.status === 'pending' || mail.status === 'queued') && (
                          <button
                            onClick={(e) => { e.preventDefault(); handleCancel(mail.id); }}
                            className="px-2 py-1 text-xs bg-red-100 text-red-700 rounded-lg hover:bg-red-200 cursor-pointer"
                          >
                            {t('common.cancel')}
                          </button>
                        )}
                      </td>
                    </tr>
                  );
                })}
                {(!mails?.items || mails.items.length === 0) && (
                  <tr><td colSpan={7} className="px-4 py-8 text-center text-gray-400">{t('mails.no_mail')}</td></tr>
                )}
              </tbody>
            </table>
          </div>

          {mails && mails.total_pages > 1 && (
            <div className="flex justify-center gap-2 mt-4">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page === 1}
                className="px-3 py-1 border rounded disabled:opacity-40 cursor-pointer"
              >
                {t('common.previous')}
              </button>
              <span className="px-3 py-1 text-sm text-gray-600">
                {t('common.page', { current: page, total: mails.total_pages })}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(mails.total_pages, p + 1))}
                disabled={page === mails.total_pages}
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
