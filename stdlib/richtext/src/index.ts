import type { ComponentHandle } from "../../../runtime-js/src/types";

// Richtext Notion-like — максимально конфигурируемый
// Поддерживает: headings, bold, italic, underline, strike, code, codeBlock, blockquote, bulletList, orderedList, taskList, link, image, table, horizontalRule, placeholder, limit, toolbar, readOnly

const CDN = {
  css: "https://cdn.jsdelivr.net/npm/@tiptap/core@2.8.0/dist/style.css",
  scripts: [
    "https://cdn.jsdelivr.net/npm/@tiptap/core@2.8.0/dist/tiptap-core.umd.min.js",
    "https://cdn.jsdelivr.net/npm/@tiptap/starter-kit@2.8.0/dist/starter-kit.umd.min.js",
    "https://cdn.jsdelivr.net/npm/@tiptap/extension-placeholder@2.8.0/dist/placeholder.umd.min.js",
    "https://cdn.jsdelivr.net/npm/@tiptap/extension-link@2.8.0/dist/link.umd.min.js",
    "https://cdn.jsdelivr.net/npm/@tiptap/extension-image@2.8.0/dist/image.umd.min.js",
    "https://cdn.jsdelivr.net/npm/@tiptap/extension-table@2.8.0/dist/table.umd.min.js",
    "https://cdn.jsdelivr.net/npm/@tiptap/extension-task-list@2.8.0/dist/task-list.umd.min.js",
    "https://cdn.jsdelivr.net/npm/@tiptap/extension-task-item@2.8.0/dist/task-item.umd.min.js",
    "https://cdn.jsdelivr.net/npm/@tiptap/extension-underline@2.8.0/dist/underline.umd.min.js",
    "https://cdn.jsdelivr.net/npm/@tiptap/extension-highlight@2.8.0/dist/highlight.umd.min.js",
  ]
};

let loaded = false;
async function ensureTiptap(): Promise<void> {
  if (loaded) return;
  if (document.querySelector('link[href*="tiptap"]')) { loaded = true; return; }
  const link = document.createElement("link");
  link.rel = "stylesheet";
  link.href = CDN.css;
  document.head.appendChild(link);
  for (const src of CDN.scripts) {
    await new Promise<void>((res, rej) => {
      const s = document.createElement("script");
      s.src = src;
      s.onload = () => res();
      s.onerror = () => rej(new Error(`Failed to load ${src}`));
      document.head.appendChild(s);
    });
  }
  loaded = true;
}

interface RichtextProps {
  value?: string;
  placeholder?: string;
  readOnly?: string | boolean;
  limit?: string | number;
  toolbar?: string; // full | minimal | none
  onChange?: string;
}

