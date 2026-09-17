import { useTranslation } from 'react-i18next';
import type { RefObject } from 'react';

type Field = HTMLInputElement | HTMLTextAreaElement;

interface VariablePickerProps {
  variables: string[];
  fieldRef: RefObject<Field | null>;
  value: string;
  onChange: (next: string) => void;
}

/**
 * VariablePicker affiche les variables déclarées sous forme de puces. Un clic
 * insère `{{.Nom}}` à la position du curseur dans le champ associé, en
 * remplaçant la sélection courante s'il y en a une.
 */
export default function VariablePicker({ variables, fieldRef, value, onChange }: VariablePickerProps) {
  const { t } = useTranslation();

  if (variables.length === 0) return null;

  const insert = (name: string) => {
    const token = `{{.${name}}}`;
    const field = fieldRef.current;
    if (!field) {
      onChange(value + token);
      return;
    }

    const start = field.selectionStart ?? value.length;
    const end = field.selectionEnd ?? start;
    onChange(value.slice(0, start) + token + value.slice(end));

    // React réécrit la valeur au rendu suivant, ce qui déplace le curseur en fin
    // de champ : on le replace derrière le jeton une fois le rendu passé.
    const caret = start + token.length;
    requestAnimationFrame(() => {
      field.focus();
      field.setSelectionRange(caret, caret);
    });
  };

  return (
    <div className="flex flex-wrap items-center gap-1.5 mb-1">
      <span className="text-xs text-gray-500">{t('templates.insert_variable')}</span>
      {variables.map((name) => (
        <button
          key={name}
          type="button"
          // Garde le curseur dans le champ : sans cela le clic le fait perdre le focus.
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => insert(name)}
          className="px-2 py-0.5 text-xs font-mono bg-indigo-50 text-indigo-700 border border-indigo-200 rounded hover:bg-indigo-100 cursor-pointer"
        >
          {`{{.${name}}}`}
        </button>
      ))}
    </div>
  );
}
