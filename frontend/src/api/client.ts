import axios from 'axios';

const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
});

// Intercepteur pour ajouter le JWT et la langue aux requêtes
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  const lang = localStorage.getItem('i18nextLng') || navigator.language || 'fr';
  config.headers['Accept-Language'] = lang;
  return config;
});

// Intercepteur pour gérer les erreurs d'authentification
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export default api;

// Types API
export interface Tenant {
  id: string;
  name: string;
  slug: string;
  api_key?: string;
  is_active: boolean;
  settings: {
    rate_limit: number;
    rate_burst: number;
    max_destinataires: number;
    default_priority: string;
    spam_score_threshold: number | undefined;
    spam_score_action: 'warn' | 'block' | undefined;
    language?: 'fr' | 'en';
    store_body: boolean;
  };
  created_at: string;
  updated_at: string;
}

export interface SMTPConfig {
  id: string;
  tenant_id: string;
  name: string;
  host: string;
  port: number;
  username?: string;
  auth_method: string;
  tls_policy: string;
  from_email: string;
  from_name: string;
  charset: string;
  encoding: string;
  is_default: boolean;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Template {
  id: string;
  tenant_id: string;
  name: string;
  slug: string;
  subject_tmpl: string;
  text_body: string;
  html_body: string;
  variables: Record<string, string>;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface MailRecipient {
  id: string;
  mail_id: string;
  type: 'to' | 'cc' | 'bcc';
  email: string;
  name?: string;
}

export interface AttachmentRef {
  attachment_id: string;
  filename: string;
  content_type: string;
  size: number;
}

export interface Mail {
  id: string;
  tenant_id: string;
  smtp_config_id?: string;
  template_id?: string;
  from_email: string;
  from_name: string;
  subject: string;
  text_body: string;
  html_body: string;
  status: string;
  status_message?: string;
  attempts: number;
  scheduled_at?: string;
  sent_at?: string;
  template_data?: Record<string, string>;
  recipients?: MailRecipient[];
  attachment_refs?: AttachmentRef[];
  metadata?: Record<string, unknown>;
  spam_score?: number;
  tags?: string[];
  created_at: string;
  updated_at: string;
}

export interface MailStats {
  pending: number;
  queued: number;
  sending: number;
  sent: number;
  failed: number;
  cancelled: number;
  rejected: number;
  total: number;
}

export interface TenantMailStats {
  tenant_id: string;
  tenant_name: string;
  sent: number;
  pending: number;
  failed: number;
  total: number;
}

export interface QueueInfo {
  name: string;
  active: number;
  pending: number;
  scheduled: number;
  retry: number;
  archived: number;
  completed: number;
  processed: number;
  failed: number;
  latency_ms: number;
  paused: boolean;
}

export interface PaginatedList<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface APIResponse<T> {
  success: boolean;
  data: T;
  message?: string;
}

export interface AuditLog {
  id: string;
  tenant_id: string;
  tenant_name: string;
  action: string;
  resource_type: string;
  resource_id: string;
  status: string;
  status_code: number;
  error_message: string;
  details: string;
  method: string;
  path: string;
  created_at: string;
}

export interface AppBranding {
  app_title: string;
  app_subtitle: string;
  timezone: string;
  logo_url: string;
  updated_at: string;
}

// Analysis types
export interface SpamRuleResult {
  name: string;
  description: string;
  score: number;
  details?: string;
}

export interface SpamCheckResult {
  score: number;
  max_score: number;
  rules: SpamRuleResult[];
  pass: boolean;
}

export interface HTMLCompatIssue {
  selector: string;
  property: string;
  description: string;
  severity: string;
  clients: string[];
}

export interface HTMLCheckResult {
  issues: HTMLCompatIssue[];
  total_count: number;
}

export interface LinkStatus {
  url: string;
  status_code?: number;
  status: string;
  details?: string;
  source: string;
}

export interface LinkCheckResult {
  links: LinkStatus[];
  total_count: number;
  broken_count: number;
}
