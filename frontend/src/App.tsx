import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { BrandingProvider } from './contexts/BrandingContext';
import { ToastProvider } from './contexts/ToastContext';
import ProtectedRoute from './components/ProtectedRoute';
import Layout from './components/Layout';
import LoginPage from './pages/LoginPage';
import DashboardPage from './pages/DashboardPage';
import MailsPage from './pages/MailsPage';
import MailDetailPage from './pages/MailDetailPage';
import TemplatesPage from './pages/TemplatesPage';
import SMTPConfigsPage from './pages/SMTPConfigsPage';
import TenantsPage from './pages/TenantsPage';
import BrandingPage from './pages/BrandingPage';
import AuditLogsPage from './pages/AuditLogsPage';
import HelpPage from './pages/HelpPage';

export default function App() {
  return (
    <ToastProvider>
    <BrandingProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route
              element={
                <ProtectedRoute>
                  <Layout />
                </ProtectedRoute>
              }
            >
              <Route index element={<DashboardPage />} />
              <Route path="mails" element={<MailsPage />} />
              <Route path="mails/:id" element={<MailDetailPage />} />
              <Route path="templates" element={<TemplatesPage />} />
              <Route path="smtp-configs" element={<SMTPConfigsPage />} />
              <Route path="tenants" element={<TenantsPage />} />
              <Route path="branding" element={<BrandingPage />} />
              <Route path="audit-logs" element={<AuditLogsPage />} />
              <Route path="help" element={<HelpPage />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </BrandingProvider>
    </ToastProvider>
  );
}
