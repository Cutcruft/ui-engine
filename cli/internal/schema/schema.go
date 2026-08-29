// Package schema содержит JSON-Schema схемы конфигов ui-engine
// и функцию их валидации (используется в ui-engine lint).
package schema

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

//go:embed schemas/*.schema.json
var schemasFS embed.FS

// Validate проверяет YAML-документ (data) против схемы по имени
// (например "app", "screen", "theme", "net", "hooks", "keys").
func Validate(name string, data []byte) error {
	// YAML -> any (JSON-типы) через roundtrip.
	var raw any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	js, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("encode %s: %w", name, err)
	}
	var doc any
	if err := json.Unmarshal(js, &doc); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}

	compiler := jsonschema.NewCompiler()

	// Загружаем все схемы, чтобы $ref внутри разрешались.
	fs.WalkDir(schemasFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".schema.json") {
			return nil
		}
		raw, err := fs.ReadFile(schemasFS, path)
		if err != nil {
			return nil
		}
		_ = compiler.AddResource("ui-engine/"+strings.TrimPrefix(path, "schemas/"), bytes.NewReader(raw))
		return nil
	})

	sch, err := compiler.Compile("ui-engine/" + name + ".schema.json")
	if err != nil {
		return fmt.Errorf("load schema %s: %w", name, err)
	}
	if err := sch.Validate(doc); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
