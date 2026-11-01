package claude

import (
	"testing"
	"time"
)

func TestTipoPedidoYSistema(t *testing.T) {
	casos := map[string]string{
		"Sí, comitéalo":       "permiso",
		"si, relánzala":       "permiso",
		"Dale":                "permiso",
		"Sin embargo, quiero": "pedido",
		"Como vamos en los avances de la Fase 5, dame toda la lista": "pedido",
	}
	for in, want := range casos {
		if got := tipoPedido(in); got != want {
			t.Errorf("%q: %s, quería %s", in, got, want)
		}
	}
	if !esDeSistema("Another Claude session sent a message: …") || esDeSistema("Sí, lánzalo") {
		t.Error("mensajes de sistema")
	}
	for _, o := range []string{`git -C 'C:\Proyectos\tienda demo' commit -F d:/tmp/commit.txt`, `cmd.exe /d /c "git commit -m x"`, `git add a; git commit -q`} {
		if !reGitCommit.MatchString(Normalizar(o)) {
			t.Errorf("no ve el commit en %q", o)
		}
	}
	for _, o := range []string{"git status --short", "echo commit", "git log; echo commit"} {
		if reGitCommit.MatchString(Normalizar(o)) {
			t.Errorf("ve un commit que no es en %q", o)
		}
	}
}

func TestSesionResume(t *testing.T) {
	h := time.Date(2026, 10, 31, 12, 20, 0, 0, time.UTC)
	ses := &archivo{agente: &Agente{ID: "s", Tipo: "sesion"}}
	sub := &archivo{agente: &Agente{ID: "s~a", Tipo: "subagente", Padre: "s", ToolUseID: "tu1"}}
	ses.pasos = []*Paso{
		{N: 0, Hora: h, Tipo: "Pide", Resumen: "Retomas Tienda Demo", Etapa: ""},
		{N: 1, Hora: h, Tipo: "Usa", Herramienta: "mcp__atalaya__salud", Etapa: "Retomar"},
		{N: 2, Hora: h, Tipo: "Edita", editado: "c:/proyectos/tienda demo/docs/trazabilidad/estado-para-retomar.md", Hilo: "docs/trazabilidad/ESTADO-para-retomar.md", Etapa: "Retomar"},
		{N: 3, Hora: h.Add(5 * time.Minute), Tipo: "Pide", Resumen: "Sí, comitéalo"},
		{N: 4, Hora: h.Add(5 * time.Minute), Tipo: "Ejecuta", orden: "git commit -F msg.txt", Etapa: "Comitear"},
		{N: 5, Hora: h.Add(6 * time.Minute), Tipo: "Pide", Resumen: "Another Claude session sent a message: listo"},
		{N: 6, Hora: h.Add(6 * time.Minute), Tipo: "Ejecuta", orden: "cuotas.exe una-vez --json", Etapa: "Mirar la cuota"},
		{N: 7, Hora: h.Add(7 * time.Minute), Tipo: "Delega", idUso: "tu1", Etapa: "Lanzar"},
		{N: 8, Hora: h.Add(8 * time.Minute), Tipo: "Responde", Resumen: "Lanzado.", Herramienta: "Respuesta"},
	}
	ix := &Indice{arch: map[string]*archivo{"s": ses, "a": sub}}
	r, ok := ix.Sesion("s")
	if !ok || len(r.Pedidos) != 2 || r.Mensajes != 1 {
		t.Fatalf("pedidos: %+v", r)
	}
	p0, p1 := r.Pedidos[0], r.Pedidos[1]
	if p0.Tipo != "pedido" || p0.Consultas != 1 || len(p0.Etapas) != 1 {
		t.Errorf("primer pedido: %+v", p0)
	}
	// El aviso de sistema no corta el pedido: lo que sigue es del «Sí, comitéalo».
	if p1.Tipo != "permiso" || p1.Commits != 1 || p1.Cuota != 1 || len(p1.Lanzados) != 1 || p1.Lanzados[0] != "s~a" || p1.Respuesta != "Lanzado." {
		t.Errorf("segundo pedido: %+v", p1)
	}
	if len(r.Documentos) != 1 || r.Documentos[0].Ediciones != 1 || len(r.Commits) != 1 || r.Lecturas != 1 || r.Consultas != 1 {
		t.Errorf("resumen: %+v", r)
	}
	if _, ok := ix.Sesion("s~a"); ok {
		t.Error("un subagente no es una sesión")
	}
}
