import { useCallback, useEffect, useRef, useState, type ReactNode, type WheelEvent } from "react";
import { LocateFixed, Maximize2, ZoomIn, ZoomOut } from "lucide-react";

export interface GraphCanvasProps {
  /** Natural pixel size of the content passed as children, used to fit/center it. */
  contentWidth: number;
  contentHeight: number;
  children: ReactNode;
}

const MIN_ZOOM = 0.25;
const MAX_ZOOM = 2;
const ZOOM_STEP = 1.2;

interface Transform {
  x: number;
  y: number;
  scale: number;
}

// Shared pan/zoom viewport for node-graph views (trace service map, agent
// run graph). Content of unknown size no longer needs a scrollbar — it's
// rendered at its natural size inside a viewport the user pans and zooms,
// auto-fit on load and whenever contentWidth/contentHeight change.
//
// Dragging pans, but is never the only way to see content (WCAG 2.2 "Dragging
// Movements"): the view auto-fits everything on load, and the zoom-out/fit
// buttons reach the same result without a pointer drag.
export function GraphCanvas({ contentWidth, contentHeight, children }: GraphCanvasProps) {
  const viewportRef = useRef<HTMLDivElement>(null);
  const [transform, setTransform] = useState<Transform>({ x: 0, y: 0, scale: 1 });
  const [dragging, setDragging] = useState(false);
  const dragStart = useRef<{ x: number; y: number; tx: number; ty: number } | null>(null);
  const [viewportSize, setViewportSize] = useState({ width: 0, height: 0 });

  // Tracked purely to decide whether the content has been panned/zoomed
  // fully out of view (below) — not used for layout, so a ResizeObserver
  // is simpler here than measuring on every render.
  useEffect(() => {
    const el = viewportRef.current;
    if (!el) return;
    setViewportSize({ width: el.clientWidth, height: el.clientHeight });
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(([entry]) => {
      if (entry) setViewportSize({ width: entry.contentRect.width, height: entry.contentRect.height });
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const fitToView = useCallback(() => {
    const viewport = viewportRef.current;
    if (!viewport || contentWidth === 0 || contentHeight === 0) return;
    const { clientWidth, clientHeight } = viewport;
    const padding = 32;
    const scale = Math.min(
      MAX_ZOOM,
      Math.max(MIN_ZOOM, Math.min((clientWidth - padding) / contentWidth, (clientHeight - padding) / contentHeight, 1)),
    );
    setTransform({
      scale,
      x: (clientWidth - contentWidth * scale) / 2,
      y: (clientHeight - contentHeight * scale) / 2,
    });
  }, [contentWidth, contentHeight]);

  // Fit whenever the graph itself changes (new run/trace loaded), not on
  // every resize — a live resize keeping the user's current pan/zoom is
  // less surprising than silently re-fitting under their cursor.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(fitToView, [contentWidth, contentHeight]);

  const zoomBy = useCallback((factor: number, center?: { x: number; y: number }) => {
    setTransform((prev) => {
      const nextScale = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, prev.scale * factor));
      if (nextScale === prev.scale) return prev;
      const viewport = viewportRef.current;
      const cx = center?.x ?? (viewport ? viewport.clientWidth / 2 : 0);
      const cy = center?.y ?? (viewport ? viewport.clientHeight / 2 : 0);
      // Keep the point under the cursor (or viewport center) fixed while scaling.
      const ratio = nextScale / prev.scale;
      return {
        scale: nextScale,
        x: cx - (cx - prev.x) * ratio,
        y: cy - (cy - prev.y) * ratio,
      };
    });
  }, []);

  const handleWheel = useCallback(
    (e: WheelEvent<HTMLDivElement>) => {
      e.preventDefault();
      const viewport = viewportRef.current;
      if (!viewport) return;
      const rect = viewport.getBoundingClientRect();
      const factor = Math.exp(-e.deltaY * 0.001);
      zoomBy(factor, { x: e.clientX - rect.left, y: e.clientY - rect.top });
    },
    [zoomBy],
  );

  const handlePointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    if (e.button !== 0) return;
    dragStart.current = { x: e.clientX, y: e.clientY, tx: transform.x, ty: transform.y };
    setDragging(true);
    try {
      // Keeps the drag alive if the pointer leaves the viewport mid-move.
      // Can throw on some targets/browsers (e.g. a nested SVG child); a
      // failed capture only means dragging past the edge stops early, so
      // it must never take down the whole interaction.
      (e.target as Element).setPointerCapture(e.pointerId);
    } catch {
      // No-op: see comment above.
    }
  };

  const handlePointerMove = (e: React.PointerEvent<HTMLDivElement>) => {
    // Captured by value, not re-read from the ref inside the updater
    // below: pointerup can null dragStart.current before React actually
    // invokes a queued setState updater, and reading the ref there
    // instead of this local crashed with "Cannot read properties of
    // null" — the exact exception that, with no error boundary, blanked
    // the entire app rather than just this canvas.
    const start = dragStart.current;
    if (!start) return;
    const dx = e.clientX - start.x;
    const dy = e.clientY - start.y;
    setTransform((prev) => ({ ...prev, x: start.tx + dx, y: start.ty + dy }));
  };

  const endDrag = () => {
    dragStart.current = null;
    setDragging(false);
  };

  // True once the content's transformed bounding box no longer overlaps
  // the viewport at all — panned or zoomed fully away, not just off-center.
  // Only then is "back to content" worth showing; surfacing it any time
  // the user has merely panned a little would just be noise.
  const contentRight = transform.x + contentWidth * transform.scale;
  const contentBottom = transform.y + contentHeight * transform.scale;
  const isLost =
    viewportSize.width > 0 &&
    (contentRight < 0 || transform.x > viewportSize.width || contentBottom < 0 || transform.y > viewportSize.height);

  return (
    <div className="relative min-h-0 flex-1 overflow-hidden bg-canvas">
      <div
        ref={viewportRef}
        onWheel={handleWheel}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={endDrag}
        onPointerLeave={endDrag}
        className={`h-full w-full touch-none ${dragging ? "cursor-grabbing" : "cursor-grab"}`}
      >
        <div
          className={`relative ${dragging ? "" : "transition-transform duration-150 ease-out motion-reduce:transition-none"}`}
          style={{
            transform: `translate(${transform.x}px, ${transform.y}px) scale(${transform.scale})`,
            transformOrigin: "0 0",
            width: contentWidth,
            height: contentHeight,
          }}
        >
          {children}
        </div>
      </div>

      {isLost && (
        <button
          type="button"
          onClick={fitToView}
          className="absolute bottom-3 left-1/2 flex -translate-x-1/2 items-center gap-1.5 rounded-full border border-accent/40 bg-surface px-3 py-1.5 text-xs font-medium text-accent shadow-lg transition-colors hover:bg-elevated motion-reduce:transition-none"
        >
          <LocateFixed className="h-3.5 w-3.5" strokeWidth={2.25} />
          Back to content
        </button>
      )}

      <div className="absolute bottom-3 right-3 flex items-center gap-0.5 rounded-md border border-border bg-surface p-0.5 shadow-lg">
        <button
          type="button"
          aria-label="Zoom out"
          title="Zoom out"
          onClick={() => zoomBy(1 / ZOOM_STEP)}
          className="flex h-7 w-7 items-center justify-center rounded text-text-muted hover:bg-elevated hover:text-text-primary"
        >
          <ZoomOut className="h-3.5 w-3.5" strokeWidth={2.25} />
        </button>
        <span className="w-10 text-center font-plex-mono text-[10px] text-text-faint">
          {Math.round(transform.scale * 100)}%
        </span>
        <button
          type="button"
          aria-label="Zoom in"
          title="Zoom in"
          onClick={() => zoomBy(ZOOM_STEP)}
          className="flex h-7 w-7 items-center justify-center rounded text-text-muted hover:bg-elevated hover:text-text-primary"
        >
          <ZoomIn className="h-3.5 w-3.5" strokeWidth={2.25} />
        </button>
        <span className="mx-0.5 h-4 w-px bg-border" aria-hidden="true" />
        <button
          type="button"
          aria-label="Fit to view"
          title="Fit to view"
          onClick={fitToView}
          className="flex h-7 w-7 items-center justify-center rounded text-text-muted hover:bg-elevated hover:text-text-primary"
        >
          <Maximize2 className="h-3.5 w-3.5" strokeWidth={2.25} />
        </button>
      </div>
    </div>
  );
}