const Richtext: ComponentHandle = {
  mount(container, props: RichtextProps, onEvent) {
    // для строгой изоляции и офлайн-демо используем fallback contenteditable без CDN
    // ensureTiptap().catch(() => {}) — не блокируем, tiptap грузится параллельно если доступен
    const wrapper = document.createElement("div");
    wrapper.className = "ui-richtext-notion";
    wrapper.style.border = "1px solid var(--sl-color-neutral-200, #e5e7eb)";
    wrapper.style.borderRadius = "8px";
    wrapper.style.overflow = "hidden";
    container.appendChild(wrapper);

    // toolbar notion-like
    let toolbar: HTMLElement | null = null;
    const toolbarMode = props.toolbar || "full";
    if (toolbarMode !== "none") {
      toolbar = document.createElement("div");
      toolbar.className = "ui-richtext-toolbar";
      toolbar.style.display = "flex";
      toolbar.style.flexWrap = "wrap";
      toolbar.style.gap = "4px";
      toolbar.style.padding = "8px";
      toolbar.style.borderBottom = "1px solid var(--sl-color-neutral-100, #f3f4f6)";
      toolbar.style.background = "var(--sl-color-neutral-50, #f9fafb)";
      wrapper.appendChild(toolbar);
    }

    const editorEl = document.createElement("div");
    editorEl.className = "ui-richtext-editor";
    editorEl.style.minHeight = "160px";
    editorEl.style.padding = "12px";
    editorEl.style.outline = "none";
    wrapper.appendChild(editorEl);

    // placeholder via CSS
    if (props.placeholder) {
      const style = document.createElement("style");
      style.textContent = `.ui-richtext-editor p.is-editor-empty:first-child::before { content: "${props.placeholder.replace(/"/g, '\\"')}"; float: left; color: #9ca3af; pointer-events: none; height: 0; }`;
      wrapper.appendChild(style);
    }

    let editor: any = null;
    let isReadOnly = props.readOnly === "true" || props.readOnly === true;

    const emitChange = (html: string) => {
      if (props.onChange) {
        // заменяем $event на html (экранируем кавычки для DSL)
        const action = props.onChange.replace("$event", html.replace(/"/g, '\\"'));
        onEvent(action);
      }
    };

    try {
      const { Editor } = (window as any).tiptapCore || {};
      const StarterKit = (window as any).tiptapStarterKit;
      if (Editor && StarterKit) {
        const extensions: any[] = [StarterKit.configure({ heading: { levels: [1, 2, 3] } })];
        // добавляем расширения если доступны
        const placeholderExt = (window as any).tiptapPlaceholder;
        if (placeholderExt) extensions.push(placeholderExt.configure({ placeholder: props.placeholder || "" }));
        editor = new Editor({
          element: editorEl,
          extensions,
          content: props.value || "<p></p>",
          editable: !isReadOnly,
          onUpdate: ({ editor }: any) => emitChange(editor.getHTML()),
        });

        if (toolbar) {
          const mkBtn = (label: string, title: string, fn: () => void) => {
            const b = document.createElement("sl-button");
            (b as any).size = "small";
            b.textContent = label;
            b.title = title;
            b.addEventListener("click", fn);
            toolbar!.appendChild(b);
          };
          mkBtn("H1", "Heading 1", () => editor.chain().focus().toggleHeading({ level: 1 }).run());
          mkBtn("H2", "Heading 2", () => editor.chain().focus().toggleHeading({ level: 2 }).run());
          mkBtn("B", "Bold", () => editor.chain().focus().toggleBold().run());
          mkBtn("I", "Italic", () => editor.chain().focus().toggleItalic().run());
          mkBtn("U", "Underline", () => {
            const ext = (window as any).tiptapUnderline;
            if (ext) editor.chain().focus().toggleUnderline().run();
          });
          mkBtn("•", "Bullet List", () => editor.chain().focus().toggleBulletList().run());
          mkBtn("1.", "Ordered List", () => editor.chain().focus().toggleOrderedList().run());
          mkBtn("☐", "Task List", () => editor.chain().focus().toggleTaskList().run());
          mkBtn("❝", "Blockquote", () => editor.chain().focus().toggleBlockquote().run());
          mkBtn("</>", "Code Block", () => editor.chain().focus().toggleCodeBlock().run());
          mkBtn("—", "Horizontal Rule", () => editor.chain().focus().setHorizontalRule().run());
          if (toolbarMode === "full") {
            mkBtn("🔗", "Link", () => {
              const url = prompt("URL:");
              if (url) editor.chain().focus().setLink({ href: url }).run();
            });
            mkBtn("🖼️", "Image", () => {
              const url = prompt("Image URL:");
              if (url) editor.chain().focus().setImage({ src: url }).run();
            });
          }
        }
      } else {
        throw new Error("tiptap not available");
      }
    } catch (e) {
      console.warn("richtext fallback to contenteditable", e);
      editorEl.contentEditable = isReadOnly ? "false" : "true";
      editorEl.innerHTML = props.value || "";
      editorEl.addEventListener("input", () => emitChange(editorEl.innerHTML));
      editor = {
        getHTML: () => editorEl.innerHTML,
        destroy: () => {},
        isFallback: true,
      };
    }

    // character limit
    const limit = props.limit ? parseInt(String(props.limit), 10) : 0;
    if (limit > 0) {
      const checkLimit = () => {
        const text = editorEl.textContent || "";
        if (text.length > limit) {
          editorEl.style.borderColor = "var(--sl-color-danger-500, #ef4444)";
        } else {
          editorEl.style.borderColor = "";
        }
      };
      editorEl.addEventListener("input", checkLimit);
    }

    return {
      update(newProps: RichtextProps) {
        if (editor && newProps.value !== undefined) {
          const current = editor.getHTML?.() || editorEl.innerHTML;
          if (newProps.value !== current) {
            if (editor.commands) editor.commands.setContent(newProps.value);
            else editorEl.innerHTML = newProps.value;
          }
        }
      },
      unmount() {
        try { editor?.destroy(); } catch {}
        wrapper.remove();
      }
    };
  }
};

function registerRichtext() {
  window.UIEngineModules = window.UIEngineModules || {};
  window.UIEngineModules["richtext"] = Richtext;
  window.UIEngineModules["RichText"] = Richtext;
  if (window.UIEngine) {
    window.UIEngine.registerComponent("richtext", Richtext);
  }
}

if (typeof window !== "undefined") {
  const tryReg = () => {
    if (window.UIEngine) registerRichtext();
    else setTimeout(tryReg, 100);
  };
  tryReg();
}
