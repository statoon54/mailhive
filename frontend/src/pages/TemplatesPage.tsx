import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import api, { type Template, type SpamCheckResult, type HTMLCheckResult, type LinkCheckResult } from '../api/client';
import { useToast } from '../contexts/ToastContext';
import { getApiError, getFieldErrors } from '../utils/apiError';
import FormTooltip from '../components/FormTooltip';
import HtmlEditor from '../components/HtmlEditor';
import AiGenerateModal from '../components/AiGenerateModal';

type AnalysisTab = 'spam' | 'html' | 'link';
interface AnalysisState {
  templateId: string;
  tab: AnalysisTab;
  loading: boolean;
  vars: Record<string, string>;
  spam?: SpamCheckResult;
  html?: HTMLCheckResult;
  links?: LinkCheckResult;
  error?: string;
}

export default function TemplatesPage() {
  const { t } = useTranslation();
  const { addToast } = useToast();
  const [templates, setTemplates] = useState<Template[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [previewData, setPreviewData] = useState<{ subject: string; text_body: string; html_body: string } | null>(null);
  const [form, setForm] = useState({
    name: '', slug: '', subject_tmpl: '', text_body: '', html_body: '', variables: {} as Record<string, string>,
  });
  const [newVarKey, setNewVarKey] = useState('');
  const [newVarDesc, setNewVarDesc] = useState('');
  const [previewVars, setPreviewVars] = useState<Record<string, string>>({});
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [showAiModal, setShowAiModal] = useState(false);
  const [aiEnabled, setAiEnabled] = useState(false);
  const [analysis, setAnalysis] = useState<AnalysisState | null>(null);

  useEffect(() => {
    api.get('/ai/status').then((res) => setAiEnabled(res.data.data?.enabled ?? false)).catch(() => {});
  }, []);

  const loadTemplates = () => {
    setLoading(true);
    api.get('/templates')
      .then((res) => setTemplates(res.data.data || []))
      .catch(() => {})
      .finally(() => setLoading(false));
  };

  useEffect(() => { loadTemplates(); }, []);

  const resetForm = () => {
    setForm({ name: '', slug: '', subject_tmpl: '', text_body: '', html_body: '', variables: {} });
    setNewVarKey('');
    setNewVarDesc('');
    setEditingId(null);
    setShowForm(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFieldErrors({});
    try {
      const payload = { ...form, variables: Object.keys(form.variables).length > 0 ? form.variables : undefined };
      if (editingId) {
        await api.put(`/templates/${editingId}`, payload);
        addToast(t('templates.updated'), 'success');
      } else {
        await api.post('/templates', payload);
        addToast(t('templates.created'), 'success');
      }
      resetForm();
      loadTemplates();
    } catch (err) {
      const fe = getFieldErrors(err);
      if (Object.keys(fe).length > 0) {
        setFieldErrors(fe);
      } else {
        addToast(getApiError(err), 'error');
      }
    }
  };

  const handleEdit = (tmpl: Template) => {
    setForm({
      name: tmpl.name, slug: tmpl.slug, subject_tmpl: tmpl.subject_tmpl,
      text_body: tmpl.text_body, html_body: tmpl.html_body,
      variables: tmpl.variables || {},
    });
    setEditingId(tmpl.id);
    setShowForm(true);
  };

  const handleDelete = async (id: string) => {
    if (!confirm(t('templates.confirm_delete'))) return;
    try {
      await api.delete(`/templates/${id}`);
      addToast(t('templates.deleted'), 'success');
      if (analysis?.templateId === id) setAnalysis(null);
      if (previewTemplateId === id) { setPreviewTemplateId(null); setPreviewData(null); }
      loadTemplates();
    } catch (err) {
      addToast(getApiError(err), 'error');
    }
  };

  const [previewTemplateId, setPreviewTemplateId] = useState<string | null>(null);

  const openPreviewForm = (tmpl: Template) => {
    const vars: Record<string, string> = {};
    if (tmpl.variables) {
      for (const key of Object.keys(tmpl.variables)) {
        vars[key] = '';
      }
    }
    setPreviewVars(vars);
    setPreviewTemplateId(tmpl.id);
    setPreviewData(null);
  };

  const handlePreview = async () => {
    if (!previewTemplateId) return;
    try {
      const res = await api.post(`/templates/${previewTemplateId}/preview`, { data: previewVars });
      setPreviewData(res.data.data);
    } catch (err) {
      addToast(getApiError(err), 'error');
    }
  };

  const openAnalysis = (tmpl: Template, tab: AnalysisTab) => {
    const vars: Record<string, string> = {};
    if (tmpl.variables) {
      for (const key of Object.keys(tmpl.variables)) {
        const lower = key.toLowerCase();
        vars[key] = lower.includes('url') || lower.includes('link') || lower.includes('href')
          ? 'https://example.com'
          : 'test';
      }
    }
    setAnalysis({ templateId: tmpl.id, tab, loading: false, vars });
  };

  const runAnalysis = async () => {
    if (!analysis) return;
    setAnalysis({ ...analysis, loading: true, error: undefined });
    const endpoint = analysis.tab === 'spam' ? 'spam-check' : analysis.tab === 'html' ? 'html-check' : 'link-check';
    try {
      const res = await api.post(`/templates/${analysis.templateId}/${endpoint}`, { data: analysis.vars });
      const data = res.data.data;
      setAnalysis((prev) => prev ? {
        ...prev,
        loading: false,
        spam: analysis.tab === 'spam' ? data : prev.spam,
        html: analysis.tab === 'html' ? data : prev.html,
        links: analysis.tab === 'link' ? data : prev.links,
      } : null);
    } catch (err) {
      const msg = getApiError(err);
      setAnalysis((prev) => prev ? { ...prev, loading: false, error: msg } : null);
    }
  };

  if (loading) return <p className="text-gray-500">{t('common.loading')}</p>;

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-gray-900">{t('templates.title')}</h2>
        <button
          onClick={() => { resetForm(); setShowForm(true); }}
          className="px-4 py-2 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700 cursor-pointer"
        >
          {t('templates.new')}
        </button>
      </div>

      {showForm && (
        <div className="bg-white rounded-xl shadow-sm p-6 mb-6">
          <h3 className="font-semibold text-gray-900 mb-4">
            {editingId ? t('templates.edit_title') : t('templates.new_title')}
          </h3>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <Input label={t('templates.name')} value={form.name} onChange={(v) => setForm({ ...form, name: v })} required error={fieldErrors['name']} tooltip={t('templates.tooltip.name')} />
              <Input label={t('templates.slug')} value={form.slug} onChange={(v) => setForm({ ...form, slug: v })} error={fieldErrors['slug']} tooltip={t('templates.tooltip.slug')} />
            </div>
            <Input label={t('templates.subject_tmpl')} value={form.subject_tmpl} onChange={(v) => setForm({ ...form, subject_tmpl: v })} required error={fieldErrors['subject_tmpl']} tooltip={t('templates.tooltip.subject_tmpl')} />
            <TextArea label={t('templates.text_body')} value={form.text_body} onChange={(v) => setForm({ ...form, text_body: v })} required error={fieldErrors['text_body']} tooltip={t('templates.tooltip.text_body')} />
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="block text-sm font-medium text-gray-700">
                  {t('templates.html_body')}<span className="text-red-500 ml-0.5">*</span>
                  <FormTooltip text={t('templates.tooltip.html_body')} />
                </label>
                {aiEnabled && (
                  <button
                    type="button"
                    onClick={() => setShowAiModal(true)}
                    className="px-3 py-1 text-xs bg-violet-100 text-violet-700 rounded-lg hover:bg-violet-200 cursor-pointer font-medium"
                  >
                    ✦ {t('ai.generate')}
                  </button>
                )}
              </div>
              <HtmlEditor value={form.html_body} onChange={(v) => setForm({ ...form, html_body: v })} error={fieldErrors['html_body']} />
            </div>

            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 text-sm text-blue-800">
              <p className="font-medium mb-1">{t('templates.variables_syntax')}</p>
              <p dangerouslySetInnerHTML={{ __html: t('templates.variables_syntax_desc') }} />
              <p className="mt-1" dangerouslySetInnerHTML={{ __html: t('templates.variables_data_desc') }} />
              <p className="mt-1" dangerouslySetInnerHTML={{ __html: t('templates.variables_example') }} />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">{t('templates.expected_variables')}</label>
              <p className="text-xs text-gray-500 mb-2">{t('templates.expected_variables_desc')}</p>
              {Object.entries(form.variables).map(([key, desc]) => (
                <div key={key} className="flex items-center gap-2 mb-2">
                  <code className="text-sm bg-gray-100 px-2 py-1 rounded min-w-30">{`{{.${key}}}`}</code>
                  <span className="text-sm text-gray-600 flex-1">{desc}</span>
                  <button
                    type="button"
                    onClick={() => {
                      const next = { ...form.variables };
                      delete next[key];
                      setForm({ ...form, variables: next });
                    }}
                    className="text-red-500 hover:text-red-700 text-xs cursor-pointer"
                  >
                    {t('templates.remove')}
                  </button>
                </div>
              ))}
              <div className="flex items-center gap-2">
                <input
                  placeholder={t('templates.var_name_placeholder')}
                  value={newVarKey}
                  onChange={(e) => setNewVarKey(e.target.value)}
                  className="px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none w-36"
                />
                <input
                  placeholder={t('templates.var_desc_placeholder')}
                  value={newVarDesc}
                  onChange={(e) => setNewVarDesc(e.target.value)}
                  className="px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none flex-1"
                />
                <button
                  type="button"
                  onClick={() => {
                    if (newVarKey.trim()) {
                      setForm({ ...form, variables: { ...form.variables, [newVarKey.trim()]: newVarDesc.trim() } });
                      setNewVarKey('');
                      setNewVarDesc('');
                    }
                  }}
                  className="px-3 py-1.5 bg-gray-100 text-sm rounded-lg hover:bg-gray-200 cursor-pointer"
                >
                  {t('templates.add')}
                </button>
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

      {previewTemplateId && (
        <div className="bg-white rounded-xl shadow-sm p-6 mb-6">
          <div className="flex justify-between items-center mb-4">
            <h3 className="font-semibold text-gray-900">
              {t('templates.preview_title')} — <span className="text-indigo-600">{templates.find((t) => t.id === previewTemplateId)?.name}</span>
            </h3>
            <button onClick={() => { setPreviewTemplateId(null); setPreviewData(null); }} className="text-gray-400 hover:text-gray-600 cursor-pointer">✕</button>
          </div>
          {Object.keys(previewVars).length > 0 && (
            <div className="mb-4 space-y-2">
              <p className="text-sm font-medium text-gray-700">{t('templates.preview_test_values')}</p>
              {Object.entries(previewVars).map(([key, val]) => (
                <div key={key} className="flex items-center gap-2">
                  <label className="text-sm text-gray-600 min-w-30"><code>{`{{.${key}}}`}</code></label>
                  <input
                    value={val}
                    onChange={(e) => setPreviewVars({ ...previewVars, [key]: e.target.value })}
                    placeholder={t('templates.preview_value_placeholder', { key })}
                    className="px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none flex-1"
                  />
                </div>
              ))}
            </div>
          )}
          <button onClick={handlePreview} className="px-4 py-2 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700 cursor-pointer mb-4">
            {previewData ? t('templates.preview_refresh') : t('templates.preview_generate')}
          </button>
          {previewData && (
            <div className="border border-gray-200 rounded-lg overflow-hidden">
              <div className="bg-gray-50 border-b border-gray-200 px-4 py-3 space-y-1">
                <div className="flex text-sm">
                  <span className="text-gray-500 w-16 shrink-0">{t('templates.preview_subject')}</span>
                  <span className="font-medium text-gray-900">{previewData.subject}</span>
                </div>
              </div>

              {previewData.html_body ? (
                <div className="bg-white">
                  <iframe
                    srcDoc={previewData.html_body}
                    title={t('mail_detail.html_preview')}
                    className="w-full border-0"
                    sandbox=""
                    style={{ minHeight: '300px' }}
                    onLoad={(e) => {
                      const frame = e.target as HTMLIFrameElement;
                      if (frame.contentDocument?.body) {
                        frame.style.height = frame.contentDocument.body.scrollHeight + 20 + 'px';
                      }
                    }}
                  />
                </div>
              ) : previewData.text_body ? (
                <pre className="bg-white p-4 text-sm text-gray-700 whitespace-pre-wrap">{previewData.text_body}</pre>
              ) : (
                <p className="p-4 text-gray-400 text-sm">{t('templates.preview_no_content')}</p>
              )}

              {previewData.html_body && previewData.text_body && (
                <details className="border-t border-gray-200">
                  <summary className="px-4 py-2 text-xs text-gray-500 cursor-pointer hover:bg-gray-50">
                    {t('templates.preview_text_version')}
                  </summary>
                  <pre className="px-4 py-3 text-sm text-gray-700 whitespace-pre-wrap bg-gray-50">{previewData.text_body}</pre>
                </details>
              )}
            </div>
          )}
        </div>
      )}

      {analysis && (
        <div className="bg-white rounded-xl shadow-sm p-6 mb-6">
          <div className="flex justify-between items-center mb-4">
            <div className="flex items-center gap-2">
              <h3 className="font-semibold text-gray-900">
                {t(`templates.${analysis.tab}_check`)} — <span className="text-indigo-600">{templates.find((tp) => tp.id === analysis.templateId)?.name}</span>
              </h3>
              <div className="flex gap-1 ml-4">
                {(['spam', 'html', 'link'] as AnalysisTab[]).map((tab) => (
                  <button
                    key={tab}
                    onClick={() => setAnalysis({ ...analysis, tab })}
                    className={`px-3 py-1 text-xs rounded-lg cursor-pointer ${analysis.tab === tab ? 'bg-indigo-100 text-indigo-700 font-medium' : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}`}
                  >
                    {t(`templates.${tab}_check`)}
                  </button>
                ))}
              </div>
            </div>
            <button onClick={() => setAnalysis(null)} className="text-gray-400 hover:text-gray-600 cursor-pointer">X</button>
          </div>

          {Object.keys(analysis.vars).length > 0 && (
            <div className="mb-4 space-y-2">
              <p className="text-sm font-medium text-gray-700">{t('templates.preview_test_values')}</p>
              {Object.entries(analysis.vars).map(([key, val]) => (
                <div key={key} className="flex items-center gap-2">
                  <label className="text-sm text-gray-600 min-w-30"><code>{`{{.${key}}}`}</code></label>
                  <input
                    value={val}
                    onChange={(e) => setAnalysis({ ...analysis, vars: { ...analysis.vars, [key]: e.target.value } })}
                    placeholder={t('templates.preview_value_placeholder', { key })}
                    className="px-3 py-1.5 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none flex-1"
                  />
                </div>
              ))}
            </div>
          )}

          <button
            onClick={runAnalysis}
            disabled={analysis.loading}
            className="px-4 py-2 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700 cursor-pointer disabled:opacity-50 mb-4"
          >
            {analysis.loading ? t('templates.analysis_loading') : t(`templates.${analysis.tab}_check`)}
          </button>

          {analysis.error && (
            <div className="bg-red-50 text-red-700 p-3 rounded-lg text-sm mb-4">{analysis.error}</div>
          )}

          {analysis.tab === 'spam' && analysis.spam && (
            <div className="border border-gray-200 rounded-lg p-4 space-y-3">
              <div className="flex items-center gap-4">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-gray-700">{t('templates.spam_score')} :</span>
                  <SpamScoreBadge score={analysis.spam.score} />
                  <span className="text-xs text-gray-400">/ {analysis.spam.max_score}</span>
                </div>
                <span className={`px-2 py-0.5 text-xs rounded-full font-medium ${analysis.spam.pass ? 'bg-emerald-100 text-emerald-700' : 'bg-red-100 text-red-700'}`}>
                  {analysis.spam.pass ? t('templates.spam_pass') : t('templates.spam_fail')}
                </span>
              </div>
              {analysis.spam.rules.length > 0 && (
                <div>
                  <p className="text-sm font-medium text-gray-700 mb-2">{t('templates.spam_rules_triggered')}</p>
                  <div className="space-y-1">
                    {analysis.spam.rules.map((rule) => (
                      <div key={rule.name} className="flex items-center gap-2 text-sm">
                        <span className="px-1.5 py-0.5 bg-red-50 text-red-600 text-xs rounded font-mono">+{rule.score.toFixed(1)}</span>
                        <span className="text-gray-700">{rule.description}</span>
                        {rule.details && <span className="text-gray-400 text-xs">({rule.details})</span>}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {analysis.tab === 'html' && analysis.html && (
            <div className="border border-gray-200 rounded-lg p-4">
              {analysis.html.total_count === 0 ? (
                <p className="text-sm text-emerald-600">{t('templates.html_no_issue')}</p>
              ) : (
                <div>
                  <p className="text-sm font-medium text-gray-700 mb-2">{t('templates.html_issues')} ({analysis.html.total_count})</p>
                  <table className="w-full text-sm">
                    <thead className="bg-gray-50 text-gray-600">
                      <tr>
                        <th className="text-left px-3 py-2 font-medium">Property</th>
                        <th className="text-left px-3 py-2 font-medium">Description</th>
                        <th className="text-left px-3 py-2 font-medium">{t('templates.html_severity')}</th>
                        <th className="text-left px-3 py-2 font-medium">{t('templates.html_clients')}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {analysis.html.issues.map((issue, i) => (
                        <tr key={i}>
                          <td className="px-3 py-2 font-mono text-xs">{issue.property}</td>
                          <td className="px-3 py-2">{issue.description}</td>
                          <td className="px-3 py-2">
                            <span className={`px-1.5 py-0.5 text-xs rounded ${issue.severity === 'error' ? 'bg-red-100 text-red-700' : 'bg-amber-100 text-amber-700'}`}>
                              {issue.severity}
                            </span>
                          </td>
                          <td className="px-3 py-2 text-xs text-gray-500">{issue.clients.join(', ')}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {analysis.tab === 'link' && analysis.links && (
            <div className="border border-gray-200 rounded-lg p-4">
              {analysis.links.total_count === 0 ? (
                <p className="text-sm text-gray-500">{t('templates.link_no_link')}</p>
              ) : (
                <div>
                  <div className="flex gap-4 mb-3">
                    <span className="text-sm text-gray-600">{t('templates.link_total', { count: analysis.links.total_count })}</span>
                    {analysis.links.broken_count > 0 && (
                      <span className="text-sm text-red-600">{t('templates.link_broken', { count: analysis.links.broken_count })}</span>
                    )}
                  </div>
                  <table className="w-full text-sm">
                    <thead className="bg-gray-50 text-gray-600">
                      <tr>
                        <th className="text-left px-3 py-2 font-medium">URL</th>
                        <th className="text-left px-3 py-2 font-medium">Source</th>
                        <th className="text-left px-3 py-2 font-medium">{t('mails.table.status')}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-100">
                      {analysis.links.links.map((link, i) => (
                        <tr key={i}>
                          <td className="px-3 py-2 font-mono text-xs break-all max-w-md">{link.url}</td>
                          <td className="px-3 py-2 text-xs text-gray-500">{link.source}</td>
                          <td className="px-3 py-2">
                            <LinkStatusBadge status={link.status} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      <div className="bg-white rounded-xl shadow-sm overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-gray-50 text-gray-600">
            <tr>
              <th className="text-left px-4 py-3 font-medium">{t('templates.table.id')}</th>
              <th className="text-left px-4 py-3 font-medium">{t('templates.table.name')}</th>
              <th className="text-left px-4 py-3 font-medium">{t('templates.table.slug')}</th>
              <th className="text-left px-4 py-3 font-medium">{t('templates.table.subject')}</th>
              <th className="text-left px-4 py-3 font-medium">{t('templates.table.variables')}</th>
              <th className="text-left px-4 py-3 font-medium">{t('templates.table.actions')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {templates.map((tmpl) => (
              <tr key={tmpl.id} className="hover:bg-gray-50">
                <td className="px-4 py-3">
                  <button
                    onClick={() => { navigator.clipboard.writeText(tmpl.id); addToast(t('templates.id_copied'), 'success'); }}
                    className="font-mono text-xs bg-gray-100 px-2 py-1 rounded hover:bg-gray-200 cursor-pointer"
                    title={t('common.clickToCopy')}
                  >
                    {tmpl.id.slice(0, 8)}...
                  </button>
                </td>
                <td className="px-4 py-3 font-medium">{tmpl.name}</td>
                <td className="px-4 py-3 text-gray-500">{tmpl.slug}</td>
                <td className="px-4 py-3 text-gray-500">{tmpl.subject_tmpl}</td>
                <td className="px-4 py-3 text-gray-500">
                  {tmpl.variables && Object.keys(tmpl.variables).length > 0
                    ? Object.keys(tmpl.variables).map((k) => <code key={k} className="inline-block bg-gray-100 text-xs px-1.5 py-0.5 rounded mr-1 mb-1">{k}</code>)
                    : <span className="text-gray-300">—</span>}
                </td>
                <td className="px-4 py-3 flex gap-2 flex-wrap">
                  <button onClick={() => openPreviewForm(tmpl)} className="text-indigo-600 hover:underline text-xs cursor-pointer">
                    {t('templates.preview')}
                  </button>
                  <button onClick={() => openAnalysis(tmpl, 'spam')} className="text-violet-600 hover:underline text-xs cursor-pointer">
                    {t('templates.spam_check')}
                  </button>
                  <button onClick={() => openAnalysis(tmpl, 'html')} className="text-violet-600 hover:underline text-xs cursor-pointer">
                    {t('templates.html_check')}
                  </button>
                  <button onClick={() => openAnalysis(tmpl, 'link')} className="text-violet-600 hover:underline text-xs cursor-pointer">
                    {t('templates.link_check')}
                  </button>
                  <button onClick={() => handleEdit(tmpl)} className="text-amber-600 hover:underline text-xs cursor-pointer">
                    {t('common.edit')}
                  </button>
                  <button onClick={() => handleDelete(tmpl.id)} className="text-red-600 hover:underline text-xs cursor-pointer">
                    {t('common.delete')}
                  </button>
                </td>
              </tr>
            ))}
            {templates.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-8 text-center text-gray-400">{t('templates.no_template')}</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {showAiModal && (
        <AiGenerateModal
          onInsert={(htmlBody, textBody) => {
            setForm((f) => ({
              ...f,
              html_body: htmlBody,
              ...(textBody ? { text_body: textBody } : {}),
            }));
            setShowAiModal(false);
          }}
          onClose={() => setShowAiModal(false)}
        />
      )}
    </div>
  );
}

function Input({ label, value, onChange, required, error, tooltip }: { label: string; value: string; onChange: (v: string) => void; required?: boolean; error?: string; tooltip?: string }) {
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
        className={`w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none ${error ? 'border-red-500' : 'border-gray-300'}`}
      />
      {error && <p className="text-red-500 text-xs mt-1">{error}</p>}
    </div>
  );
}

function SpamScoreBadge({ score }: { score: number }) {
  const color = score <= 3 ? 'bg-emerald-100 text-emerald-700' : score <= 6 ? 'bg-amber-100 text-amber-700' : 'bg-red-100 text-red-700';
  return <span className={`px-2 py-0.5 text-sm rounded font-bold ${color}`}>{score.toFixed(1)}</span>;
}

function LinkStatusBadge({ status }: { status: string }) {
  const styles: Record<string, string> = {
    ok: 'bg-emerald-100 text-emerald-700',
    broken: 'bg-red-100 text-red-700',
    redirect: 'bg-amber-100 text-amber-700',
    insecure: 'bg-orange-100 text-orange-700',
    timeout: 'bg-gray-100 text-gray-700',
    invalid: 'bg-red-50 text-red-600',
  };
  return <span className={`px-2 py-0.5 text-xs rounded font-medium ${styles[status] || 'bg-gray-100'}`}>{status}</span>;
}

function TextArea({ label, value, onChange, required, error, tooltip }: { label: string; value: string; onChange: (v: string) => void; required?: boolean; error?: string; tooltip?: string }) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">
        {label}{required && <span className="text-red-500 ml-0.5">*</span>}
        {tooltip && <FormTooltip text={tooltip} />}
      </label>
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        rows={4}
        required={required}
        className={`w-full px-3 py-2 border rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none ${error ? 'border-red-500' : 'border-gray-300'}`}
      />
      {error && <p className="text-red-500 text-xs mt-1">{error}</p>}
    </div>
  );
}
