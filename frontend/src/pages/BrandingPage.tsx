import { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import api, { type APIResponse, type AppBranding } from '../api/client';
import { useBranding } from '../contexts/BrandingContext';
import { useToast } from '../contexts/ToastContext';
import { getApiError } from '../utils/apiError';

const COMMON_TIMEZONES = [
  'Europe/Paris', 'Europe/London', 'Europe/Berlin', 'Europe/Brussels',
  'Europe/Madrid', 'Europe/Rome', 'Europe/Zurich', 'Europe/Amsterdam',
  'Europe/Moscow', 'America/New_York', 'America/Chicago', 'America/Denver',
  'America/Los_Angeles', 'America/Toronto', 'America/Sao_Paulo',
  'America/Mexico_City', 'Asia/Tokyo', 'Asia/Shanghai', 'Asia/Kolkata',
  'Asia/Dubai', 'Asia/Singapore', 'Australia/Sydney', 'Pacific/Auckland',
  'Africa/Casablanca', 'Africa/Lagos', 'UTC',
];

function getTimezoneLabel(tz: string, locale: string): string {
  try {
    const now = new Date();
    const formatter = new Intl.DateTimeFormat(locale, { timeZone: tz, timeZoneName: 'shortOffset' });
    const parts = formatter.formatToParts(now);
    const offset = parts.find(p => p.type === 'timeZoneName')?.value || '';
    return `${tz.replace(/_/g, ' ')} (${offset})`;
  } catch {
    return tz;
  }
}

export default function BrandingPage() {
  const { t, i18n } = useTranslation();
  const { branding, refreshBranding } = useBranding();
  const { addToast } = useToast();
  const [title, setTitle] = useState('');
  const [subtitle, setSubtitle] = useState('');
  const [timezone, setTimezone] = useState('Europe/Paris');
  const [logoFile, setLogoFile] = useState<File | null>(null);
  const [logoPreview, setLogoPreview] = useState<string>('');
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);

  const locale = i18n.language === 'fr' ? 'fr-FR' : 'en-US';

  useEffect(() => {
    setTitle(branding.app_title);
    setSubtitle(branding.app_subtitle);
    setTimezone(branding.timezone || 'Europe/Paris');
    setLogoPreview(branding.logo_url ? branding.logo_url + '?t=' + Date.now() : '');
  }, [branding]);

  const sortedTimezones = useMemo(() =>
    [...COMMON_TIMEZONES].sort((a, b) => getTimezoneLabel(a, locale).localeCompare(getTimezoneLabel(b, locale))),
  [locale]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);

    try {
      await api.put<APIResponse<AppBranding>>('/admin/branding', {
        app_title: title,
        app_subtitle: subtitle,
        timezone: timezone,
      });
      await refreshBranding();
      addToast(t('branding.saved'), 'success');
    } catch (err) {
      addToast(getApiError(err, t('branding.save_error')), 'error');
    } finally {
      setSaving(false);
    }
  };

  const handleLogoChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setLogoFile(file);
      setLogoPreview(URL.createObjectURL(file));
    }
  };

  const handleLogoUpload = async () => {
    if (!logoFile) return;
    setUploading(true);

    try {
      const formData = new FormData();
      formData.append('logo', logoFile);
      await api.post('/admin/branding/logo', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
      setLogoFile(null);
      await refreshBranding();
      addToast(t('branding.logo_saved'), 'success');
    } catch (err) {
      addToast(getApiError(err, t('branding.logo_error')), 'error');
    } finally {
      setUploading(false);
    }
  };

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">{t('branding.title')}</h1>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('branding.title_subtitle')}</h2>
          {/* Sélecteur de langue */}
          <div className="mb-6 pb-6 border-b border-gray-200">
            <label htmlFor="language" className="block text-sm font-medium text-gray-700 mb-1">
              {t('branding.language')}
            </label>
            <select
              id="language"
              value={i18n.language.startsWith('fr') ? 'fr' : 'en'}
              onChange={(e) => i18n.changeLanguage(e.target.value)}
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none bg-white"
            >
              <option value="fr">{t('language.fr')}</option>
              <option value="en">{t('language.en')}</option>
            </select>
            <p className="mt-1 text-xs text-gray-500">
              {t('branding.language_hint')}
            </p>
          </div>
          <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label htmlFor="title" className="block text-sm font-medium text-gray-700 mb-1">
                {t('branding.app_title')}
              </label>
              <input
                id="title"
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none"
                maxLength={255}
                required
              />
            </div>
            <div>
              <label htmlFor="subtitle" className="block text-sm font-medium text-gray-700 mb-1">
                {t('branding.app_subtitle')}
              </label>
              <input
                id="subtitle"
                type="text"
                value={subtitle}
                onChange={(e) => setSubtitle(e.target.value)}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none"
                maxLength={255}
              />
            </div>
            <div>
              <label htmlFor="timezone" className="block text-sm font-medium text-gray-700 mb-1">
                {t('branding.timezone')}
              </label>
              <select
                id="timezone"
                value={timezone}
                onChange={(e) => setTimezone(e.target.value)}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none bg-white"
              >
                {sortedTimezones.map((tz) => (
                  <option key={tz} value={tz}>
                    {getTimezoneLabel(tz, locale)}
                  </option>
                ))}
              </select>
              <p className="mt-1 text-xs text-gray-500">
                {t('branding.timezone_hint')}
              </p>
            </div>
            <button
              type="submit"
              disabled={saving}
              className="px-6 py-2 bg-indigo-600 text-white font-medium rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors cursor-pointer"
            >
              {saving ? t('common.saving') : t('common.save')}
            </button>
          </form>
        </div>

        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">{t('branding.logo_title')}</h2>
          <div className="space-y-4">
            {logoPreview && (
              <div className="flex items-center justify-center p-4 bg-gray-50 rounded-lg">
                <img src={logoPreview} alt={t('branding.logo_preview')} className="max-h-24 max-w-full object-contain" />
              </div>
            )}
            <div>
              <label htmlFor="logo" className="block text-sm font-medium text-gray-700 mb-1">
                {t('branding.logo_label')}
              </label>
              <input
                id="logo"
                type="file"
                accept="image/png,image/jpeg,image/svg+xml"
                onChange={handleLogoChange}
                className="w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded-lg file:border-0 file:text-sm file:font-medium file:bg-indigo-50 file:text-indigo-700 hover:file:bg-indigo-100 file:cursor-pointer"
              />
            </div>
            {logoFile && (
              <button
                type="button"
                onClick={handleLogoUpload}
                disabled={uploading}
                className="px-6 py-2 bg-indigo-600 text-white font-medium rounded-lg hover:bg-indigo-700 disabled:opacity-50 transition-colors cursor-pointer"
              >
                {uploading ? t('branding.uploading') : t('branding.upload')}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
