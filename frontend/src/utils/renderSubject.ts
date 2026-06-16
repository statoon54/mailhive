export function renderSubject(subject: string, data?: Record<string, string>): string {
  if (!data || !subject) return subject;
  return subject.replace(/\{\{\s*\.(\w+)\s*\}\}/g, (_, key) => data[key] ?? '');
}
