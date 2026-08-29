package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoIdent(t *testing.T) {
	cases := map[string]string{
		"auth.login":  "AuthLogin",
		"items.list":  "ItemsList",
		"per_page":    "PerPage",
		"my-special":  "MySpecial",
		"ui.drawer_open": "UiDrawerOpen",
		"id":          "ID",
		"url":         "URL",
		"1login":      "V1Login",
		"state.counter.count": "StateCounterCount",
	}
	for in, want := range cases {
		if got := goIdent(in); got != want {
			t.Errorf("goIdent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGen(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("app.yaml", "name: gen-app\nroot: main\n")
	write("net.yaml", `
url: wss://example.com/ws
events:
  client:
    auth.login:
      request:
        login: {type: string, required: true}
        password: {type: string, required: true}
  server:
    auth.result:
      payload:
        id: {type: string, required: true}
        token: {type: string, secret: true}
`)
	write("state.yaml", `
state:
  session:
    type: object
    fields:
      token:
        type: string
        secret: true
      login:
        type: string
`)

	for i := 0; i < 2; i++ { // второй прогон должен дать идентичный результат
		if err := Gen(dir); err != nil {
			t.Fatalf("Gen run %d: %v", i+1, err)
		}
	}

	models, err := os.ReadFile(filepath.Join(dir, "generated", "net_models.go"))
	if err != nil {
		t.Fatalf("net_models.go: %v", err)
	}
	modelsStr := string(models)
	for _, want := range []string{
		"type AuthLoginRequest struct",
		"type AuthResult struct",
		`ID    string`, // идиома ID
		`json:"-"`,     // secret-поле не маршалится
	} {
		if !strings.Contains(modelsStr, want) {
			t.Errorf("net_models.go missing %q", want)
		}
	}

	stateModels, err := os.ReadFile(filepath.Join(dir, "generated", "state_models.go"))
	if err != nil {
		t.Fatalf("state_models.go: %v", err)
	}
	for _, want := range []string{"type Session struct", `Token string`, `json:"-"`} {
		if !strings.Contains(string(stateModels), want) {
			t.Errorf("state_models.go missing %q", want)
		}
	}

	send, err := os.ReadFile(filepath.Join(dir, "generated", "net_send.go"))
	if err != nil {
		t.Fatalf("net_send.go: %v", err)
	}
	for _, want := range []string{
		"func SendAuthLoginPayload", "func SendAuthLogin", `c.Send("auth.login"`,
	} {
		if !strings.Contains(string(send), want) {
			t.Errorf("net_send.go missing %q", want)
		}
	}
}

func TestGenDeterministic(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "app.yaml"), []byte("name: d\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "state.yaml"), []byte("state:\n  a:\n    type: object\n    fields:\n      b: {type: int}\n      c: {type: string}\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "net.yaml"), []byte("url: wss://x/y\nevents:\n  server:\n    z.res:\n      payload:\n        n: {type: int}\n"), 0o644)

	run := func() string {
		if err := Gen(dir); err != nil {
			t.Fatalf("Gen: %v", err)
		}
		var out strings.Builder
		for _, f := range []string{"net_models.go", "state_models.go", "net_send.go"} {
			b, _ := os.ReadFile(filepath.Join(dir, "generated", f))
			out.Write(b)
		}
		return out.String()
	}

	first := run()
	second := run()
	if first != second {
		t.Fatal("gen output differs between runs (not deterministic)")
	}
}