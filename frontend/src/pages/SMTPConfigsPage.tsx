import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import api, { type SMTPConfig } from '../api/client';
import { useToast } from '../contexts/ToastContext';
import { getApiError, getFieldErrors } from '../utils/apiError';
import FormTooltip from '../components/FormTooltip';

export default function SMTPConfigsPage() {
  const { t } = useTranslation();
  const { addToast } = useToast();
  const [configs, setConfigs] = useState<SMTPConfig[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<{ id: string; ok: boolean; msg: string } | null>(null);
  const [form, setForm] = useState({
    name: '', host: '', port: 587, username: '', password: '',
    auth_method: 'PLAIN', tls_policy: 'opportunistic',
    from_email: '', from_name: '', is_default: false,
    charset: 'UTF-8', encoding: 'quoted-printable',
  });
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  const loadConfigs = () => {
    setLoading(true);
    api.get('/smtp-configs')
      .then((res) => setConfigs(res.data.data || []))
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  useEffect(() => { loadConfigs(); }, []);

  const resetForm = () => {
    setForm({
      name: '', host: '', port: 587, username: '', password: '',
      auth_method: 'PLAIN', tls_policy: 'opportunistic',
      from_email: '', from_name: '', is_default: false,
      charset: 'UTF-8', encoding: 'quoted-printable',
    });
    setEditingId(null);
    setShowForm(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFieldErrors({});
    try {
      if (editingId) {
        await api.put(`/smtp-configs/${editingId}`, form);
        addToast(t('smtp.updated'), 'success');
      } else {
        await api.post('/smtp-configs', form);
        addToast(t('smtp.created'), 'success');
      }
      resetForm();
      loadConfigs();
    } catch (err) {
      const fe = getFieldErrors(err);
      if (Object.keys(fe).length > 0) {
        setFieldErrors(fe);
      } else {
        addToast(getApiError(err), 'error');
      }
    }
  };

  const handleEdit = (cfg: SMTPConfig) => {
    setForm({
      name: cfg.name, host: cfg.host, port: cfg.port,
      username: cfg.username || '', password: '',
      auth_method: cfg.auth_method, tls_policy: cfg.tls_policy,
      from_email: cfg.from_email, from_name: cfg.from_name,
      is_default: cfg.is_default,
      charset: cfg.charset || 'UTF-8', encoding: cfg.encoding || 'quoted-printable',
    });
    setEditingId(cfg.id);
    setShowForm(true);
  };

  const handleDelete = async (id: string) => {
    if (!confirm(t('smtp.confirm_delete'))) return;
    try {
      await api.delete(`/smtp-configs/${id}`);
      addToast(t('smtp.deleted'), 'success');
      loadConfigs();
    } catch (err) {
      addToast(getApiError(err), 'error');
    }
  };

  const handleTest = async (id: string) => {
    setTestResult(null);
    try {
      await api.post(`/smtp-configs/${id}/test`);
      setTestResult({ id, ok: true, msg: t('smtp.test_success') });
    } catch {
      setTestResult({ id, ok: false, msg: t('smtp.test_failed') });
    }
  };

  if (loading) return <p className="text-gray-500">{t('common.loading')}</p>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900">{t('smtp.title')}</h2>
        <button
          onClick={() => { resetForm(); setShowForm(true); }}
          className="px-4 py-2 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700 cursor-pointer"
        >
          {t('smtp.new')}
        </button>
      </div>

      {showForm && (
        <div className="bg-white rounded-xl shadow-sm p-6 mb-6">
          <h3 className="font-semibold text-gray-900 mb-4">
            {editingId ? t('smtp.edit_title') : t('smtp.new_title')}
          </h3>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <FormInput label={t('smtp.name')} value={form.name} onChange={(v) => setForm({ ...form, name: v })} required error={fieldErrors['name']} tooltip={t('smtp.tooltip.name')} />
              <FormInput label={t('smtp.host')} value={form.host} onChange={(v) => setForm({ ...form, host: v })} required error={fieldErrors['host']} tooltip={t('smtp.tooltip.host')} />
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {t('smtp.port')}
                  <FormTooltip text={t('smtp.tooltip.port')} />
                </label>
                <input
                  type="number"
                  value={form.port}
                  onChange={(e) => setForm({ ...form, port: Number(e.target.value) })}
                  className={`w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none ${fieldErrors['port'] ? 'border-red-500' : 'border-gray-300'}`}
                />
                {fieldErrors['port'] && <p className="text-red-500 text-xs mt-1">{fieldErrors['port']}</p>}
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {t('smtp.auth_method')}
                  <FormTooltip text={t('smtp.tooltip.auth_method')} />
                </label>
                <select
                  value={form.auth_method}
                  onChange={(e) => setForm({ ...form, auth_method: e.target.value })}
                  className={`w-full px-3 py-2 border rounded-lg text-sm ${fieldErrors['auth_method'] ? 'border-red-500' : 'border-gray-300'}`}
                >
                  <option value="PLAIN">PLAIN</option>
                  <option value="LOGIN">LOGIN</option>
                  <option value="CRAM-MD5">CRAM-MD5</option>
                  <option value="NONE">{t('common.none')}</option>
                </select>
                {fieldErrors['auth_method'] && <p className="text-red-500 text-xs mt-1">{fieldErrors['auth_method']}</p>}
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {t('smtp.tls_policy')}
                  <FormTooltip text={t('smtp.tooltip.tls_policy')} />
                </label>
                <select
                  value={form.tls_policy}
                  onChange={(e) => setForm({ ...form, tls_policy: e.target.value })}
                  className={`w-full px-3 py-2 border rounded-lg text-sm ${fieldErrors['tls_policy'] ? 'border-red-500' : 'border-gray-300'}`}
                >
                  <option value="mandatory">{t('smtp.tls_mandatory')}</option>
                  <option value="opportunistic">{t('smtp.tls_opportunistic')}</option>
                  <option value="none">{t('common.none')}</option>
                </select>
                {fieldErrors['tls_policy'] && <p className="text-red-500 text-xs mt-1">{fieldErrors['tls_policy']}</p>}
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <FormInput label={t('smtp.username')} value={form.username} onChange={(v) => setForm({ ...form, username: v })} error={fieldErrors['username']} tooltip={t('smtp.tooltip.username')} />
              <FormInput label={t('smtp.password')} value={form.password} onChange={(v) => setForm({ ...form, password: v })} type="password" error={fieldErrors['password']} tooltip={t('smtp.tooltip.password')} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <FormInput label={t('smtp.from_email')} value={form.from_email} onChange={(v) => setForm({ ...form, from_email: v })} required error={fieldErrors['from_email']} tooltip={t('smtp.tooltip.from_email')} />
              <FormInput label={t('smtp.from_name')} value={form.from_name} onChange={(v) => setForm({ ...form, from_name: v })} error={fieldErrors['from_name']} tooltip={t('smtp.tooltip.from_name')} />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {t('smtp.charset')}
                  <FormTooltip text={t('smtp.tooltip.charset')} />
                </label>
                <select
                  value={form.charset}
                  onChange={(e) => setForm({ ...form, charset: e.target.value })}
                  className={`w-full px-3 py-2 border rounded-lg text-sm ${fieldErrors['charset'] ? 'border-red-500' : 'border-gray-300'}`}
                >
                  <option value="UTF-8">UTF-8</option>
                  <option value="US-ASCII">US-ASCII</option>
                  <option value="ISO-8859-1">ISO-8859-1</option>
                  <option value="ISO-8859-15">ISO-8859-15</option>
                </select>
                {fieldErrors['charset'] && <p className="text-red-500 text-xs mt-1">{fieldErrors['charset']}</p>}
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {t('smtp.encoding')}
                  <FormTooltip text={t('smtp.tooltip.encoding')} />
                </label>
                <select
                  value={form.encoding}
                  onChange={(e) => setForm({ ...form, encoding: e.target.value })}
                  className={`w-full px-3 py-2 border rounded-lg text-sm ${fieldErrors['encoding'] ? 'border-red-500' : 'border-gray-300'}`}
                >
                  <option value="quoted-printable">Quoted-Printable</option>
                  <option value="base64">Base64</option>
                  <option value="7bit">7bit</option>
                  <option value="8bit">8bit</option>
                </select>
                {fieldErrors['encoding'] && <p className="text-red-500 text-xs mt-1">{fieldErrors['encoding']}</p>}
              </div>
            </div>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.is_default}
                onChange={(e) => setForm({ ...form, is_default: e.target.checked })}
              />
              {t('smtp.is_default')}
            </label>
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
        {configs.map((cfg) => (
          <div key={cfg.id} className="bg-white rounded-xl shadow-sm p-5">
            <div className="flex items-start justify-between">
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="font-semibold text-gray-900">{cfg.name}</h3>
                  {cfg.is_default && (
                    <span className="px-2 py-0.5 bg-indigo-100 text-indigo-700 text-xs rounded-full">{t('common.default')}</span>
                  )}
                  <span className={`px-2 py-0.5 text-xs rounded-full ${cfg.is_active ? 'bg-emerald-100 text-emerald-700' : 'bg-gray-100 text-gray-500'}`}>
                    {cfg.is_active ? t('common.active') : t('common.inactive')}
                  </span>
                </div>
                <p className="text-sm text-gray-500 mt-1">
                  {cfg.host}:{cfg.port} — {cfg.from_name} &lt;{cfg.from_email}&gt;
                </p>
                <p className="text-xs text-gray-400 mt-1">
                  {t('smtp.auth_info')} : {cfg.auth_method} | {t('smtp.tls_info')} : {cfg.tls_policy} | {cfg.charset} / {cfg.encoding}
                </p>
                <button
                  onClick={() => { navigator.clipboard.writeText(cfg.id); addToast(t('smtp.id_copied'), 'success'); }}
                  className="mt-1 font-mono text-xs bg-gray-100 px-2 py-1 rounded hover:bg-gray-200 cursor-pointer"
                  title={t('common.clickToCopy')}
                >
                  {cfg.id.slice(0, 8)}...
                </button>
              </div>
              <div className="flex gap-2 text-xs">
                <button onClick={() => handleTest(cfg.id)} className="text-emerald-600 hover:underline cursor-pointer">{t('smtp.test')}</button>
                <button onClick={() => handleEdit(cfg)} className="text-amber-600 hover:underline cursor-pointer">{t('common.edit')}</button>
                <button onClick={() => handleDelete(cfg.id)} className="text-red-600 hover:underline cursor-pointer">{t('common.delete')}</button>
              </div>
            </div>
            {testResult && testResult.id === cfg.id && (
              <p className={`text-sm mt-2 ${testResult.ok ? 'text-emerald-600' : 'text-red-600'}`}>
                {testResult.msg}
              </p>
            )}
          </div>
        ))}
        {configs.length === 0 && (
          <p className="text-center text-gray-400 py-8">{t('smtp.no_config')}</p>
        )}
      </div>
    </div>
  );
}

function FormInput({ label, value, onChange, type = 'text', required, error, tooltip }: { label: string; value: string; onChange: (v: string) => void; type?: string; required?: boolean; error?: string; tooltip?: string }) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">
        {label}{required && <span className="text-red-500 ml-0.5">*</span>}
        {tooltip && <FormTooltip text={tooltip} />}
      </label>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        required={required}
        className={`w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none ${error ? 'border-red-500' : 'border-gray-300'}`}
      />
      {error && <p className="text-red-500 text-xs mt-1">{error}</p>}
    </div>
  );
}
