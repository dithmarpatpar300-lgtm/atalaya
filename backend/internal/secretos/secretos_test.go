package secretos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// sinPistas: Tapar como era antes de las pistas, todas las expresiones siempre. Es la referencia.
func sinPistas(s string) string {
	for _, p := range patrones {
		s = p.re.ReplaceAllString(s, p.por)
	}
	return s
}

var ejemplos = []string{
	"export ANTHROPIC_API_KEY=sk-ant-api03-abcdefghijklmnop",
	"OPENAI sk-proj-abcdefghijklmnopqrstuvwxyz0123",
	"remote: ghp_abcdefghijklmnopqrstuvwxyz0123 y github_pat_abcdefghijklmnopqrstuvwxyz",
	"aws AKIAABCDEFGHIJKLMNOP listo",
	"Authorization: BEARER abcdefghijklmnopqrstuvwxyz==",
	"curl -H 'authorization: bearer abc.def-ghi_jkl~mno+pqr/stu'",
	`{"Access_Token": "abcdef123456", "PASSWORD":"hunter2hunter2"}`,
	`{"Api_Key" : "0123456789abcdef"}`,
	"https://x.blob.core.windows.net/c?sv=1&SIG=abcdefghijkl&se=2",
	"https://s3.amazonaws.com/b?X-Amz-Signature=abcdef0123456789&x=1",
	"https://api.example.com/v1?token=abcdefghij",
	"jwt eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abcdefghijklmnop",
	"nada secreto aquí: go test ./... y un AVANCE.md con 88,8 %",
	"",
}

func TestTaparIgualQueSinPistas(t *testing.T) {
	for _, e := range ejemplos {
		if got, want := Tapar(e), sinPistas(e); got != want {
			t.Errorf("Tapar(%q)\n = %q\nquiero %q", e, got, want)
		}
	}
}

func TestTaparTapaCadaPatron(t *testing.T) {
	for _, e := range ejemplos[:12] {
		if got := Tapar(e); got == e || !strings.Contains(got, "tapado") {
			t.Errorf("no tapó %q: %q", e, got)
		}
	}
}

// LeerTapado: lee tapado, guarda lo leído y vuelve a leer cuando el archivo cambia.
func TestLeerTapado(t *testing.T) {
	r := filepath.Join(t.TempDir(), "AVANCE.md")
	if err := os.WriteFile(r, []byte("token sk-ant-api03-abcdefghijklmnop"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := LeerTapado(r)
	if err != nil || a != "token sk-ant-[tapado]" {
		t.Fatalf("primera: %q %v", a, err)
	}
	if b, _ := LeerTapado(r); b != a {
		t.Fatalf("de la caché: %q", b)
	}
	if err := os.WriteFile(r, []byte("Avance: 50 %"), 0o644); err != nil {
		t.Fatal(err)
	}
	luego := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(r, luego, luego)
	if c, _ := LeerTapado(r); c != "Avance: 50 %" {
		t.Fatalf("tras cambiar: %q", c)
	}
	if _, err := LeerTapado(filepath.Join(t.TempDir(), "no-existe.md")); err == nil {
		t.Fatal("un archivo que no existe tiene que dar error")
	}
}
