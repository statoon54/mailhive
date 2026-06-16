import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import api, { type AppBranding, type APIResponse } from '../api/client';

interface BrandingContextType {
  branding: AppBranding;
  refreshBranding: () => Promise<void>;
}

const defaultBranding: AppBranding = {
  app_title: 'MailHive',
  app_subtitle: 'Gestion des mails',
  timezone: 'Europe/Paris',
  logo_url: '/mailhive-logo.png',
  updated_at: '',
};

const BrandingContext = createContext<BrandingContextType | null>(null);

export function BrandingProvider({ children }: { children: ReactNode }) {
  const [branding, setBranding] = useState<AppBranding>(defaultBranding);

  const refreshBranding = useCallback(async () => {
    try {
      const response = await api.get<APIResponse<AppBranding>>('/branding');
      const data = response.data.data;
      setBranding({ ...data, logo_url: data.logo_url || defaultBranding.logo_url });
    } catch {
      // Garder les valeurs par défaut en cas d'erreur
    }
  }, []);

  useEffect(() => {
    refreshBranding();
  }, [refreshBranding]);

  useEffect(() => {
    document.title = branding.app_title;
  }, [branding.app_title]);

  return (
    <BrandingContext.Provider value={{ branding, refreshBranding }}>
      {children}
    </BrandingContext.Provider>
  );
}

export function useBranding() {
  const context = useContext(BrandingContext);
  if (!context) {
    throw new Error('useBranding doit être utilisé dans un BrandingProvider');
  }
  return context;
}
