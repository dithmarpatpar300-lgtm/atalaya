/**
 * La marca: tres hilos que se separan y vuelven a juntarse. Arriba Claude (ciruela), al medio Codex (celeste) y
 * abajo el riel neutro; es el motivo de toda la identidad «Grafito con hilos».
 */
export function Marca({ className = "size-[22px]" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" fill="none" aria-hidden="true" focusable="false" className={className}>
      <path d="M4 5c5 0 6 7 8 7s3-7 8-7" stroke="var(--acento)" strokeWidth="2" strokeLinecap="round" />
      <path d="M4 12h16" stroke="var(--opcional)" strokeWidth="2" strokeLinecap="round" />
      <path d="M4 19c5 0 6-7 8-7s3 7 8 7" stroke="var(--texto-suave)" strokeWidth="2" strokeLinecap="round" />
    </svg>
  );
}
