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

export function IconEye({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

export function IconEyeOff({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <path d="M3 3l18 18M10.6 10.6A3 3 0 0 0 12 15a3 3 0 0 0 2.4-1.2M9.9 5.1A11 11 0 0 1 12 5c6 0 10 7 10 7a18 18 0 0 1-5.1 5.4M6.1 6.1C3.7 7.8 2 12 2 12a18 18 0 0 0 6.2 6.4" />
    </svg>
  );
}

export function IconCopy({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <rect x="9" y="9" width="11" height="11" rx="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
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

export function IconHelp({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <circle cx="12" cy="12" r="9" />
      <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
      <path d="M12 17h.01" />
    </svg>
  );
}

export function IconComputer({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <rect x="3" y="4" width="18" height="12" rx="2" />
      <path d="M8 20h8M12 16v4" />
    </svg>
  );
}

export function IconPhone({ className }) {
  return (
    <svg viewBox="0 0 24 24" className={className} {...stroke}>
      <rect x="7" y="2.5" width="10" height="19" rx="2" />
      <path d="M11 17.5h2" />
    </svg>
  );
}
