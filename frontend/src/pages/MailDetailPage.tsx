import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import api, { type Mail, type AttachmentRef } from '../api/client';
import { renderSubject } from '../utils/renderSubject';
import { useToast } from '../contexts/ToastContext';

export default function MailDetailPage() {
  const { t, i18n } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { addToast } = useToast();
  const [mail, setMail] = useState<Mail | null>(null);
  const [loading, setLoading] = useState(true);
  const [showBody, setShowBody] = useState(false);

  const locale = i18n.language === 'fr' ? 'fr-FR' : 'en-US';

  // Télécharge une pièce jointe via l'API (le JWT est ajouté par l'intercepteur
  // axios — un simple <a href> ne pourrait pas porter l'en-tête Authorization).
  const handleDownload = async (att: AttachmentRef) => {
    try {
      const res = await api.get(`/mails/${id}/attachments/${att.attachment_id}`, { responseType: 'blob' });
      const url = URL.createObjectURL(res.data as Blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = att.filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch {
      addToast(t('mail_detail.download_error', 'Échec du téléchargement de la pièce jointe'), 'error');
    }
  };

  // silent : rafraîchissement de fond (sondage) — ne pas réafficher l'écran de
  // chargement ni quitter la page sur une erreur transitoire.
  const loadMail = (silent = false) => {
    if (!silent) setLoading(true);
    api.get(`/mails/${id}`)
      .then((res) => setMail(res.data.data))
      .catch(() => { if (!silent) navigate('/mails'); })
      .finally(() => { if (!silent) setLoading(false); });
  };

  useEffect(() => { loadMail(); }, [id]);

  // Tant que le mail peut encore changer de statut, on le rafraîchit toutes les
  // 2 s (suspendu si l'onglet est masqué). Une fois terminal, plus de sondage.
  const isActive = mail != null
    && (mail.status === 'pending' || mail.status === 'queued' || mail.status === 'sending');
  useEffect(() => {
    if (!isActive) return;
    const tick = () => { if (document.visibilityState === 'visible') loadMail(true); };
    const interval = setInterval(tick, 2_000);
    const onVisible = () => { if (document.visibilityState === 'visible') loadMail(true); };
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      clearInterval(interval);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [isActive, id]);

  const handleCancel = async () => {
    await api.post(`/mails/${id}/cancel`);
    loadMail();
  };

  const handleRetry = async () => {
    await api.post(`/mails/${id}/retry`);
    loadMail();
  };

  if (loading || !mail) return <p className="text-gray-500">{t('common.loading')}</p>;

  return (
    <div>
      <button onClick={() => navigate('/mails')} className="text-indigo-600 text-sm mb-4 hover:underline cursor-pointer">
        ← {t('mail_detail.back')}
      </button>

      <div className="bg-white rounded-xl shadow-sm p-6">
        <div className="flex items-start justify-between mb-6">
          <div>
            <h2 className="text-xl font-bold text-gray-900">{renderSubject(mail.subject, mail.template_data) || t('mail_detail.no_subject')}</h2>
            <p className="text-gray-500 text-sm mt-1">{t('mail_detail.from')} : {mail.from_name} &lt;{mail.from_email}&gt;</p>
          </div>
          <div className="flex gap-2">
            {(mail.status === 'pending' || mail.status === 'queued') && (
              <button onClick={handleCancel} className="px-3 py-1.5 text-sm bg-red-100 text-red-700 rounded-lg hover:bg-red-200 cursor-pointer">
                {t('mail_detail.cancel')}
              </button>
            )}
            {mail.status === 'failed' && (
              <button onClick={handleRetry} className="px-3 py-1.5 text-sm bg-indigo-100 text-indigo-700 rounded-lg hover:bg-indigo-200 cursor-pointer">
                {t('mail_detail.retry')}
              </button>
            )}
          </div>
        </div>

        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
          <InfoItem label={t('mail_detail.status')} value={mail.status} />
          <InfoItem label={t('mail_detail.attempts')} value={String(mail.attempts)} />
          <InfoItem label={t('mail_detail.created_at')} value={new Date(mail.created_at).toLocaleString(locale)} />
          <InfoItem label={t('mail_detail.sent_at')} value={mail.sent_at ? new Date(mail.sent_at).toLocaleString(locale) : '-'} />
        </div>

        {mail.spam_score != null && (
          <div className="mb-6">
            <h3 className="font-semibold text-gray-700 mb-2">{t('mail_detail.spam_score')}</h3>
            <div className="flex items-center gap-3">
              <div className="w-48 h-3 bg-gray-200 rounded-full overflow-hidden">
                <div
                  className={`h-full rounded-full transition-all ${mail.spam_score <= 3 ? 'bg-emerald-500' : mail.spam_score <= 6 ? 'bg-amber-500' : 'bg-red-500'}`}
                  style={{ width: `${Math.min(100, (mail.spam_score / 10) * 100)}%` }}
                />
              </div>
              <span className={`text-sm font-bold ${mail.spam_score <= 3 ? 'text-emerald-700' : mail.spam_score <= 6 ? 'text-amber-700' : 'text-red-700'}`}>
                {mail.spam_score.toFixed(1)} / 10
              </span>
            </div>
          </div>
        )}

        {mail.tags && mail.tags.length > 0 && (
          <div className="mb-6">
            <h3 className="font-semibold text-gray-700 mb-2">{t('mail_detail.tags')}</h3>
            <div className="flex flex-wrap gap-2">
              {mail.tags.map((tag) => (
                <span key={tag} className="px-2 py-1 bg-indigo-50 text-indigo-700 rounded text-sm font-medium">{tag}</span>
              ))}
            </div>
          </div>
        )}

        {mail.status_message && (
          <div className="bg-red-50 text-red-700 p-3 rounded-lg text-sm mb-6">
            {mail.status_message}
          </div>
        )}

        {mail.recipients && mail.recipients.length > 0 && (
          <div className="mb-6">
            <h3 className="font-semibold text-gray-700 mb-2">{t('mail_detail.recipients')}</h3>
            <div className="flex flex-wrap gap-2">
              {mail.recipients.map((r) => (
                <span key={r.id} className="px-2 py-1 bg-gray-100 rounded text-sm">
                  <span className="text-gray-400 uppercase text-xs mr-1">{r.type}</span>
                  {r.name ? `${r.name} <${r.email}>` : r.email}
                </span>
              ))}
            </div>
          </div>
        )}

        {mail.attachment_refs && mail.attachment_refs.length > 0 && (
          <div className="mb-6">
            <h3 className="font-semibold text-gray-700 mb-2">{t('mail_detail.attachments', 'Pièces jointes')}</h3>
            <div className="flex flex-wrap gap-2">
              {mail.attachment_refs.map((att, i) => (
                <button
                  key={i}
                  onClick={() => handleDownload(att)}
                  title={t('mail_detail.download', 'Télécharger')}
                  className="inline-flex items-center gap-2 px-2 py-1 bg-gray-100 hover:bg-gray-200 rounded text-sm cursor-pointer"
                >
                  <span aria-hidden>📎</span>
                  <span className="font-medium text-gray-700">{att.filename}</span>
                  <span className="text-gray-400 text-xs">{att.content_type} · {formatBytes(att.size)}</span>
                  <span aria-hidden className="text-indigo-600">⬇</span>
                </button>
              ))}
            </div>
          </div>
        )}

        {(mail.text_body || mail.html_body) ? (
          <div>
            <button
              onClick={() => setShowBody(!showBody)}
              className="flex items-center gap-2 px-3 py-1.5 text-sm bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 cursor-pointer mb-4"
            >
              <span className={`transition-transform ${showBody ? 'rotate-90' : ''}`}>&#9654;</span>
              {showBody ? t('mail_detail.hide_body') : t('mail_detail.show_body')}
            </button>
            {showBody && (
              <>
                {mail.text_body && (
                  <div className="mb-6">
                    <h3 className="font-semibold text-gray-700 mb-2">{t('mail_detail.text_body')}</h3>
                    <pre className="bg-gray-50 p-4 rounded-lg text-sm whitespace-pre-wrap">{mail.text_body}</pre>
                  </div>
                )}
                {mail.html_body && (
                  <div>
                    <h3 className="font-semibold text-gray-700 mb-2">{t('mail_detail.html_body')}</h3>
                    <div className="bg-gray-50 p-4 rounded-lg border">
                      <iframe
                        srcDoc={mail.html_body}
                        title={t('mail_detail.html_preview')}
                        className="w-full h-64 border-0"
                        sandbox=""
                      />
                    </div>
                  </div>
                )}
              </>
            )}
          </div>
        ) : (mail.status === 'sent' || mail.status === 'failed') && (
          <p className="text-sm text-gray-400 italic">{t('mail_detail.body_purged')}</p>
        )}
      </div>
    </div>
  );
}

// formatBytes affiche une taille d'octets en o / Ko / Mo.
function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} o`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} Ko`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} Mo`;
}

function InfoItem({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs text-gray-400 uppercase tracking-wide">{label}</p>
      <p className="text-sm font-medium text-gray-900 mt-0.5">{value}</p>
    </div>
  );
}
