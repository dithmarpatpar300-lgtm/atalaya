import type { ReactNode } from "react";

/** Iconos de trazo sobre una cuadrícula de 24. Siempre acompañan a un texto o llevan su nombre aparte. */
type Props = { className?: string };

function Trazo({ className = "size-5", children }: Props & { children: ReactNode }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      className={className}
    >
      {children}
    </svg>
  );
}

export function IconoCheck({ className }: Props) {
  return (
    <Trazo className={className}>
      <path d="M5 12.5l4.5 4.5L19 7.5" />
    </Trazo>
  );
}

export function IconoSol({ className }: Props) {
  return (
    <Trazo className={className}>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2.5v2M12 19.5v2M4.6 4.6l1.4 1.4M18 18l1.4 1.4M2.5 12h2M19.5 12h2M4.6 19.4L6 18M18 6l1.4-1.4" />
    </Trazo>
  );
}

export function IconoLuna({ className }: Props) {
  return (
    <Trazo className={className}>
      <path d="M20 14.5A8 8 0 019.5 4a7 7 0 1010.5 10.5z" />
    </Trazo>
  );
}

/** "El del sistema": una pantalla. */
export function IconoPantalla({ className }: Props) {
  return (
    <Trazo className={className}>
      <rect x="3" y="4" width="18" height="12" rx="2" />
      <path d="M8.5 20h7M12 16v4" />
    </Trazo>
  );
}

export function IconoChevron({ className }: Props) {
  return (
    <Trazo className={className}>
      <path d="m9 6 6 6-6 6" />
    </Trazo>
  );
}

export function IconoRecargar({ className }: Props) {
  return (
    <Trazo className={className}>
      <path d="M20 11a8 8 0 10-2.3 5.7M20 4v7h-7" />
    </Trazo>
  );
}

export function IconoBuscar({ className }: Props) {
  return (
    <Trazo className={className}>
      <circle cx="11" cy="11" r="6.5" />
      <path d="M16 16l4.5 4.5" />
    </Trazo>
  );
}

/** Abrir un menú de elección: dos flechas, arriba y abajo. */
export function IconoArribaAbajo({ className }: Props) {
  return (
    <Trazo className={className}>
      <path d="M8 9l4-4 4 4M8 15l4 4 4-4" />
    </Trazo>
  );
}

export function IconoAjustes({ className }: Props) {
  return (
    <Trazo className={className}>
      <path d="M4 7h10M18 7h2M4 17h4M12 17h8" />
      <circle cx="16" cy="7" r="2" />
      <circle cx="10" cy="17" r="2" />
    </Trazo>
  );
}

export function IconoCerrar({ className }: Props) {
  return (
    <Trazo className={className}>
      <path d="M6 6l12 12M18 6L6 18" />
    </Trazo>
  );
}

export function IconoCampana({ className }: Props) {
  return (
    <Trazo className={className}>
      <path d="M6 8a6 6 0 1 1 12 0c0 7 3 9 3 9H3s3-2 3-9" />
      <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" />
    </Trazo>
  );
}

export function IconoAviso({ className }: Props) {
  return (
    <Trazo className={className}>
      <path d="M12 9v4M12 17h.01" />
      <path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z" />
    </Trazo>
  );
}
