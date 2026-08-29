module github.com/ui-engine/cli

go 1.24

require (
	github.com/fsnotify/fsnotify v1.8.0
	github.com/ui-engine/core v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	golang.org/x/sys v0.13.0 // indirect
)

replace github.com/ui-engine/core => ../core
