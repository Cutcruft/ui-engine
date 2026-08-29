package module

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manifest — module.yaml
type Manifest struct {
	Module ModuleMeta `yaml:"module"`
}

type ModuleMeta struct {
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version"`
	Type        string            `yaml:"type"` // component | layout | wrapper | service
	Description string            `yaml:"description"`
	Author      string            `yaml:"author"`
	License     string            `yaml:"license"`
	Entry       Entry             `yaml:"entry"`
	Components  []ComponentDecl   `yaml:"components"`
	Depends     []string          `yaml:"depends"`
	Props       map[string]Prop   `yaml:"props"`
}

type Entry struct {
	Wasm string `yaml:"wasm"`
	JS   string `yaml:"js"`
	CSS  string `yaml:"css"`
}

type ComponentDecl struct {
	Name  string          `yaml:"name"`
	Props map[string]Prop `yaml:"props"`
}

type Prop struct {
	Type    string   `yaml:"type"`
	Default any      `yaml:"default"`
	Enum    []string `yaml:"enum"`
}

// LoadManifest loads module.yaml from dir
func LoadManifest(dir string) (*Manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "module.yaml"))
	if err != nil {
		return nil, fmt.Errorf("module.yaml not found in %s: %w", dir, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse module.yaml: %w", err)
	}
	if m.Module.Name == "" {
		return nil, fmt.Errorf("module name required")
	}
	if m.Module.Version == "" {
		m.Module.Version = "0.1.0"
	}
	return &m, nil
}

// Validate checks manifest
func (m *Manifest) Validate() error {
	if m.Module.Name == "" {
		return fmt.Errorf("name required")
	}
	// name should be kebab-case
	return nil
}
