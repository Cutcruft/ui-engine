// Package theme переносит токены из theme.yaml в CSS-переменные, которые
// потребляют Web Components (Shoelace) и пользовательские стили.
package theme

import (
	"fmt"
	"strings"
)

// Resolver применяет активную тему.
type Resolver struct {
	// Tokens — набор css-переменных активной темы (имя -> значение).
	Tokens map[string]string
}

// New создаёт резолвер на основе токенов активной темы.
func New(tokens map[string]string) *Resolver {
	return &Resolver{Tokens: tokens}
}

// NewFromDesignTokens создаёт резолвер из дизайн-токенов (colors, spacing и т.д.)
func NewFromDesignTokens(tokens map[string]string, raw map[string]string) *Resolver {
	merged := map[string]string{}
	for k, v := range tokens {
		merged[k] = v
	}
	for k, v := range raw {
		merged[k] = v
	}
	return New(merged)
}

// ApplyCSS возвращает содержимое <style> с переменными темы + анимациями.
func (r *Resolver) ApplyCSS() string {
	var b strings.Builder
	b.WriteString(":root{")
	for k, v := range r.Tokens {
		name := k
		if !strings.HasPrefix(name, "--") {
			name = "--" + name
		}
		// поддержка nested токенов: colors.primary -> --colors-primary или --primary
		name = strings.ReplaceAll(name, ".", "-")
		fmt.Fprintf(&b, "%s:%s;", name, v)
	}
	b.WriteString("}")
	// анимации переключения темы
	b.WriteString(" *{transition: background-color 0.3s ease, color 0.3s ease, border-color 0.3s ease;}")
	return b.String()
}

// CSSVar возвращает имя css-переменной для токена.
func CSSVar(name string) string {
	if strings.HasPrefix(name, "--") {
		return name
	}
	name = strings.ReplaceAll(name, ".", "-")
	return "--" + name
}

// TokensToCSSVars конвертирует DesignTokens в плоский map css-переменных
func TokensToCSSVars(dt map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range dt {
		out[k] = v
	}
	return out
}
