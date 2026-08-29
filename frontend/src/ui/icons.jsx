export const stroke = {
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.6,
  strokeLinecap: "round",
  strokeLinejoin: "round",
};

export function IconCheck({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <path d="M20 6 9 17l-5-5" />
    </svg>
  );
}

export function IconAlert({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <path d="M12 9v4m0 4h.01M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" />
    </svg>
  );
}

export function IconUpload({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M17 8l-5-5-5 5M12 3v12" />
    </svg>
  );
}

export function IconDownload({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3" />
    </svg>
  );
}

export function IconSearch({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <circle cx="11" cy="11" r="7" />
      <path d="m21 21-4.3-4.3" />
    </svg>
  );
}

export function IconX({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <path d="M18 6 6 18M6 6l12 12" />
    </svg>
  );
}

export function IconChevron({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <path d="m6 9 6 6 6-6" />
    </svg>
  );
}

export function IconFile({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <path d="M14 2v6h6" />
    </svg>
  );
}

export function IconList({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <path d="M4 5h16M4 12h10M4 19h6" />
    </svg>
  );
}

export function IconPlus({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <path d="M12 5v14M5 12h14" />
    </svg>
  );
}
