package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// devBroker рассылает события перезагрузки браузерам через SSE.
type devBroker struct {
	mu      sync.Mutex
	clients map[chan string]struct{}
	version int
}

func (b *devBroker) add() chan string {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan string, 4)
	b.clients[ch] = struct{}{}
	return ch
}

func (b *devBroker) remove(ch chan string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, ch)
}

func (b *devBroker) broadcast(event string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.version++
	for ch := range b.clients {
		select {
		case ch <- event:
		default:
		}
	}
}

// Dev запускает dev-сервер с hot-reload.
func Dev(root string) error {
	ls, err := newLayout(root)
	if err != nil {
		return err
	}
	dir := ls.Root

	// Первичная сборка.
	if err := Build(dir); err != nil {
		return err
	}

	addr := envOr("UI_ADDR", "127.0.0.1:8033")
	broker := &devBroker{clients: map[chan string]struct{}{}}

	// Наблюдатель.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	watchDirs := []string{
		dir,
		filepath.Join(dir, "screens"),
		filepath.Join(dir, "src", "go"),
		filepath.Join(dir, "src", "js"),
		filepath.Join(dir, "theme.yaml"),
	}
	_ = watchDirs
	addWatchRecursive(watcher, dir)

	// goroutine hot-reload.
	go func() {
		for {
			select {
			case ev, ok := <-watcher.Events:
				if !ok {
					return
				}
				if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
					continue
				}
				ext := strings.ToLower(filepath.Ext(ev.Name))
				if ext == ".yaml" || ext == ".yml" {
					log.Printf("YAML изменён: %s -> reload", filepath.Base(ev.Name))
					broker.broadcast("reload-config")
					continue
				}
				if strings.Contains(ev.Name, string(filepath.Separator)+"src") || strings.HasSuffix(ev.Name, ".go") {
					log.Printf("код изменён: %s -> пересборка wasm", filepath.Base(ev.Name))
					if err := Build(dir); err != nil {
						log.Printf("ошибка пересборки: %v", err)
						continue
					}
					broker.broadcast("reload-config")
				}
			case err, ok := <-watcher.Errors:
				if ok {
					log.Printf("watch error: %v", err)
				}
			}
		}
	}()

	// HTTP-сервер.
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Для index.html внедряем dev-инъекцию SSE-reload.
		path := r.URL.Path
		if path == "/" || path == "/index.html" {
			data, err := os.ReadFile(filepath.Join(ls.Build, "index.html"))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			inject := `<script src="/dev-inject.js"></script>`
			html := strings.Replace(string(data), "</body>", inject+"</body>", 1)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, html)
			return
		}
		http.FileServer(http.Dir(ls.Build)).ServeHTTP(w, r)
	})
	mux.HandleFunc("/__config__", func(w http.ResponseWriter, r *http.Request) {
		serveConfig(w, ls)
	})
	mux.HandleFunc("/__dev-events__", func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		ch := broker.add()
		defer broker.remove(ch)
		// начальное событие
		fmt.Fprintf(w, "event: hello\ndata: connected\n\n")
		fl.Flush()
		for ev := range ch {
			fmt.Fprintf(w, "event: reload\ndata: %s\n\n", ev)
			fl.Flush()
		}
	})
	// Инъекция dev-скрипта в index.html (reload по SSE).
	mux.HandleFunc("/dev-inject.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, devInjectJS)
	})

	log.Printf("dev-сервер: http://%s", addr)
	return http.ListenAndServe(addr, mux)
}

func addWatchRecursive(w *fsnotify.Watcher, root string) {
	_ = w.Add(root)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() {
			if strings.Contains(path, ".git") || strings.Contains(path, "build") {
				return filepath.SkipDir
			}
			w.Add(path)
		}
		return nil
	})
}

const devInjectJS = `
(() => {
  const es = new EventSource('/__dev-events__');
  es.addEventListener('reload', () => {
    location.reload();
  });
  es.addEventListener('error', () => {
    // сервер перезапустился - попробуем переподключиться
  });
})();
`
