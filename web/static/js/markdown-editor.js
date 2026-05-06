(function initMarkdownEditorBridge() {
  const instances = new Map();

  const getCtor = () => (typeof MarkdownEditor === "function" ? MarkdownEditor : null);

  const keyFor = (target) => {
    const el = typeof target === "string" ? document.querySelector(target) : target;
    if (!el) return "";
    return el.id || el.name || "";
  };

  const syncInputEvent = (el) => {
    if (!el) return;
    el.dispatchEvent(new Event("input", { bubbles: true }));
  };

  const attachEditorSync = (instance, el) => {
    if (!instance || !el) return;

    if (typeof instance.insertText === "function") {
      const originalInsertText = instance.insertText.bind(instance);
      instance.insertText = (...args) => {
        originalInsertText(...args);
        syncInputEvent(el);
      };
    }

    const undoRedo = instance.undoRedoManager;
    if (undoRedo) {
      if (typeof undoRedo.undo === "function") {
        const originalUndo = undoRedo.undo.bind(undoRedo);
        undoRedo.undo = () => {
          originalUndo();
          syncInputEvent(el);
        };
      }
      if (typeof undoRedo.redo === "function") {
        const originalRedo = undoRedo.redo.bind(undoRedo);
        undoRedo.redo = () => {
          originalRedo();
          syncInputEvent(el);
        };
      }
    }

    const wrapper = el.closest(".markdown-editor-wrapper");
    if (wrapper) {
      wrapper.addEventListener(
        "click",
        (event) => {
          if (event.target.closest(".markdown-btn, .fj\\:me-menu-item, [role='menuitem']")) {
            queueMicrotask(() => syncInputEvent(el));
          }
        },
        true,
      );
    }
  };

  window.getMarkdownTextareaValue = function getMarkdownTextareaValue(target) {
    const el = typeof target === "string" ? document.querySelector(target) : target;
    if (!el || el.tagName !== "TEXTAREA") return null;
    return el.value;
  };

  window.destroyMarkdownEditor = function destroyMarkdownEditor(key) {
    if (!key) return;
    const entry = instances.get(key);
    if (!entry) return;

    const ta = entry.instance?.usertextarea || entry.el;
    instances.delete(key);

    if (!ta) return;

    const wrapper = ta.closest(".markdown-editor-wrapper");
    if (wrapper && wrapper.parentNode) {
      wrapper.parentNode.insertBefore(ta, wrapper);
      wrapper.remove();
    }

    if (entry.originalClass != null) ta.className = entry.originalClass;
    if (entry.originalDisabled != null) ta.disabled = entry.originalDisabled;
  };

  window.destroyAllMarkdownEditors = function destroyAllMarkdownEditors() {
    for (const key of [...instances.keys()]) {
      window.destroyMarkdownEditor(key);
    }
  };

  window.notifyMarkdownEditor = function notifyMarkdownEditor(target) {
    const el = typeof target === "string" ? document.querySelector(target) : target;
    if (!el) return;
    el.dispatchEvent(new Event("input", { bubbles: true }));
    const key = keyFor(el);
    const entry = key ? instances.get(key) : null;
    if (entry?.instance && typeof entry.instance.render === "function") {
      entry.instance.render();
    }
  };

  window.initMarkdownEditor = function initMarkdownEditor(target, options = {}) {
    const el = typeof target === "string" ? document.querySelector(target) : target;
    if (!el || el.tagName !== "TEXTAREA") return null;

    const key = keyFor(el);
    if (!key) return null;

    if (!el.isConnected) return null;

    const Ctor = getCtor();
    if (!Ctor) return null;

    window.destroyMarkdownEditor(key);

    const originalClass = el.className;
    const originalDisabled = el.disabled;

    const instance = new Ctor(el, options);
    attachEditorSync(instance, el);
    instances.set(key, { instance, el, originalClass, originalDisabled });
    return instance;
  };

  window.isMarkdownEditorReady = function isMarkdownEditorReady() {
    return Boolean(getCtor());
  };
})();
