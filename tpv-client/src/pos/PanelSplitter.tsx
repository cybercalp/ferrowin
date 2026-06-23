import { useCallback, useRef, useState } from "react";

interface Props {
  onWidthChange: (px: number) => void;
  onCommit: (px: number) => void;
  min?: number;
  max?: number;
}

/**
 * A vertical splitter handle placed between .pos-left-panel and
 * .pos-right-panel.  Uses mousedown/mousemove/mouseup to track
 * drag and report the new right-panel width.
 *
 * Reads the starting width from the DOM at mousedown so it works
 * regardless of what CSS or inline styles set it.
 */
export function PanelSplitter({
  onWidthChange,
  onCommit,
  min = 320,
  max = 700,
}: Props) {
  const [active, setActive] = useState(false);
  const dragRef = useRef<{
    startX: number;
    startWidth: number;
    lastWidth: number;
  } | null>(null);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      // Read the current rendered width of the right panel at drag start
      const panel = (e.currentTarget as HTMLElement)
        .closest(".pos-body")
        ?.querySelector(".pos-right-panel") as HTMLElement | null;
      if (!panel) return;
      const startWidth = panel.offsetWidth;
      dragRef.current = { startX: e.clientX, startWidth, lastWidth: startWidth };
      setActive(true);

      const onMove = (ev: MouseEvent) => {
        const d = dragRef.current;
        if (!d) return;
        const delta = d.startX - ev.clientX;
        d.lastWidth = Math.min(max, Math.max(min, d.startWidth + delta));
        onWidthChange(d.lastWidth);
      };

      const onUp = () => {
        const d = dragRef.current;
        if (d) onCommit(d.lastWidth);
        dragRef.current = null;
        setActive(false);
        document.removeEventListener("mousemove", onMove);
        document.removeEventListener("mouseup", onUp);
      };

      document.addEventListener("mousemove", onMove);
      document.addEventListener("mouseup", onUp);
    },
    [onWidthChange, onCommit, min, max],
  );

  return (
    <div
      className={`panel-splitter${active ? " splitter-active" : ""}`}
      onMouseDown={handleMouseDown}
    />
  );
}
