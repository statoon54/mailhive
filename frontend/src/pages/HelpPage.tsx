import { useState } from 'react';
import { useTranslation } from 'react-i18next';

function CodeBlock({ children }: { children: string }) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(children);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative group">
      <button
        onClick={handleCopy}
        className="absolute top-2 right-2 px-2 py-1 text-xs rounded bg-gray-700 text-gray-300 hover:bg-gray-600 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
      >
        {copied ? t('common.copied') : t('common.copy')}
      </button>
      <pre className="bg-gray-900 text-gray-100 rounded-lg p-4 text-sm overflow-x-auto leading-relaxed">
        <code>{children}</code>
      </pre>
    </div>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="mb-10">
      <h3 className="text-lg font-semibold text-gray-900 mb-3 flex items-center gap-2">
        {title}
      </h3>
      {children}
    </section>
  );
}

function Param({ name, type, required, children }: { name: string; type: string; required?: boolean; children: React.ReactNode }) {
  const { t } = useTranslation();
  return (
    <tr className="border-b border-gray-100">
      <td className="px-4 py-2 font-mono text-sm text-indigo-600">{name}</td>
      <td className="px-4 py-2 text-sm text-gray-500">{type}</td>
      <td className="px-4 py-2 text-sm">
        {required ? (
          <span className="text-red-500 font-medium text-xs">{t('common.required')}</span>
        ) : (
          <span className="text-gray-400 text-xs">{t('common.optional')}</span>
        )}
      </td>
      <td className="px-4 py-2 text-sm text-gray-700">{children}</td>
    </tr>
  );
}

