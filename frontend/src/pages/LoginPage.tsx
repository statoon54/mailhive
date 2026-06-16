import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../contexts/AuthContext';
import { useBranding } from '../contexts/BrandingContext';

export default function LoginPage() {
  const { t, i18n } = useTranslation();
  const [apiKey, setApiKey] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const { branding } = useBranding();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await login(apiKey);
      navigate('/');
    } catch {
      setError(t('login.error'));
    } finally {
      setLoading(false);
    }
  };

  const toggleLang = () => {
    i18n.changeLanguage(i18n.language === 'fr' ? 'en' : 'fr');
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="w-full max-w-md">
        <div className="bg-white rounded-2xl shadow-lg p-8">
          <div className="text-center mb-8">
            {branding.logo_url && (
              <img src={branding.logo_url} alt="Logo" className="h-16 mx-auto mb-4 object-contain" />
            )}
            <h1 className="text-2xl font-bold text-gray-900">{branding.app_title}</h1>
            <p className="text-gray-500 mt-2">{t('login.subtitle')}</p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-6">
            <div>
              <label htmlFor="apiKey" className="block text-sm font-medium text-gray-700 mb-2">
                {t('login.api_key')}
              </label>
              <input
                id="apiKey"
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder={t('login.api_key_placeholder')}
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none transition-colors"
                required
              />
            </div>

            {error && (
              <p className="text-red-600 text-sm bg-red-50 p-3 rounded-lg">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full py-3 bg-indigo-600 text-white font-medium rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors cursor-pointer"
            >
              {loading ? t('login.submitting') : t('login.submit')}
            </button>
          </form>

          <div className="mt-4 text-center">
            <button onClick={toggleLang} className="text-xs text-gray-400 hover:text-gray-600 cursor-pointer">
              {i18n.language === 'fr' ? '🇬🇧 English' : '🇫🇷 Français'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
