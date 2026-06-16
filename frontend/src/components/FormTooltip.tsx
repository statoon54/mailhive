import { useCallback, useRef, useState } from 'react';

export default function FormTooltip({ text }: { text: string }) {
  const [visible, setVisible] = useState(false);
  const [position, setPosition] = useState<'top' | 'bottom'>('top');
  const triggerRef = useRef<HTMLSpanElement>(null);

  const show = useCallback(() => {
    if (triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect();
      // Si pas assez de place au-dessus (< 80px), afficher en dessous
      setPosition(rect.top < 80 ? 'bottom' : 'top');
    }
    setVisible(true);
  }, []);

  const posClass = position === 'top'
    ? 'bottom-full left-1/2 -translate-x-1/2 mb-2'
    : 'top-full left-1/2 -translate-x-1/2 mt-2';

  return (
    <span
      ref={triggerRef}
      className="relative inline-flex items-center ml-1 cursor-help"
      onMouseEnter={show}
      onMouseLeave={() => setVisible(false)}
    >
      <span className="w-4 h-4 inline-flex items-center justify-center rounded-full bg-gray-200 text-gray-500 text-[10px] font-bold leading-none">?</span>
      {visible && (
        <span className={`absolute z-50 ${posClass} px-3 py-2 text-xs text-white bg-gray-900 rounded-lg shadow-lg whitespace-normal max-w-xs break-words`}>
          {text}
        </span>
      )}
    </span>
  );
}
