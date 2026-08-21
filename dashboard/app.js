(() => {
  "use strict";

  let csrfToken = "";

  async function initialize() {
    const response = await fetch("/v1/session", { credentials: "same-origin", cache: "no-store" });
    if (!response.ok) throw new Error("session_unavailable");
    const session = await response.json();
    if (session.authenticated === false) {
      window.location.replace("/login.html");
      throw new Error("authentication_required");
    }
    if (typeof session.csrf_token !== "string" || session.csrf_token.length < 32) {
      throw new Error("session_invalid");
    }
    csrfToken = session.csrf_token;
    const workspace = document.querySelector("#workspace-name");
    if (workspace) workspace.textContent = session.workspace?.name || "Local MVP workspace";
    return session;
  }

  async function api(path, options = {}) {
    const headers = new Headers(options.headers || {});
    headers.set("X-ProofLayer-CSRF", csrfToken);
    const response = await fetch(path, { ...options, headers, credentials: "same-origin", cache: "no-store" });
    let document = null;
    if (response.status !== 204) {
      try { document = await response.json(); } catch { document = null; }
    }
    if (response.status === 401) {
      window.location.replace("/login.html");
      throw new Error("authentication_required");
    }
    if (!response.ok) {
      const error = new Error(document?.code || "request_failed");
      error.code = document?.code || "request_failed";
      error.status = response.status;
      throw error;
    }
    return document;
  }

  function idempotencyKey() {
    const bytes = crypto.getRandomValues(new Uint8Array(24));
    return btoa(String.fromCharCode(...bytes)).replaceAll("+", "-").replaceAll("/", "_").replaceAll("=", "");
  }

  function clear(element) { element.replaceChildren(); }
  function node(tag, text, className) {
    const element = document.createElement(tag);
    if (text !== undefined) element.textContent = text;
    if (className) element.className = className;
    return element;
  }

  window.ProofLayer = Object.freeze({ initialize, api, idempotencyKey, clear, node });
})();
