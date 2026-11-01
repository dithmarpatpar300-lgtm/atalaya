import type { Register } from 'claude-code'

// Atalaya lee la cuota con `claude -p /usage` cada 10 minutos; este mod le avisa en el momento en que la app mueve un
// punto entero de la ventana de 5 horas o de la semanal, sin esperar a la próxima lectura.
const DESTINO = 'http://127.0.0.1:7420/api/uso-claude/empujar'
const VENTANAS = new Set(['five_hour', 'seven_day'])

export const register: Register = on => {
  on('session.measure', async ($, e, next) => {
    const r = await next(e)
    if (!e.changed.includes('rateLimits')) return r
    const rate_limits = e.rateLimits
      .filter(l => VENTANAS.has(l.kind) && l.resetsAt)
      .map(l => ({ kind: l.kind, percent_used: l.percentUsed, resets_at: l.resetsAt }))
    if (rate_limits.length > 0) {
      // Sin esperar: si Atalaya no está corriendo, no hay nada que avisar y la sesión sigue igual.
      void $.http
        .fetch(DESTINO, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-Atalaya-Mod': 'uso-claude' },
          body: JSON.stringify({ rate_limits }),
        })
        .catch(() => undefined)
    }
    return r
  })
}
