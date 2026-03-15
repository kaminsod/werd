import { useState, useRef, useEffect, useCallback, type ReactNode } from "react";
import { createPortal } from "react-dom";

interface InfoIconProps {
  tooltip: string;
  children: ReactNode;
}

export default function InfoIcon({ tooltip, children }: InfoIconProps) {
  const [showTooltip, setShowTooltip] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [tooltipPos, setTooltipPos] = useState({ top: 0, left: 0 });
  const btnRef = useRef<HTMLButtonElement>(null);

  const updateTooltipPos = useCallback(() => {
    if (!btnRef.current) return;
    const rect = btnRef.current.getBoundingClientRect();
    const tipTop = rect.top - 6;
    const tipLeft = rect.left + rect.width / 2;
    setTooltipPos({ top: tipTop, left: tipLeft });
  }, []);

  // Recalculate position on scroll/resize while visible.
  useEffect(() => {
    if (!showTooltip) return;
    updateTooltipPos();
    window.addEventListener("scroll", updateTooltipPos, true);
    window.addEventListener("resize", updateTooltipPos);
    return () => {
      window.removeEventListener("scroll", updateTooltipPos, true);
      window.removeEventListener("resize", updateTooltipPos);
    };
  }, [showTooltip, updateTooltipPos]);

  // Close modal on Escape key.
  useEffect(() => {
    if (!showModal) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") setShowModal(false);
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [showModal]);

  return (
    <>
      <button
        ref={btnRef}
        type="button"
        onClick={(e) => { e.preventDefault(); e.stopPropagation(); setShowModal(true); }}
        onMouseEnter={() => { updateTooltipPos(); setShowTooltip(true); }}
        onMouseLeave={() => setShowTooltip(false)}
        className="ml-1 inline-flex h-4 w-4 shrink-0 items-center justify-center rounded-full bg-gray-200 text-[10px] font-bold leading-none text-gray-500 hover:bg-blue-100 hover:text-blue-600 align-middle"
        aria-label="More info"
      >
        i
      </button>

      {/* Tooltip — rendered in a portal with fixed positioning */}
      {showTooltip && !showModal && createPortal(
        <div
          style={{ top: tooltipPos.top, left: tooltipPos.left }}
          className="pointer-events-none fixed z-[9999] -translate-x-1/2 -translate-y-full whitespace-nowrap rounded bg-gray-800 px-2.5 py-1 text-xs text-white shadow-lg"
        >
          {tooltip}
          <div className="absolute left-1/2 top-full -translate-x-1/2 border-4 border-transparent border-t-gray-800" />
        </div>,
        document.body,
      )}

      {/* Modal on click */}
      {showModal && createPortal(
        <div
          className="fixed inset-0 z-[9999] flex items-center justify-center bg-black/40 p-4"
          onClick={() => setShowModal(false)}
        >
          <div
            className="max-h-[80vh] w-full max-w-md overflow-y-auto rounded-lg bg-white p-5 shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="mb-4 flex items-start justify-between">
              <div className="text-sm leading-relaxed text-gray-700">
                {children}
              </div>
              <button
                onClick={() => setShowModal(false)}
                className="ml-4 shrink-0 rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                aria-label="Close"
              >
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>
        </div>,
        document.body,
      )}
    </>
  );
}
