// Package demo prepara datos inventados para recorrer Atalaya sin instalaciones de agentes.
package demo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"atalaya/internal/proyectos"
)

// Mundo: las rutas que main pasa a sus opciones.
type Mundo struct {
	Raiz, Claude, Codex, Registro, Diario, DB, Cuotas string
}

// Preparar borra y vuelve a escribir el mundo en dir, con las horas contadas hacia atrás desde ahora.
func Preparar(dir string, ahora time.Time) (Mundo, error) {
	m := Mundo{Raiz: dir, Claude: filepath.Join(dir, "Claude"), Codex: filepath.Join(dir, "Codex"), Registro: filepath.Join(dir, "proyectos.json"), Diario: filepath.Join(dir, "diario"), DB: filepath.Join(dir, "atalaya.db"), Cuotas: filepath.Join(dir, "cuotas")}
	if err := os.RemoveAll(dir); err != nil {
		return m, err
	}
	for _, d := range []string{m.Claude, m.Codex, m.Diario, m.Cuotas} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return m, err
		}
	}
	escribir := func(ruta, contenido string) error {
		if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
			return err
		}
		return os.WriteFile(ruta, []byte(contenido), 0o644)
	}
	jsonTexto := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	hora := func(min int) string { return ahora.Add(-time.Duration(min) * time.Minute).Format(time.RFC3339Nano) }
	ps := []proyectos.Proyecto{}
	for _, p := range []struct {
		nombre, sigla, fase string
		nivel, pct          int
	}{{"Tienda", "TI", "F2", 3, 50}, {"Recetario", "RE", "F1", 2, 50}, {"Blog", "BL", "F0", 1, 100}} {
		raiz := filepath.Join(dir, "proyectos", p.nombre)
		ps = append(ps, proyectos.Proyecto{Nombre: p.nombre, Ruta: raiz, Sigla: p.sigla, Nivel: p.nivel})
		reglas := "# " + p.nombre + "\n\nNingún agente pasa de 30 minutos.\nEn paralelo, nunca hay dos sobre el mismo archivo.\nCada agente lleva una bitácora.\nEl acta es obligatoria.\n"
		filas := "| A | Preparación | 2 | hecho | 100 | Base lista |\n| B | Contenido | 2 | sin empezar | 0 | Pendiente |\n"
		if p.nombre == "Tienda" {
			filas = "| A | Catálogo | 2 | hecho | 100 | Listo |\n| B | Carrito | 2 | hecho | 100 | Listo |\n| C | Pagos | 2 | en curso | 50 | Revisión |\n| D | Cupones | 2 | sin empezar | 0 | Pendiente |\n| E | Envíos | 2 | sin empezar | 0 | Pendiente |\n"
		}
		if p.nombre == "Blog" {
			filas = "| A | Publicación | 2 | hecho | 100 | Lista |\n"
		}
		fase := fmt.Sprintf("Avance: %d %% — trabajo de %s\nPlan: docs/plan.md\n\n| Paquete | Qué | Horas | Estado | %% | Evidencia |\n|---|---|---|---|---|---|\n%s\n- %s · Seguimiento de la fase\n", p.pct, p.nombre, filas, ahora.Format("15:04"))
		archivos := map[string]string{"AGENTS.md": reglas, "docs/plan.md": "# Plan\n\nPreparar y revisar el proyecto.\n", "docs/catalogo.md": "# Catálogo\n\nContenido de ejemplo del proyecto.\n", "docs/trazabilidad/ESTADO-para-retomar.md": "# Estado\n\nSeguir con la fase " + p.fase + ".\n", "docs/trazabilidad/" + p.fase + "/AVANCE.md": fase,
			// El contrato de su nivel, para que la portada no lo muestre a medias.
			".mcp.json": "{\n  \"mcpServers\": {\n    \"atalaya\": { \"type\": \"http\", \"url\": \"http://127.0.0.1:7420/mcp?proyecto=" + p.nombre + "\" }\n  }\n}\n"}
		if p.nivel >= 2 {
			archivos["docs/operacion/AGENTES.md"] = "# Agentes\n\nClases, roles, modelos y effort de este proyecto.\n"
			archivos[".claude/agents/codificador.md"] = "---\nname: codificador\ndescription: Escribe el código de un encargo cerrado.\nmodel: sonnet\neffort: medium\n---\n\nEres el codificador.\n"
		}
		for ruta, contenido := range archivos {
			if err := escribir(filepath.Join(raiz, filepath.FromSlash(ruta)), contenido); err != nil {
				return m, err
			}
		}
	}
	if err := escribir(m.Registro, jsonTexto(ps)); err != nil {
		return m, err
	}
	carpeta := func(raiz string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return '-'
		}, raiz)
	}
	linea := func(tipo string, minuto int, cwd string, msg any) string {
		return jsonTexto(map[string]any{"type": tipo, "timestamp": hora(minuto), "cwd": cwd, "message": msg}) + "\n"
	}
	usuario := func(s string) any { return map[string]any{"role": "user", "content": s} }
	// Cada respuesta con su uso de tokens (Estadísticas y el detalle los suman); cifras plausibles, sin pretensión.
	asistente := func(id, modelo string, bloques ...any) any {
		return map[string]any{"id": id, "model": modelo, "role": "assistant", "content": bloques,
			"usage": map[string]any{"input_tokens": 1200, "cache_creation_input_tokens": 2400, "cache_read_input_tokens": 38000, "output_tokens": 900}}
	}
	uso := func(id, nombre string, input any) any {
		return map[string]any{"type": "tool_use", "id": id, "name": nombre, "input": input}
	}
	resultado := func(id, s string) any {
		return map[string]any{"role": "user", "content": []any{map[string]any{"type": "tool_result", "tool_use_id": id, "content": s}}}
	}
	type sub struct {
		nombre, clase, rol, modelo, effort string
		inicio, ultimo                     int
		terminado                          bool
	}
	// Las sesiones, en minutos hacia atrás desde ahora: la de hoy en Tienda trabaja y espera una respuesta; las de días
	// anteriores ya terminaron y dejan historia para Estadísticas, la línea de tiempo y el diario.
	sesiones := []struct {
		p                   int // índice en ps
		id, fase            string
		inicio, ultimo      int
		pregunta, respuesta string
		subs                []sub
	}{
		{0, "sesion-tienda", "F2", 50, 1, "¿Continuamos con la revisión del pago?", "", []sub{
			{"codificador-carrito", "I1", "codificador", "sonnet", "medium", 45, 27, true},
			{"auditor-pagos", "E2", "auditor", "opus", "high", 12, 1, false},
			{"codificador-cupones", "I2", "codificador", "sonnet", "high", 34, 1, false},
			{"codificador-envios", "I1", "codificador", "sonnet", "medium", 10, 1, false}}},
		{0, "sesion-tienda-ayer", "F2", 1500, 1380, "¿Mezclo el catálogo en la rama de trabajo?", "Sí, mézclalo", []sub{
			{"codificador-catalogo", "I1", "codificador", "sonnet", "medium", 1490, 1468, true},
			{"auditor-catalogo", "E1", "auditor", "sonnet", "high", 1462, 1449, true},
			{"depurador-fotos", "I2", "depurador", "opus", "high", 1440, 1414, true}}},
		{0, "sesion-tienda-lunes", "F1", 4400, 4250, "", "", []sub{
			{"arquitecto-pagos", "I2", "arquitecto", "opus", "high", 4390, 4362, true},
			{"codificador-precios", "I1", "codificador", "sonnet", "medium", 4350, 4331, true},
			{"ejecutor-migracion", "M", "ejecutor", "haiku", "low", 4320, 4314, true}}},
		{1, "sesion-recetario", "F1", 1450, 1443, "", "", []sub{
			{"codificador-recetas", "I1", "codificador", "sonnet", "medium", 1450, 1432, true}}},
	}
	for _, ses := range sesiones {
		p := ps[ses.p]
		sesion, fase, inicio, subs := ses.id, ses.fase, ses.inicio, ses.subs
		base := filepath.Join(m.Claude, carpeta(p.Ruta))
		trans := linea("user", inicio, p.Ruta, usuario("Prepara la siguiente fase del proyecto."))
		trans += linea("assistant", inicio-1, p.Ruta, asistente("m1", "claude-opus", uso("r1", "Read", map[string]any{"file_path": filepath.Join(p.Ruta, "docs", "plan.md")})))
		trans += linea("user", inicio-2, p.Ruta, resultado("r1", "Plan leído."))
		trans += linea("user", inicio-3, p.Ruta, usuario("Revisa el avance y prepara los cambios."))
		trans += linea("assistant", inicio-4, p.Ruta, asistente("m2", "claude-opus", uso("e1", "Edit", map[string]any{"file_path": filepath.Join(p.Ruta, "docs", "plan.md"), "old_string": "Preparar", "new_string": "Revisar"})))
		trans += linea("user", inicio-5, p.Ruta, resultado("e1", "Archivo actualizado."))
		trans += linea("assistant", inicio-6, p.Ruta, asistente("m3", "claude-opus", uso("b1", "Bash", map[string]any{"command": "go test ./...", "description": "Comprueba el proyecto"})))
		trans += linea("user", inicio-7, p.Ruta, resultado("b1", "ok: pruebas correctas"))
		if ses.pregunta != "" {
			// La pregunta con botones al cerrar el turno; con respuesta, el siguiente mensaje del usuario la contesta.
			cuando := ses.ultimo
			if ses.respuesta != "" {
				cuando = ses.ultimo + 2
			}
			trans += linea("assistant", cuando, p.Ruta, asistente("m4", "claude-opus", uso("w1", "mcp__ccd_turn__wrap_up", map[string]any{"recap": "El carrito está listo; pagos y envíos siguen en revisión.", "ask": ses.pregunta, "actions": []any{map[string]any{"label": "Continuar"}, map[string]any{"label": "Esperar"}}})))
			if ses.respuesta != "" {
				trans += linea("user", ses.ultimo, p.Ruta, usuario(ses.respuesta))
			}
		}
		rutaSesion := filepath.Join(base, sesion+".jsonl")
		if err := escribir(rutaSesion, trans); err != nil {
			return m, err
		}
		fechaSesion := ahora.Add(-time.Duration(ses.ultimo) * time.Minute)
		if err := os.Chtimes(rutaSesion, fechaSesion, fechaSesion); err != nil {
			return m, err
		}
		for _, s := range subs {
			bit := "docs/trazabilidad/" + fase + "/avance/" + s.nombre + ".md"
			encargo := fmt.Sprintf("**Asignación: nivel enfocado · clase %s · rol %s · %s/%s · presupuesto 30 min · base `main` en `abc123`**\nTu bitácora: `%s`.", s.clase, s.rol, s.modelo, s.effort, bit)
			contenido := linea("user", s.inicio, p.Ruta, usuario(encargo))
			archivo := filepath.Join(p.Ruta, "src", "carrito", "totales.go")
			herr := "Edit"
			input := map[string]any{"file_path": archivo, "old_string": "total", "new_string": "totalConDescuento"}
			if s.rol == "auditor" {
				herr = "Read"
				input = map[string]any{"file_path": filepath.Join(p.Ruta, "docs", "plan.md")}
			}
			contenido += linea("assistant", s.ultimo, p.Ruta, asistente("a1", "claude-"+s.modelo, uso("u1", herr, input)))
			if s.terminado {
				contenido += linea("user", s.ultimo, p.Ruta, resultado("u1", "Listo."))
				contenido += linea("assistant", s.ultimo, p.Ruta, asistente("a2", "claude-"+s.modelo, uso("fin", "SubagentHandback", map[string]any{})))
			}
			ruta := filepath.Join(base, sesion, "subagents", "agent-"+s.nombre)
			if err := escribir(ruta+".jsonl", contenido); err != nil {
				return m, err
			}
			if err := os.Chtimes(ruta+".jsonl", ahora.Add(-time.Duration(s.ultimo)*time.Minute), ahora.Add(-time.Duration(s.ultimo)*time.Minute)); err != nil {
				return m, err
			}
			if err := escribir(ruta+".meta.json", jsonTexto(map[string]any{"agentType": s.rol, "description": s.nombre, "model": s.modelo, "effort": s.effort})); err != nil {
				return m, err
			}
			pct := 65
			if s.terminado {
				pct = 100
			}
			if err := escribir(filepath.Join(p.Ruta, filepath.FromSlash(bit)), fmt.Sprintf("Avance: %d %% — %s\n\n- %s · Seguimiento del encargo\n", pct, s.nombre, ahora.Format("15:04"))); err != nil {
				return m, err
			}
		}
	}
	for _, c := range []struct {
		tema, clase string
		fin         int
		escribe     bool
	}{{"revision-pagos", "E1", 20, false}, {"ajuste-carrito", "I1", 5, true}} {
		base := filepath.Join(m.Codex, c.tema)
		meta := map[string]any{"tema": c.tema, "clase": c.clase, "modelo": "gpt-6.1-terra", "effort": "high", "fase": "F2", "raiz": ps[0].Ruta, "sesion": "sesion-tienda", "inicio": hora(c.fin + 8), "fin": hora(c.fin), "salida": 0, "segundos": 480, "escribe": c.escribe}
		if err := escribir(base+".meta.json", jsonTexto(meta)); err != nil {
			return m, err
		}
		traza := "{\"type\":\"thread.started\",\"thread_id\":\"demo\"}\n{\"type\":\"turn.started\"}\n" + jsonTexto(map[string]any{"type": "item.completed", "item": map[string]any{"id": "i1", "type": "agent_message", "text": "Revisión terminada: cambios listos para contrastar."}}) + "\n{\"type\":\"turn.completed\",\"usage\":{\"input_tokens\":1000,\"output_tokens\":100}}\n"
		if err := escribir(base+".jsonl", traza); err != nil {
			return m, err
		}
	}
	return m, nil
}
