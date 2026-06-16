import { useState } from 'react';
import { useTranslation } from 'react-i18next';

interface HtmlEditorProps {
  value: string;
  onChange: (html: string) => void;
  error?: string;
}

export default function HtmlEditor({ value, onChange, error }: HtmlEditorProps) {
  const { t } = useTranslation();
  const [showPreview, setShowPreview] = useState(false);

  return (
    <div>
      <div className="flex items-center gap-2 mb-1">
        <button
          type="button"
          onClick={() => setShowPreview(false)}
          className={`text-xs cursor-pointer ${!showPreview ? 'text-indigo-600 font-medium' : 'text-gray-500 hover:text-indigo-600'}`}
        >
          {t('editor.source')}
        </button>
        <span className="text-gray-300">|</span>
        <button
          type="button"
          onClick={() => setShowPreview(true)}
          className={`text-xs cursor-pointer ${showPreview ? 'text-indigo-600 font-medium' : 'text-gray-500 hover:text-indigo-600'}`}
        >
          {t('editor.preview')}
        </button>
      </div>

      {showPreview ? (
        <div className={`border rounded-lg overflow-hidden ${error ? 'border-red-500' : 'border-gray-300'}`}>
          {value ? (
            <iframe
              srcDoc={value}
              title="HTML Preview"
              className="w-full border-0"
              sandbox=""
              style={{ minHeight: '250px' }}
              onLoad={(e) => {
                const frame = e.target as HTMLIFrameElement;
                if (frame.contentDocument?.body) {
                  frame.style.height = Math.min(frame.contentDocument.body.scrollHeight + 20, 500) + 'px';
                }
              }}
            />
          ) : (
            <div className="px-4 py-8 text-center text-gray-400 text-sm">{t('editor.empty_preview')}</div>
          )}
        </div>
      ) : (
        <textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          rows={12}
          className={`w-full px-3 py-2 border rounded-lg text-sm font-mono focus:ring-2 focus:ring-indigo-500 outline-none ${error ? 'border-red-500' : 'border-gray-300'}`}
        />
      )}
      {error && <p className="text-red-500 text-xs mt-1">{error}</p>}
    </div>
  );
}
