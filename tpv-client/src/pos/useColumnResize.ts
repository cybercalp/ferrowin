import { useCallback, useRef, useState } from "react";

const STORAGE_KEY = "ferrowin:colWidths";
const MIN = 40;
const MAX = 500;
const DEFAULTS: Record<string, number | null> = {
  imagen: 60,
  codigo: 100,
  nombre: null, // null = flexible (fills remaining space)
  ud: 60,
  precio: 100,
  add: 48,
};

function load(): Record<string, number | null> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? { ...DEFAULTS, ...JSON.parse(raw) } : { ...DEFAULTS };
  } catch {
    return { ...DEFAULTS };
  }
}

export interface ColResizeApi {
  widths: Record<string, number | null>;
  getStyle: (key: string) => React.CSSProperties | undefined;
  createHandler: (key: string) => (e: React.MouseEvent) => void;
}

/**
 * Hook for draggable column-width persistence.
 *
 * `widths[key]` — pixel value or `null` for flexible (no explicit width).
 * `getStyle(key)` — returns `{ width: px }` or `undefined` for flexible.
 * `createHandler(key)` — returns mousedown handler for the resize handle.
 */
export function useColumnResize(): ColResizeApi {
  const [widths, setWidths] = useState<Record<string, number | null>>(load);

  // Ref for the commit callback so it always sees the latest widths
  const widthsRef = useRef(widths);
  widthsRef.current = widths;

  const commit = useCallback(() => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(widthsRef.current));
  }, []);

  const getStyle = useCallback(
    (key: string): React.CSSProperties | undefined => {
      const w = widths[key];
      return w != null ? { width: w } : undefined;
    },
    [widths],
  );

  const createHandler = useCallback(
    (key: string) => {
      return (e: React.MouseEvent) => {
        e.preventDefault();
        e.stopPropagation();

        const startX = e.clientX;
        // Read the ACTUAL rendered width from the <th> so dragging
        // starts smoothly even for columns with no stored width yet.
        const th = (e.currentTarget as HTMLElement).closest("th");
        const renderedW = th ? th.offsetWidth : 0;
        const startW = renderedW > 0 ? renderedW : (widthsRef.current[key] ?? DEFAULTS[key] ?? 80);

        // Clamp startWidth to sensible bounds
        const clampedStart = Math.min(MAX, Math.max(MIN, startW));

        const onMove = (ev: MouseEvent) => {
          const delta = ev.clientX - startX;
          const newW = Math.min(MAX, Math.max(MIN, clampedStart + delta));
          setWidths((prev) => ({ ...prev, [key]: newW }));
        };

        const onUp = () => {
          document.removeEventListener("mousemove", onMove);
          document.removeEventListener("mouseup", onUp);
          commit();
        };

        document.addEventListener("mousemove", onMove);
        document.addEventListener("mouseup", onUp);
      };
    },
    [commit],
  );

  return { widths, getStyle, createHandler };
}
