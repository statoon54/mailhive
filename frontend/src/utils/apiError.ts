import type { AxiosError } from 'axios';

interface FieldError {
  field: string;
  message: string;
}

interface ApiErrorData {
  error?: string;
  fields?: FieldError[];
}

export function getApiError(error: unknown, fallback = 'Une erreur est survenue'): string {
  const axErr = error as AxiosError<ApiErrorData>;
  const data = axErr?.response?.data;
  if (!data) return fallback;

  if (data.fields && data.fields.length > 0) {
    return data.fields.map((f) => `${f.field} : ${f.message}`).join('\n');
  }

  return data.error || fallback;
}

export function getFieldErrors(error: unknown): Record<string, string> {
  const axErr = error as AxiosError<ApiErrorData>;
  const fields = axErr?.response?.data?.fields;
  if (!fields || fields.length === 0) return {};
  const map: Record<string, string> = {};
  for (const f of fields) {
    map[f.field] = f.message;
  }
  return map;
}
