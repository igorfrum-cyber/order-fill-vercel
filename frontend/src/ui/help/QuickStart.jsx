import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { tourForRole } from "../../features/help/copy.js";
import { isLastTourStep, nextTourIndex, prevTourIndex, spotlightRect, tooltipLayout, visibleTourSteps } from "../../features/help/tour.js";
import { IconChevron } from "../icons.jsx";
import { GhostButton, PrimaryButton } from "../widgets.jsx";

const CARD_WIDTH = 320;

export function QuickStart({ me, onDismiss, onLater }) {
  const role = me.role;
  const [steps, setSteps] = useState(() => tourForRole(role));
  const [index, setIndex] = useState(0);
  const [cardSize, setCardSize] = useState({ width: CARD_WIDTH, height: 180 });
  const [displaySpot, setDisplaySpot] = useState(null);
  const [entered, setEntered] = useState(false);
  const cardRef = useRef(null);
  const step = steps[index] || steps[0];
  const last = isLastTourStep(index, steps.length);

  useLayoutEffect(() => {
    const catalog = tourForRole(role);
    const visible = visibleTourSteps(catalog, (id) => Boolean(document.querySelector(`[data-tour="${id}"]`)));
    setSteps(visible.length ? visible : catalog);
  }, [role]);

  useLayoutEffect(() => {
    if (!step) return undefined;

    function measure() {
      const node = document.querySelector(`[data-tour="${step.target}"]`);
      setDisplaySpot(node ? spotlightRect(node.getBoundingClientRect(), 8) : null);
      if (!cardRef.current) return;
      const rect = cardRef.current.getBoundingClientRect();
      setCardSize((current) =>
        current.width === rect.width && current.height === rect.height ? current : { width: rect.width, height: rect.height },
      );
    }

    measure();
    const frame = requestAnimationFrame(() => setEntered(true));
    window.addEventListener("resize", measure);
    window.addEventListener("scroll", measure, true);
    return () => {
      cancelAnimationFrame(frame);
      window.removeEventListener("resize", measure);
      window.removeEventListener("scroll", measure, true);
    };
  }, [index, step?.target]);

  useEffect(() => {
    function onKey(event) {
      if (event.key === "Escape") onLater();
      if (event.key === "ArrowRight") {
        if (last) onDismiss();
        else setIndex((current) => nextTourIndex(current, steps.length));
      }
      if (event.key === "ArrowLeft") setIndex((current) => prevTourIndex(current, steps.length));
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [last, onDismiss, onLater, steps.length]);

  const viewport = { width: window.innerWidth, height: window.innerHeight };
  if (!step) return null;
  const card = displaySpot
    ? tooltipLayout(displaySpot, cardSize, viewport, step.placement || "bottom")
    : { left: Math.max(16, (viewport.width - CARD_WIDTH) / 2), top: Math.max(16, viewport.height / 3), arrow: null, arrowLeft: CARD_WIDTH / 2 };

  return (
    <div className={`tour-shell ${entered ? "tour-shell-ready" : ""}`} role="dialog" aria-modal="true" aria-labelledby="quick-start-title">
      <div className="absolute inset-0" />
      {displaySpot ? (
        <div
          className="tour-spot pointer-events-none absolute rounded-2xl bg-transparent"
          style={{ left: displaySpot.left, top: displaySpot.top, width: displaySpot.width, height: displaySpot.height }}
        />
      ) : (
        <div className="pointer-events-none absolute inset-0 bg-slate-900/55" />
      )}
      <div
        ref={cardRef}
        className="tour-card absolute w-[min(320px,calc(100vw-32px))] overflow-visible rounded-xl border border-[var(--color-line)] bg-[var(--color-surface)] p-5"
        style={{ left: card.left, top: card.top }}
      >
        {card.arrow ? <TourArrow side={card.arrow} offset={card.arrowLeft} /> : null}
        <div key={step.target} className="tour-copy">
          <p className="text-[12px] font-medium text-[var(--color-brand)]">
            Шаг {index + 1} из {steps.length}
          </p>
          <h2 id="quick-start-title" className="mt-1 text-[18px] font-semibold tracking-tight">
            {step.title}
          </h2>
          <p className="mt-2 text-[14px] leading-relaxed text-[var(--color-ink-soft)]">{step.body}</p>
        </div>
        <div className="mt-4 flex flex-wrap items-center justify-between gap-2">
          <div className="flex items-center gap-1.5">
            {steps.map((item, stepIndex) => (
              <span
                key={item.target}
                className={`tour-dot h-1.5 rounded-full ${stepIndex === index ? "tour-dot-active bg-[var(--color-brand)]" : "w-1.5 bg-[var(--color-line)]"}`}
              />
            ))}
          </div>
          <div className="flex gap-2">
            {index > 0 ? (
              <GhostButton onClick={() => setIndex((current) => prevTourIndex(current, steps.length))}>
                <IconChevron className="h-4 w-4 rotate-90" />
                Назад
              </GhostButton>
            ) : (
              <GhostButton onClick={onLater}>Позже</GhostButton>
            )}
            <PrimaryButton
              onClick={() => {
                if (last) onDismiss();
                else setIndex((current) => nextTourIndex(current, steps.length));
              }}
            >
              {last ? "Понятно" : "Далее"}
              {last ? null : <IconChevron className="h-4 w-4 -rotate-90" />}
            </PrimaryButton>
          </div>
        </div>
      </div>
    </div>
  );
}

function TourArrow({ side, offset }) {
  const triangle = {
    top: "top-0 -translate-x-1/2 -translate-y-full border-x-8 border-b-[10px] border-x-transparent border-b-[var(--color-surface)]",
    bottom: "bottom-0 -translate-x-1/2 translate-y-full border-x-8 border-t-[10px] border-x-transparent border-t-[var(--color-surface)]",
    left: "left-0 top-1/2 -translate-x-full -translate-y-1/2 border-y-8 border-r-[10px] border-y-transparent border-r-[var(--color-surface)]",
    right: "right-0 top-1/2 translate-x-full -translate-y-1/2 border-y-8 border-l-[10px] border-y-transparent border-l-[var(--color-surface)]",
  }[side];
  const along = side === "top" || side === "bottom" ? { left: offset, transition: "left 0.5s cubic-bezier(0.22, 1, 0.36, 1)" } : { transition: "top 0.5s cubic-bezier(0.22, 1, 0.36, 1)" };
  return <span className={`absolute h-0 w-0 ${triangle}`} style={along} aria-hidden />;
}
