//go:build js && wasm

// Package main — точка входа WASM-ядра в браузере.
//
// Ожидает, что bootstrap (runtime-js) положил YAML-конфиги в
// window.__UI_CONFIG__ как { app: string, theme: string, screens: { name: string } }.
package main

import (
	"fmt"
	"syscall/js"

	"github.com/ui-engine/core/config"
	"github.com/ui-engine/core/dom"
	"github.com/ui-engine/core/net"
	"github.com/ui-engine/core/runtime"
	"github.com/ui-engine/core/state"
	"github.com/ui-engine/core/theme"
	"gopkg.in/yaml.v3"
)

func main() {
	// Конфиги уже загружены в window.__UI_CONFIG__ до go.run (см. app.js).
	boot()

	// Держим рантайм живым (go.run в JS не завершается, пока канал открыт).
	c := make(chan struct{}, 0)
	<-c
}

func boot() {
	cfgJS := js.Global().Get("__UI_CONFIG__")
	if !cfgJS.Truthy() {
		println("ui-engine: __UI_CONFIG__ not found")
		return
	}

	// app.yaml
	var app config.App
	if err := yaml.Unmarshal([]byte(cfgJS.Get("app").String()), &app); err != nil {
		println("ui-engine: app parse:", err.Error())
		return
	}

	// theme.yaml
	var th config.Theme
	if s := cfgJS.Get("theme").String(); s != "" {
		if err := yaml.Unmarshal([]byte(s), &th); err != nil {
			println("ui-engine: theme parse:", err.Error())
		}
	} else {
		th.Themes = map[string]map[string]string{}
	}
	active := th.Active
	if active == "" {
		active = app.ThemeActive
	}

	// screens
	screens := map[string]*config.Screen{}
	scJS := cfgJS.Get("screens")
	if scJS.Truthy() {
		keys := js.Global().Get("Object").Call("keys", scJS)
		for i := 0; i < keys.Length(); i++ {
			name := keys.Index(i).String()
			var s config.Screen
			if err := yaml.Unmarshal([]byte(scJS.Get(name).String()), &s); err != nil {
				println("ui-engine: screen", name, "parse:", err.Error())
				continue
			}
			s.Name = name
			screens[name] = &s
		}
	}

	// состояние
	st := state.New()
	seedInitial(st)
	// default-значения из state.yaml
	if raw := cfgJS.Get("state").String(); raw != "" {
		var stCfg config.State
		if err := yaml.Unmarshal([]byte(raw), &stCfg); err != nil {
			println("ui-engine: state parse:", err.Error())
		} else {
			applyStateDefaults(st, "state", stCfg.Vars)
		}
	}

	// theme css — поддержка дизайн-токенов и кастомных тем
	themeTokens := map[string]string{}
	// базовые токены
	for k, v := range th.Tokens.Colors {
		themeTokens["colors-"+k] = v
	}
	for k, v := range th.Tokens.Spacing {
		themeTokens["spacing-"+k] = v
	}
	for k, v := range th.Tokens.Radius {
		themeTokens["radius-"+k] = v
	}
	for k, v := range th.Tokens.Typography {
		themeTokens["typography-"+k] = v
	}
	for k, v := range th.Tokens.Shadows {
		themeTokens["shadows-"+k] = v
	}
	for k, v := range th.Tokens.Animations {
		themeTokens["animations-"+k] = v
	}
	// активная тема (поддержка как плоских, так и вложенных)
	if raw, ok := th.RawThemes[active]; ok {
		for k, v := range raw.Colors {
			themeTokens["colors-"+k] = v
		}
		for k, v := range raw.Animations {
			themeTokens["animations-"+k] = v
		}
	}
	// плоские темы (обратная совместимость)
	if flat, ok := th.Themes[active]; ok {
		for k, v := range flat {
			themeTokens[k] = v
		}
	}
	tr := theme.New(themeTokens)
	injectThemeStyle(tr.ApplyCSS())
	// реактивное переключение темы через state.ui.theme
	st.Subscribe("state.ui.theme", func() {
		newTheme := st.GetString("state.ui.theme")
		if newTheme == "" {
			newTheme = th.Active
		}
		newTokens := map[string]string{}
		for k, v := range themeTokens {
			newTokens[k] = v
		}
		if raw, ok := th.RawThemes[newTheme]; ok {
			for k, v := range raw.Colors {
				newTokens["colors-"+k] = v
			}
		}
		if flat, ok := th.Themes[newTheme]; ok {
			for k, v := range flat {
				newTokens[k] = v
			}
		}
		tr2 := theme.New(newTokens)
		injectThemeStyle(tr2.ApplyCSS())
	})

	// engine
	eng := runtime.NewEngine(&app, screens, st)
	registerActions(eng, st)

	// net: контракт из net.yaml
	var netCfg *config.Net
	if raw := cfgJS.Get("net").String(); raw != "" {
		var n config.Net
		if err := yaml.Unmarshal([]byte(raw), &n); err != nil {
			println("ui-engine: net parse:", err.Error())
		} else {
			netCfg = &n
		}
	}

	// hooks из hooks.yaml
	var hooksCfg *config.Hooks
	if raw := cfgJS.Get("hooks").String(); raw != "" {
		var h config.Hooks
		if err := yaml.Unmarshal([]byte(raw), &h); err != nil {
			println("ui-engine: hooks parse:", err.Error())
		} else {
			hooksCfg = &h
		}
	}

	// keys из keys.yaml
	var keysCfg *config.Keys
	if raw := cfgJS.Get("keys").String(); raw != "" {
		var k config.Keys
		if err := yaml.Unmarshal([]byte(raw), &k); err != nil {
			println("ui-engine: keys parse:", err.Error())
		} else {
			keysCfg = &k
		}
	}

	eng.Configure(netCfg, nil, keysCfg, hooksCfg)

	// WS-клиент
	var client *net.Client
	if netCfg != nil && netCfg.URL != "" {
		client = net.New(netCfg, st, eng, net.NewJSTransport())
		eng.BindNet(client)
		client.Connect()
	}

	// рендерер
	containerID := app.Entry
	if containerID == "" {
		containerID = "root"
	}
	rend, err := dom.NewRenderer(containerID, func(key string) {
		eng.Dispatch(key)
	})
	if err != nil {
		println("ui-engine: renderer:", err.Error())
		return
	}

	eng.SetRenderer(rend)
	eng.Mount()

	// Реактивность: любое изменение состояния -> diff-рендер текущего экрана.
	st.Subscribe("state", func() {
		eng.Apply()
	})

	// Хуки жизненного цикла.
	if hooksCfg != nil {
		eng.RunHooks(hooksCfg.OnReady)
	}
	runtime.BindKeys(eng)

	// JS Bridge — window.UIEngine (расширяем существующий от runtime-js, не перезаписываем)
	var uiObj js.Value
	if existing := js.Global().Get("UIEngine"); existing.Truthy() {
		uiObj = existing
	} else {
		uiObj = js.Global().Get("Object").New()
		js.Global().Set("UIEngine", uiObj)
	}
	dispatchFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			eng.Dispatch(args[0].String())
		}
		return nil
	})
	getStateFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			path := args[0].String()
			if v, ok := st.Get(path); ok {
				switch t := v.(type) {
				case string:
					return js.ValueOf(t)
				case bool:
					return js.ValueOf(t)
				case int:
					return js.ValueOf(t)
				case int64:
					return js.ValueOf(int(t))
				case float64:
					return js.ValueOf(t)
				default:
					return js.ValueOf(fmt.Sprintf("%v", v))
				}
			}
		}
		return js.Null()
	})
	setStateFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 2 {
			path := args[0].String()
			val := args[1]
			var goVal any
			switch val.Type() {
			case js.TypeString:
				goVal = val.String()
			case js.TypeNumber:
				goVal = val.Float()
			case js.TypeBoolean:
				goVal = val.Bool()
			default:
				goVal = val.String()
			}
			st.Set(path, goVal)
		}
		return nil
	})
	registerFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) >= 2 {
			key := args[0].String()
			fn := args[1]
			eng.RegisterAction("action."+key, func() {
				fn.Invoke()
			})
		}
		return nil
	})
	// сохраняем существующие методы от runtime-js (registerComponent/getComponent)
	if !uiObj.Get("dispatch").Truthy() {
		uiObj.Set("dispatch", dispatchFn)
	} else {
		// оборачиваем, чтобы сохранить оба
		uiObj.Set("dispatch", dispatchFn)
	}
	uiObj.Set("getState", getStateFn)
	uiObj.Set("setState", setStateFn)
	uiObj.Set("registerAction", registerFn)
	// registerComponent/getComponent уже есть от runtime-js, не перезаписываем
	if !uiObj.Get("registerComponent").Truthy() {
		// fallback если runtime-js не загрузился
		uiObj.Set("registerComponent", js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) >= 2 {
				name := args[0].String()
				handle := args[1]
				js.Global().Get("UIEngineModules").Set(name, handle)
			}
			return nil
		}))
	}
	if !uiObj.Get("getComponent").Truthy() {
		uiObj.Set("getComponent", js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				name := args[0].String()
				if mod := js.Global().Get("UIEngineModules").Get(name); mod.Truthy() {
					return mod
				}
			}
			return js.Null()
		}))
	}
	// делаем доступным для TS-мостов
	js.Global().Set("__goDispatch", dispatchFn)
	js.Global().Set("__goGetState", getStateFn)
	js.Global().Set("__goSetState", setStateFn)
	js.Global().Set("__goRegisterAction", registerFn)

	// держим ссылки чтобы не GC
	keep = append(keep, rend, eng, client, dispatchFn, getStateFn, setStateFn, registerFn, uiObj)

	println("ui-engine: booted")
}

