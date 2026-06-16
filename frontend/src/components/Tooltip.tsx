import { useState, type ReactNode } from 'react';

export default function Tooltip({ content, children }: { content: string; children: ReactNode }) {
  const [visible, setVisible] = useState(false);

  return (
    <span
      className="relative"
      onMouseEnter={() => setVisible(true)}
      onMouseLeave={() => setVisible(false)}
    >
      {children}
      {visible && (
        <span className="absolute z-50 bottom-full left-0 mb-2 px-3 py-2 text-xs text-white bg-gray-900 rounded-lg shadow-lg whitespace-pre-wrap max-w-sm max-h-48 overflow-y-auto break-words">
          {content}
        </span>
      )}
    </span>
  );
}
