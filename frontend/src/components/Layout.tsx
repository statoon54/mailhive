import { NavLink, Outlet } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../contexts/AuthContext';
import { useBranding } from '../contexts/BrandingContext';

export default function Layout() {
  const { t } = useTranslation();
  const { logout, isAdmin } = useAuth();
  const { branding } = useBranding();

  const navItems = [
    { to: '/', label: t('nav.dashboard'), icon: '◈' },
    { to: '/tenants', label: t('nav.tenants'), icon: '⊞' },
    { to: '/mails', label: t('nav.mails'), icon: '✉' },
    { to: '/templates', label: t('nav.templates'), icon: '❐' },
    { to: '/smtp-configs', label: t('nav.smtp_configs'), icon: '⚙' },
    { to: '/audit-logs', label: t('nav.audit_logs'), icon: '⊘' },
    { to: '/help', label: t('nav.help'), icon: '?' },
  ];

  const adminNavItems = [
    { to: '/branding', label: t('nav.branding'), icon: '✦' },
  ];

  const adminItems = [
    { href: '/monitoring', label: t('nav.monitoring'), icon: '⊙' },
  ];

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Sidebar */}
      <aside className="w-64 bg-gray-900 text-white flex flex-col">
        <div className="p-6 border-b border-gray-700">
          <div className="flex items-center gap-3">
            {branding.logo_url && (
              <img src={branding.logo_url} alt="Logo" className="h-8 w-8 object-contain" />
            )}
            <div>
              <h1 className="text-xl font-bold tracking-tight">{branding.app_title}</h1>
              <p className="text-xs text-gray-400 mt-1">{branding.app_subtitle}</p>
            </div>
          </div>
        </div>

        <nav className="flex-1 py-4">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) =>
                `flex items-center gap-3 px-6 py-3 text-sm transition-colors ${
                  isActive
                    ? 'bg-indigo-600 text-white font-medium'
                    : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                }`
              }
            >
              <span className="text-lg">{item.icon}</span>
              {item.label}
            </NavLink>
          ))}
        </nav>

        {isAdmin && (
          <div className="border-t border-gray-700 py-4">
            <p className="px-6 pb-2 text-xs font-semibold text-gray-500 uppercase tracking-wider">{t('nav.admin')}</p>
            {adminNavItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  `flex items-center gap-3 px-6 py-3 text-sm transition-colors ${
                    isActive
                      ? 'bg-indigo-600 text-white font-medium'
                      : 'text-gray-300 hover:bg-gray-800 hover:text-white'
                  }`
                }
              >
                <span className="text-lg">{item.icon}</span>
                {item.label}
              </NavLink>
            ))}
            {adminItems.map((item) => (
              <a
                key={item.href}
                href={item.href}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-3 px-6 py-3 text-sm text-gray-300 hover:bg-gray-800 hover:text-white transition-colors"
              >
                <span className="text-lg">{item.icon}</span>
                {item.label}
                <span className="ml-auto text-xs text-gray-500">↗</span>
              </a>
            ))}
          </div>
        )}

        <div className="p-4 border-t border-gray-700 flex flex-col gap-2">
          <button
            onClick={logout}
            className="w-full px-4 py-2 text-sm text-gray-300 hover:text-white hover:bg-gray-800 rounded transition-colors cursor-pointer"
          >
            {t('common.logout')}
          </button>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <div className="p-8">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
