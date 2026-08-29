package cmd

import (
	"fmt"
	"os"

	"github.com/ui-engine/cli/internal/module"
)

// Module handles `ui-engine module ...`
func Module(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("использование: ui-engine module <new|add|list|remove> [args]")
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "new":
		if len(rest) < 1 {
			return fmt.Errorf("использование: ui-engine module new <name> [--type component|layout|wrapper|service] [--dir ./path] [--ts] [--with-tests] [--with-docs] [--with-example]")
		}
		name := rest[0]
		typeName := "component"
		dir := "."
		opts := module.ScaffoldOpts{Type: "component"}
		for i := 1; i < len(rest); i++ {
			switch rest[i] {
			case "--type":
				if i+1 < len(rest) {
					typeName = rest[i+1]
					opts.Type = typeName
					i++
				}
			case "--dir":
				if i+1 < len(rest) {
					dir = rest[i+1]
					i++
				}
			case "--ts":
				opts.TS = true
			case "--with-tests":
				opts.WithTests = true
			case "--with-docs":
				opts.WithDocs = true
			case "--with-example":
				opts.WithExample = true
			}
		}
		if opts.Type == "" {
			opts.Type = typeName
		}
		return module.ScaffoldWithOpts(dir, name, opts)
	case "add":
		if len(rest) < 1 {
			return fmt.Errorf("использование: ui-engine module add <name|path> [--local]")
		}
		source := rest[0]
		// handle --local flag
		for _, a := range rest[1:] {
			if a == "--local" {
				// source is already local path
			}
		}
		projectRoot := "."
		if len(rest) > 1 && !isFlag(rest[1]) {
			// second arg could be project dir
			if _, err := os.Stat(rest[1]); err == nil {
				// check if it's a dir
			}
		}
		return module.Add(projectRoot, source)
	case "list":
		projectRoot := "."
		if len(rest) > 0 && !isFlag(rest[0]) {
			projectRoot = rest[0]
		}
		return module.List(projectRoot)
	case "remove":
		if len(rest) < 1 {
			return fmt.Errorf("использование: ui-engine module remove <name>")
		}
		return module.Remove(".", rest[0])
	default:
		return fmt.Errorf("неизвестная подкоманда module %s (new|add|list|remove)", sub)
	}
}

func isFlag(s string) bool { return len(s) > 0 && s[0] == '-' }