export default function HelpPage() {
  const { t } = useTranslation();

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-2">{t('help.title')}</h2>
      <p className="text-gray-500 mb-8">{t('help.subtitle')}</p>

      <Section title={t('help.section1_title')}>
        <p className="text-sm text-gray-700 mb-3">{t('help.section1_desc')}</p>
        <CodeBlock>{`curl -X POST https://votre-domaine/api/v1/auth/token \\
  -H "Content-Type: application/json" \\
  -d '{"api_key": "votre-cle-api"}'`}</CodeBlock>
        <p className="text-sm text-gray-500 mt-3" dangerouslySetInnerHTML={{ __html: t('help.section1_note') }} />
      </Section>

      <Section title={t('help.section2_title')}>
        <p className="text-sm text-gray-700 mb-3">{t('help.section2_desc')}</p>
        <CodeBlock>{`curl -X POST https://votre-domaine/api/v1/mails \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <token>" \\
  -d '{
    "to": [
      {"email": "destinataire@example.com", "name": "Jean Dupont"}
    ],
    "subject": "Bienvenue",
    "text_body": "Bonjour Jean, bienvenue sur notre plateforme !",
    "html_body": "<h1>Bonjour Jean</h1><p>Bienvenue sur notre plateforme !</p>"
  }'`}</CodeBlock>
        <div className="mt-4 bg-amber-50 border border-amber-200 rounded-lg p-3 text-sm text-amber-800" dangerouslySetInnerHTML={{ __html: t('help.section2_note') }} />
      </Section>

      <Section title={t('help.section3_title')}>
        <div className="bg-white rounded-xl shadow-sm overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-gray-600 border-b">
              <tr>
                <th className="text-left px-4 py-2 font-medium">{t('help.param.field')}</th>
                <th className="text-left px-4 py-2 font-medium">{t('help.param.type')}</th>
                <th className="text-left px-4 py-2 font-medium">{t('help.param.status')}</th>
                <th className="text-left px-4 py-2 font-medium">{t('help.param.description')}</th>
              </tr>
            </thead>
            <tbody>
              <Param name="to" type="EmailAddress[]" required>{t('help.param.to')}</Param>
              <Param name="cc" type="EmailAddress[]">{t('help.param.cc')}</Param>
              <Param name="bcc" type="EmailAddress[]">{t('help.param.bcc')}</Param>
              <Param name="subject" type="string">{t('help.param.subject')}</Param>
              <Param name="text_body" type="string">{t('help.param.text_body')}</Param>
              <Param name="html_body" type="string">{t('help.param.html_body')}</Param>
              <Param name="from" type="EmailAddress">{t('help.param.from')}</Param>
              <Param name="smtp_config_id" type="uuid">{t('help.param.smtp_config_id')}</Param>
              <Param name="template_id" type="uuid">{t('help.param.template_id')}</Param>
              <Param name="template_data" type="object">{t('help.param.template_data')}</Param>
              <Param name="attachments" type="Attachment[]">{t('help.param.attachments')}</Param>
              <Param name="priority" type="string">{t('help.param.priority')}</Param>
              <Param name="individuel" type="boolean">{t('help.param.individuel')}</Param>
              <Param name="metadata" type="object">{t('help.param.metadata')}</Param>
              <Param name="scheduled_at" type="datetime">{t('help.param.scheduled_at')}</Param>
            </tbody>
          </table>
        </div>
        <p className="text-sm text-gray-500 mt-3" dangerouslySetInnerHTML={{ __html: t('help.section3_note') }} />
      </Section>

      <Section title={t('help.section4_title')}>
        <div className="bg-indigo-50 border border-indigo-200 rounded-lg p-4 text-sm text-indigo-800 space-y-2">
          <p dangerouslySetInnerHTML={{ __html: t('help.section4_default') }} />
          <p dangerouslySetInnerHTML={{ __html: t('help.section4_override') }} />
        </div>
        <CodeBlock>{`curl -X POST https://votre-domaine/api/v1/mails \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <token>" \\
  -d '{
    "from": {"email": "noreply@monsite.fr", "name": "Mon Site"},
    "to": [{"email": "user@example.com"}],
    "subject": "Notification",
    "text_body": "Ceci est envoyé depuis une adresse personnalisée."
  }'`}</CodeBlock>
      </Section>

      <Section title={t('help.section5_title')}>
        <p className="text-sm text-gray-700 mb-3">{t('help.section5_desc')}</p>
        <CodeBlock>{`curl -X POST https://votre-domaine/api/v1/mails \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <token>" \\
  -d '{
    "to": [{"email": "client@example.com", "name": "Marie Martin"}],
    "template_id": "uuid-du-template",
    "template_data": {
      "prenom": "Marie",
      "lien_activation": "https://monsite.fr/activer?token=abc123"
    }
  }'`}</CodeBlock>
        <p className="text-sm text-gray-500 mt-3">{t('help.section5_note')}</p>
      </Section>

      <Section title={t('help.section6_title')}>
        <p className="text-sm text-gray-700 mb-3" dangerouslySetInnerHTML={{ __html: t('help.section6_desc') }} />
        <CodeBlock>{`curl -X POST https://votre-domaine/api/v1/mails \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <token>" \\
  -d '{
    "to": [{"email": "user@example.com"}],
    "subject": "Votre facture",
    "text_body": "Veuillez trouver votre facture ci-jointe.",
    "attachments": [
      {
        "filename": "facture.pdf",
        "content_type": "application/pdf",
        "content": "JVBERi0xLjQK... (base64)"
      }
    ]
  }'`}</CodeBlock>
      </Section>

      <Section title={t('help.section7_title')}>
        <p className="text-sm text-gray-700 mb-3" dangerouslySetInnerHTML={{ __html: t('help.section7_desc') }} />
        <CodeBlock>{`curl -X POST https://votre-domaine/api/v1/mails \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <token>" \\
  -d '{
    "individuel": true,
    "template_id": "uuid-du-template",
    "template_data": {"entreprise": "ACME"},
    "to": [
      {"email": "alice@example.com", "name": "Alice", "template_data": {"prenom": "Alice"}},
      {"email": "bob@example.com", "name": "Bob", "template_data": {"prenom": "Bob"}}
    ]
  }'`}</CodeBlock>
        <div className="mt-4 bg-indigo-50 border border-indigo-200 rounded-lg p-3 text-sm text-indigo-800" dangerouslySetInnerHTML={{ __html: t('help.section7_note') }} />
      </Section>

      <Section title={t('help.section8_title')}>
        <p className="text-sm text-gray-700 mb-3">{t('help.section8_desc')}</p>
        <div className="bg-white rounded-xl shadow-sm overflow-hidden mb-3">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-gray-600 border-b">
              <tr>
                <th className="text-left px-4 py-2 font-medium">{t('help.priority.name')}</th>
                <th className="text-left px-4 py-2 font-medium">{t('help.priority.weight')}</th>
                <th className="text-left px-4 py-2 font-medium">{t('help.priority.usage')}</th>
              </tr>
            </thead>
            <tbody>
              <tr className="border-b border-gray-100">
                <td className="px-4 py-2 font-mono text-sm text-red-600">critical</td>
                <td className="px-4 py-2 text-sm">6</td>
                <td className="px-4 py-2 text-sm text-gray-700">{t('help.priority.critical_usage')}</td>
              </tr>
              <tr className="border-b border-gray-100">
                <td className="px-4 py-2 font-mono text-sm text-blue-600">default</td>
                <td className="px-4 py-2 text-sm">3</td>
                <td className="px-4 py-2 text-sm text-gray-700">{t('help.priority.default_usage')}</td>
              </tr>
              <tr>
                <td className="px-4 py-2 font-mono text-sm text-gray-500">low</td>
                <td className="px-4 py-2 text-sm">1</td>
                <td className="px-4 py-2 text-sm text-gray-700">{t('help.priority.low_usage')}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p className="text-sm text-gray-500" dangerouslySetInnerHTML={{ __html: t('help.section8_note') }} />
      </Section>

      <Section title={t('help.section9_title')}>
        <div className="bg-white rounded-xl shadow-sm p-4 text-sm space-y-3">
          <div>
            <p className="font-medium text-gray-900 mb-1">{t('help.section9_retry_title')}</p>
            <p className="text-gray-600" dangerouslySetInnerHTML={{ __html: t('help.section9_retry_desc') }} />
          </div>
          <div>
            <p className="font-medium text-gray-900 mb-1">{t('help.section9_cb_title')}</p>
            <p className="text-gray-600">{t('help.section9_cb_desc')}</p>
          </div>
          <div>
            <p className="font-medium text-gray-900 mb-1">{t('help.section9_perm_title')}</p>
            <p className="text-gray-600" dangerouslySetInnerHTML={{ __html: t('help.section9_perm_desc') }} />
          </div>
        </div>
      </Section>

      <Section title={t('help.section10_title')}>
        <p className="text-sm text-gray-700 mb-3" dangerouslySetInnerHTML={{ __html: t('help.section10_desc') }} />
        <div className="bg-white rounded-xl shadow-sm overflow-hidden mb-3">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-gray-600 border-b">
              <tr>
                <th className="text-left px-4 py-2 font-medium">{t('help.charset.title')}</th>
                <th className="text-left px-4 py-2 font-medium">{t('help.charset.desc')}</th>
              </tr>
            </thead>
            <tbody>
              <tr className="border-b border-gray-100">
                <td className="px-4 py-2 font-mono text-sm text-indigo-600">UTF-8</td>
                <td className="px-4 py-2 text-sm text-gray-700">{t('help.charset.utf8')}</td>
              </tr>
              <tr className="border-b border-gray-100">
                <td className="px-4 py-2 font-mono text-sm">US-ASCII</td>
                <td className="px-4 py-2 text-sm text-gray-700">{t('help.charset.ascii')}</td>
              </tr>
              <tr className="border-b border-gray-100">
                <td className="px-4 py-2 font-mono text-sm">ISO-8859-1</td>
                <td className="px-4 py-2 text-sm text-gray-700">{t('help.charset.latin1')}</td>
              </tr>
              <tr>
                <td className="px-4 py-2 font-mono text-sm">ISO-8859-15</td>
                <td className="px-4 py-2 text-sm text-gray-700">{t('help.charset.latin9')}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div className="bg-white rounded-xl shadow-sm overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-gray-600 border-b">
              <tr>
                <th className="text-left px-4 py-2 font-medium">{t('help.encoding.title')}</th>
                <th className="text-left px-4 py-2 font-medium">{t('help.encoding.desc')}</th>
              </tr>
            </thead>
            <tbody>
              <tr className="border-b border-gray-100">
                <td className="px-4 py-2 font-mono text-sm text-indigo-600">quoted-printable</td>
                <td className="px-4 py-2 text-sm text-gray-700">{t('help.encoding.qp')}</td>
              </tr>
              <tr className="border-b border-gray-100">
                <td className="px-4 py-2 font-mono text-sm">base64</td>
                <td className="px-4 py-2 text-sm text-gray-700">{t('help.encoding.base64')}</td>
              </tr>
              <tr className="border-b border-gray-100">
                <td className="px-4 py-2 font-mono text-sm">7bit</td>
                <td className="px-4 py-2 text-sm text-gray-700">{t('help.encoding.7bit')}</td>
              </tr>
              <tr>
                <td className="px-4 py-2 font-mono text-sm">8bit</td>
                <td className="px-4 py-2 text-sm text-gray-700">{t('help.encoding.8bit')}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </Section>

      <Section title={t('help.section11_title')}>
        <div className="space-y-3">
          <div>
            <p className="text-sm font-medium text-gray-700 mb-1">{t('help.section11_success')}</p>
            <CodeBlock>{`{
  "success": true,
  "data": {
    "id": "uuid-du-mail",
    "status": "pending",
    "from_email": "noreply@monsite.fr",
    "subject": "Bienvenue",
    ...
  }
}`}</CodeBlock>
          </div>
          <div>
            <p className="text-sm font-medium text-gray-700 mb-1">{t('help.section11_error')}</p>
            <CodeBlock>{`{
  "success": false,
  "error": "Données invalides",
  "fields": [
    {"field": "to", "message": "Au moins 1 élément(s) requis"},
    {"field": "email", "message": "L'adresse email n'est pas valide"}
  ]
}`}</CodeBlock>
          </div>
        </div>
      </Section>

      <Section title={t('help.section12_title')}>
        <p className="text-sm text-gray-700 mb-3">{t('help.section12_desc')}</p>
        <CodeBlock>{`curl https://votre-domaine/api/v1/mails/<mail-id> \\
  -H "Authorization: Bearer <token>"`}</CodeBlock>
        <p className="text-sm text-gray-500 mt-3" dangerouslySetInnerHTML={{ __html: t('help.section12_note') }} />
      </Section>
    </div>
  );
}
