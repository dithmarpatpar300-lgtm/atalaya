package avisos

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config: qué tipos de aviso están encendidos. Se cambia desde la campana del panel y vive en avisos.json, junto a la
// base de la datapedia (%LOCALAPPDATA%\Atalaya).
type Config struct {
	Decisiones bool `json:"decisiones"`
	Alertas    bool `json:"alertas"`
}

// PorOmision: todo encendido.
var PorOmision = Config{Decisiones: true, Alertas: true}

// LeerConfig: la de ruta, o PorOmision si no existe o no se lee (un archivo roto no apaga los avisos).
func LeerConfig(ruta string) Config {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return PorOmision
	}
	c := PorOmision
	if json.Unmarshal(b, &c) != nil {
		return PorOmision
	}
	return c
}

// GuardarConfig escribe en un archivo aparte y lo cambia de nombre: nunca queda uno a medias.
func GuardarConfig(ruta string, c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}