func injectThemeStyle(css string) {
	if css == "" {
		return
	}
	doc := js.Global().Get("document")
	style := doc.Call("createElement", "style")
	style.Set("textContent", css)
	doc.Get("head").Call("appendChild", style)
}

var keep []any

// applyStateDefaults рекурсивно применяет default-значения из state.yaml в Store.
// path — префикс в состоянии (например "state"), vars — карта переменных контракта.
func applyStateDefaults(st *state.Store, path string, vars map[string]config.Variable) {
	for name, v := range vars {
		key := name
		if path != "" {
			key = path + "." + name
		}
		if len(v.Fields) > 0 {
			applyStateDefaults(st, key, v.Fields)
			continue
		}
		if v.Default != nil {
			st.Set(key, v.Default)
		}
	}
}

// seedInitial закладывает стартовые значения состояния (демо).
func seedInitial(st *state.Store) {
	st.Set("state.ui.loading", false)
	st.Set("state.ui.theme", "light")
}

// registerActions — базовые действия по умолчанию (могут быть расширены из YAML).
func registerActions(eng *runtime.Engine, st *state.Store) {
	eng.RegisterAction("action.toggleTheme", func() {
		cur := st.GetString("state.ui.theme")
		if cur == "light" {
			st.Set("state.ui.theme", "dark")
		} else {
			st.Set("state.ui.theme", "light")
		}
	})
}
