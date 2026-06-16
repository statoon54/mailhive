import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import api, { type Tenant } from '../api/client';
import { useAuth } from '../contexts/AuthContext';
import { useToast } from '../contexts/ToastContext';
import { getApiError, getFieldErrors } from '../utils/apiError';
import FormTooltip from '../components/FormTooltip';

export default function TenantsPage() {
  const { t, i18n } = useTranslation();
  const { isAdmin } = useAuth();
  const { addToast } = useToast();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [form, setForm] = useState({
    name: '', slug: '',
    settings: {
      rate_limit: 100, rate_burst: 200, max_destinataires: 500, default_priority: 'default',
      spam_score_threshold: undefined as number | undefined,
      spam_score_action: undefined as 'warn' | 'block' | undefined,
      language: '' as '' | 'fr' | 'en',
      store_body: false,
    },
  });
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const locale = i18n.language === 'fr' ? 'fr-FR' : 'en-US';

  const loadTenants = () => {
    setLoading(true);
    if (isAdmin) {
      api.get('/admin/tenants')
        .then((res) => setTenants(res.data.data?.items || []))
        .catch(() => {})
        .finally(() => setLoading(false));
    } else {
      api.get('/tenant/me')
        .then((res) => setTenants([res.data.data]))
        .catch(() => {})
        .finally(() => setLoading(false));
    }
  };

  useEffect(() => { loadTenants(); }, []);

  const resetForm = () => {
    setForm({ name: '', slug: '', settings: { rate_limit: 100, rate_burst: 200, max_destinataires: 500, default_priority: 'default', spam_score_threshold: undefined, spam_score_action: undefined, language: '', store_body: false } });
    setEditingId(null);
    setShowForm(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFieldErrors({});
    try {
      if (editingId) {
        await api.put(`/admin/tenants/${editingId}`, form);
        addToast(t('tenants.updated'), 'success');
      } else {
        await api.post('/admin/tenants', form);
        addToast(t('tenants.created'), 'success');
      }
      resetForm();
      loadTenants();
    } catch (err) {
      const fe = getFieldErrors(err);
      if (Object.keys(fe).length > 0) {
        setFieldErrors(fe);
      } else {
        addToast(getApiError(err), 'error');
      }
    }
  };

  const handleEdit = (tenant: Tenant) => {
    setForm({
      name: tenant.name, slug: tenant.slug,
      settings: {
        ...tenant.settings,
        spam_score_threshold: tenant.settings.spam_score_threshold ?? undefined,
        spam_score_action: tenant.settings.spam_score_action ?? undefined,
        language: tenant.settings.language ?? '',
        store_body: tenant.settings.store_body ?? false,
      },
    });
    setEditingId(tenant.id);
    setShowForm(true);
  };

  const handleToggleActive = async (tenant: Tenant) => {
    try {
      await api.put(`/admin/tenants/${tenant.id}`, { is_active: !tenant.is_active });
      addToast(tenant.is_active ? t('tenants.deactivated') : t('tenants.activated'), 'success');
      loadTenants();
    } catch (err) {
      addToast(getApiError(err), 'error');
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm(t('tenants.confirm_delete'))) return;
    try {
      await api.delete(`/admin/tenants/${id}`);
      addToast(t('tenants.deleted'), 'success');
      loadTenants();
    } catch (err) {
      addToast(getApiError(err), 'error');
    }
  };

  const handleRegenerateKey = async (tenant: Tenant) => {
    if (!confirm(t('tenants.confirm_regenerate_key'))) return;
    try {
      await api.post(`/admin/tenants/${tenant.id}/regenerate-key`);
      addToast(t('tenants.key_regenerated'), 'success');
      loadTenants();
    } catch (err) {
      addToast(getApiError(err), 'error');
    }
  };

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(id);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  if (loading) return <p className="text-gray-500">{t('common.loading')}</p>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900">{isAdmin ? t('tenants.title') : t('tenants.my_tenant')}</h2>
        {isAdmin && (
          <button
            onClick={() => { resetForm(); setShowForm(true); }}
            className="px-4 py-2 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700 cursor-pointer"
          >
            {t('tenants.new')}
          </button>
        )}
      </div>

      {isAdmin && showForm && (
        <div className="bg-white rounded-xl shadow-sm p-6 mb-6">
          <h3 className="font-semibold text-gray-900 mb-4">
            {editingId ? t('tenants.edit_title') : t('tenants.new_title')}
          </h3>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <FormInput label={t('tenants.name')} value={form.name} onChange={(v) => setForm({ ...form, name: v })} required error={fieldErrors['name']} tooltip={t('tenants.tooltip.name')} />
              <FormInput label={t('tenants.slug')} value={form.slug} onChange={(v) => setForm({ ...form, slug: v })} error={fieldErrors['slug']} placeholder={t('tenants.slug_placeholder')} tooltip={t('tenants.tooltip.slug')} />
            </div>
            <div className="grid grid-cols-4 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {t('tenants.rate_limit')}
                  <FormTooltip text={t('tenants.tooltip.rate_limit')} />
                </label>
                <input
                  type="number"
                  value={form.settings.rate_limit}
                  onChange={(e) => setForm({ ...form, settings: { ...form.settings, rate_limit: Number(e.target.value) } })}
                  className={`w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none ${fieldErrors['rate_limit'] || fieldErrors['settings.rate_limit'] ? 'border-red-500' : 'border-gray-300'}`}
                />
                {(fieldErrors['rate_limit'] || fieldErrors['settings.rate_limit']) && <p className="text-red-500 text-xs mt-1">{fieldErrors['rate_limit'] || fieldErrors['settings.rate_limit']}</p>}
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {t('tenants.rate_burst')}
                  <FormTooltip text={t('tenants.tooltip.rate_burst')} />
                </label>
                <input
                  type="number"
                  value={form.settings.rate_burst}
                  onChange={(e) => setForm({ ...form, settings: { ...form.settings, rate_burst: Number(e.target.value) } })}
                  className={`w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none ${fieldErrors['rate_burst'] || fieldErrors['settings.rate_burst'] ? 'border-red-500' : 'border-gray-300'}`}
                />
                {(fieldErrors['rate_burst'] || fieldErrors['settings.rate_burst']) && <p className="text-red-500 text-xs mt-1">{fieldErrors['rate_burst'] || fieldErrors['settings.rate_burst']}</p>}
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {t('tenants.max_recipients')}
                  <FormTooltip text={t('tenants.tooltip.max_recipients')} />
                </label>
                <input
                  type="number"
                  value={form.settings.max_destinataires}
                  onChange={(e) => setForm({ ...form, settings: { ...form.settings, max_destinataires: Number(e.target.value) } })}
                  className={`w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none ${fieldErrors['max_destinataires'] || fieldErrors['settings.max_destinataires'] ? 'border-red-500' : 'border-gray-300'}`}
                />
                {(fieldErrors['max_destinataires'] || fieldErrors['settings.max_destinataires']) && <p className="text-red-500 text-xs mt-1">{fieldErrors['max_destinataires'] || fieldErrors['settings.max_destinataires']}</p>}
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {t('tenants.default_priority')}
                  <FormTooltip text={t('tenants.tooltip.default_priority')} />
                </label>
                <select
                  value={form.settings.default_priority}
                  onChange={(e) => setForm({ ...form, settings: { ...form.settings, default_priority: e.target.value } })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
                >
                  <option value="low">{t('tenants.priority_low')}</option>
                  <option value="default">{t('tenants.priority_default')}</option>
                  <option value="critical">{t('tenants.priority_critical')}</option>
                </select>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {t('tenants.spam_threshold')}
                  <FormTooltip text={t('tenants.tooltip.spam_threshold')} />
                </label>
                <input
                  type="number"
                  step="0.5"
                  min="0"
                  max="10"
                  value={form.settings.spam_score_threshold ?? ''}
                  onChange={(e) => setForm({ ...form, settings: { ...form.settings, spam_score_threshold: e.target.value ? Number(e.target.value) : undefined } })}
                  placeholder="—"
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {t('tenants.spam_action')}
                  <FormTooltip text={t('tenants.tooltip.spam_action')} />
                </label>
                <select
                  value={form.settings.spam_score_action ?? ''}
                  onChange={(e) => setForm({ ...form, settings: { ...form.settings, spam_score_action: (e.target.value || undefined) as 'warn' | 'block' | undefined } })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
                >
                  <option value="">{t('tenants.spam_action_none')}</option>
                  <option value="warn">{t('tenants.spam_action_warn')}</option>
                  <option value="block">{t('tenants.spam_action_block')}</option>
                </select>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {t('tenants.language')}
                  <FormTooltip text={t('tenants.tooltip.language')} />
                </label>
                <select
                  value={form.settings.language}
                  onChange={(e) => setForm({ ...form, settings: { ...form.settings, language: (e.target.value || '') as '' | 'fr' | 'en' } })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
                >
                  <option value="">{t('tenants.language_default')}</option>
                  <option value="fr">{t('tenants.language_fr')}</option>
                  <option value="en">{t('tenants.language_en')}</option>
                </select>
              </div>
              <div className="flex items-center pt-6">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={form.settings.store_body}
                    onChange={(e) => setForm({ ...form, settings: { ...form.settings, store_body: e.target.checked } })}
                    className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                  />
                  <span className="text-sm font-medium text-gray-700">
                    {t('tenants.store_body')}
                    <FormTooltip text={t('tenants.tooltip.store_body')} />
                  </span>
                </label>
              </div>
            </div>
            <div className="flex gap-2">
              <button type="submit" className="px-4 py-2 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700 cursor-pointer">
                {editingId ? t('common.edit') : t('common.create')}
              </button>
              <button type="button" onClick={resetForm} className="px-4 py-2 border text-sm rounded-lg hover:bg-gray-50 cursor-pointer">
                {t('common.cancel')}
              </button>
            </div>
          </form>
        </div>
      )}

      <div className="grid gap-4">
        {tenants.map((tenant) => (
          <div key={tenant.id} className="bg-white rounded-xl shadow-sm p-5">
            <div className="flex items-start justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="font-semibold text-gray-900">{tenant.name}</h3>
                  <span className="px-2 py-0.5 bg-gray-100 text-gray-500 text-xs rounded-full font-mono">
                    {tenant.slug}
                  </span>
                  <span className={`px-2 py-0.5 text-xs rounded-full ${tenant.is_active ? 'bg-emerald-100 text-emerald-700' : 'bg-red-100 text-red-700'}`}>
                    {tenant.is_active ? t('common.active') : t('common.inactive')}
                  </span>
                </div>
                {tenant.api_key && (
                  <div className="flex items-center gap-2 mt-2">
                    <code className="text-xs bg-gray-50 px-2 py-1 rounded font-mono text-gray-600 select-all">
                      {tenant.api_key}
                    </code>
                    <button
                      onClick={() => copyToClipboard(tenant.api_key!, tenant.id)}
                      className="text-xs text-indigo-600 hover:underline cursor-pointer"
                    >
                      {copiedKey === tenant.id ? t('common.copied') : t('common.copy')}
                    </button>
                    {isAdmin && (
                      <button
                        onClick={() => handleRegenerateKey(tenant)}
                        className="text-xs text-amber-600 hover:underline cursor-pointer"
                      >
                        {t('tenants.regenerate_key')}
                      </button>
                    )}
                  </div>
                )}
                <p className="text-xs text-gray-400 mt-2">
                  {t('tenants.info_line', {
                    rate: tenant.settings.rate_limit,
                    burst: tenant.settings.rate_burst,
                    max: tenant.settings.max_destinataires,
                    priority: tenant.settings.default_priority || 'default',
                  })}
                </p>
                <p className="text-xs text-gray-400 mt-0.5">
                  {t('tenants.created_at', { date: new Date(tenant.created_at).toLocaleDateString(locale) })}
                </p>
              </div>
              {isAdmin && (
                <div className="flex gap-2 text-xs">
                  <button onClick={() => handleToggleActive(tenant)} className={`${tenant.is_active ? 'text-amber-600' : 'text-emerald-600'} hover:underline cursor-pointer`}>
                    {tenant.is_active ? t('tenants.deactivate') : t('tenants.activate')}
                  </button>
                  <button onClick={() => handleEdit(tenant)} className="text-indigo-600 hover:underline cursor-pointer">
                    {t('common.edit')}
                  </button>
                  <button onClick={() => handleDelete(tenant.id)} className="text-red-600 hover:underline cursor-pointer">
                    {t('common.delete')}
                  </button>
                </div>
              )}
            </div>
          </div>
        ))}
        {tenants.length === 0 && (
          <p className="text-center text-gray-400 py-8">{t('tenants.no_tenant')}</p>
        )}
      </div>
    </div>
  );
}

function FormInput({ label, value, onChange, required = false, error, placeholder, tooltip }: { label: string; value: string; onChange: (v: string) => void; required?: boolean; error?: string; placeholder?: string; tooltip?: string }) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">
        {label}{required && <span className="text-red-500 ml-0.5">*</span>}
        {tooltip && <FormTooltip text={tooltip} />}
      </label>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        required={required}
        placeholder={placeholder}
        className={`w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none ${error ? 'border-red-500' : 'border-gray-300'}`}
      />
      {error && <p className="text-red-500 text-xs mt-1">{error}</p>}
    </div>
  );
}
