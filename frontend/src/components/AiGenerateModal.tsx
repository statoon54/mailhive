import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import api from '../api/client';
import { getApiError } from '../utils/apiError';

interface AiGenerateModalProps {
  onInsert: (htmlBody: string, textBody: string) => void;
  onClose: () => void;
}

export default function AiGenerateModal({ onInsert, onClose }: AiGenerateModalProps) {
  const { t, i18n } = useTranslation();
  const [prompt, setPrompt] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<{ html_body: string; text_body: string } | null>(null);

  const handleGenerate = async () => {
    if (!prompt.trim()) return;
    setLoading(true);
    setError('');
    setResult(null);

    try {
      const res = await api.post('/ai/generate', {
        prompt: prompt.trim(),
        language: i18n.language,
      });
      setResult(res.data.data);
    } catch (err) {
      setError(getApiError(err, t('ai.error')));
    } finally {
      setLoading(false);
    }
  };

  const handleInsertBoth = () => {
    if (result) {
      onInsert(result.html_body, result.text_body);
    }
  };

  const handleInsertHtml = () => {
    if (result) {
      onInsert(result.html_body, '');
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="bg-white rounded-2xl shadow-2xl w-full max-w-2xl max-h-[90vh] flex flex-col">
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900">{t('ai.title')}</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 text-xl cursor-pointer">&times;</button>
        </div>

        <div className="px-6 py-4 space-y-4 overflow-y-auto flex-1">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              {t('ai.prompt_label')}
            </label>
            <textarea
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder={t('ai.prompt_placeholder')}
              rows={4}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-indigo-500 outline-none"
              disabled={loading}
            />
          </div>

          <button
            type="button"
            onClick={handleGenerate}
            disabled={loading || !prompt.trim()}
            className="px-4 py-2 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700 disabled:opacity-50 cursor-pointer flex items-center gap-2"
          >
            {loading && (
              <span className="inline-block w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            )}
            {loading ? t('ai.generating') : (result ? t('ai.retry') : t('ai.generate'))}
          </button>

          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
              {error}
            </div>
          )}

          {result && (
            <div>
              <p className="text-sm font-medium text-gray-700 mb-2">{t('ai.preview')}</p>
              <div className="border border-gray-200 rounded-lg overflow-hidden">
                <iframe
                  srcDoc={result.html_body}
                  title="AI Preview"
                  className="w-full border-0"
                  sandbox=""
                  style={{ minHeight: '200px' }}
                  onLoad={(e) => {
                    const frame = e.target as HTMLIFrameElement;
                    if (frame.contentDocument?.body) {
                      frame.style.height = Math.min(frame.contentDocument.body.scrollHeight + 20, 400) + 'px';
                    }
                  }}
                />
              </div>
            </div>
          )}
        </div>

        {result && (
          <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-gray-200">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border text-sm rounded-lg hover:bg-gray-50 cursor-pointer"
            >
              {t('ai.cancel')}
            </button>
            <button
              type="button"
              onClick={handleInsertHtml}
              className="px-4 py-2 bg-gray-100 text-gray-700 text-sm rounded-lg hover:bg-gray-200 cursor-pointer"
            >
              {t('ai.insert_html_only')}
            </button>
            <button
              type="button"
              onClick={handleInsertBoth}
              className="px-4 py-2 bg-indigo-600 text-white text-sm rounded-lg hover:bg-indigo-700 cursor-pointer"
            >
              {t('ai.insert_both')}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
