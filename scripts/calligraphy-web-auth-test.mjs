import fs from "node:fs";
import vm from "node:vm";
import assert from "node:assert/strict";
import { TextEncoder } from "node:util";
import { webcrypto } from "node:crypto";

class FakeElement {
  constructor(id = "") {
    this.id = id;
    this.value = "";
    this.textContent = "";
    this.innerHTML = "";
    this.className = "";
    this.hidden = false;
    this.disabled = false;
    this.href = "";
    this.download = "";
    this.dataset = {};
    this.attrs = new Map();
    this.children = [];
    this.listeners = new Map();
  }

  addEventListener(event, listener) {
    this.listeners.set(event, listener);
  }

  append(...items) {
    this.children.push(...items);
  }

  remove() {}

  click() {}

  getAttribute(name) {
    return this.attrs.get(name) || "";
  }

  setAttribute(name, value) {
    this.attrs.set(name, String(value));
  }

  removeAttribute(name) {
    this.attrs.delete(name);
  }

  querySelector() {
    return null;
  }

  querySelectorAll() {
    return [];
  }
}

const elementIds = [
  "authUsername",
  "authPassword",
  "authState",
  "authMode",
  "registerButton",
  "loginButton",
  "logoutButton",
  "ownerUserId",
  "ownerDisplay",
  "textInput",
  "styleInput",
  "formatInput",
  "widthInput",
  "heightInput",
  "marginInput",
  "signatureInput",
  "sealInput",
  "glyphInput",
  "glyphStyleInput",
  "previewButton",
  "saveButton",
  "exportButton",
  "refreshButton",
  "glyphSearchButton",
  "glyphDetail",
  "previewSurface",
  "status",
  "layoutMeta",
  "artworkTitle",
  "draftList",
  "glyphResults",
  "downloadLink",
  "exportList",
  "presetRefreshButton",
  "presetGroups",
  "learningRefreshButton",
  "learningStats",
  "favoriteList",
  "practiceList",
];

function makeStorage(initial = {}) {
  const values = new Map(Object.entries(initial));
  return {
    values,
    getItem: (key) => values.get(key) || "",
    setItem: (key, value) => values.set(key, String(value)),
    removeItem: (key) => values.delete(key),
  };
}

function makeElements() {
  const elements = new Map(elementIds.map((id) => [id, new FakeElement(id)]));
  elements.get("authUsername").value = "learner";
  elements.get("authPassword").value = "secret123";
  elements.get("textInput").value = "山水清音";
  elements.get("styleInput").value = "ou";
  elements.get("formatInput").value = "doufang";
  elements.get("widthInput").value = "69";
  elements.get("heightInput").value = "68";
  elements.get("marginInput").value = "3";
  elements.get("signatureInput").value = "试作";
  elements.get("sealInput").value = "1";
  elements.get("glyphInput").value = "山";
  elements.get("glyphStyleInput").value = "ou";
  return elements;
}

function jsonResponse(payload, ok = true) {
  return {
    ok,
    json: async () => payload,
  };
}

function makeContext({ runtimeConfig, locationSearch = "" }) {
  const elements = makeElements();
  const localStorage = makeStorage();
  const sessionStorage = makeStorage();
  const calls = [];
  const redirects = [];
  const historyReplacements = [];

  const context = {
    console,
    TextEncoder,
    crypto: webcrypto,
    btoa: (value) => Buffer.from(value, "binary").toString("base64"),
    Blob: class Blob {
      constructor(parts, opts) {
        this.parts = parts;
        this.type = opts?.type || "";
      }
    },
    URL,
    URLSearchParams,
    localStorage,
    sessionStorage,
    location: {
      origin: "https://calligraphy.example",
      pathname: "/",
      search: locationSearch,
      hash: "",
      assign: (url) => redirects.push(url),
    },
    history: {
      replaceState: (_state, _title, url) => historyReplacements.push(url),
    },
    document: {
      title: "Nebula Calligraphy",
      body: new FakeElement("body"),
      querySelector(selector) {
        if (!selector.startsWith("#")) return null;
        return elements.get(selector.slice(1)) || null;
      },
      createElement(tag) {
        return new FakeElement(tag);
      },
    },
    fetch: async (path, options = {}) => {
      calls.push({ path: String(path), options });
      if (path === "/api/v1/calligraphy/runtime-config") return jsonResponse(runtimeConfig);
      if (path === "/api/v1/calligraphy/layouts/preview") {
        return jsonResponse({
          paper: { width_cm: 69, height_cm: 68 },
          slots: [],
          signature_slots: [],
          seal_slots: [],
          character_count: 4,
          columns: 2,
          rows: 2,
          glyph_size_cm: 20,
        });
      }
      if (String(path).startsWith("/api/v1/calligraphy/glyphs/search")) return jsonResponse({ items: [] });
      if (String(path).startsWith("/api/v1/calligraphy/glyphs/presets")) return jsonResponse({ items: [] });
      if (path === "https://identity.example/api/v1/auth/login") {
        const body = JSON.parse(options.body);
        assert.deepEqual(body, { username: "learner", password: "secret123" });
        return jsonResponse({ data: { access_token: "identity-token" } });
      }
      if (path === "https://identity.example/api/v1/auth/token") {
        const body = new URLSearchParams(options.body);
        assert.equal(body.get("grant_type"), "authorization_code");
        assert.equal(body.get("client_id"), "nebula-calligraphy-web");
        assert.equal(body.get("code"), "auth-code");
        assert.equal(body.get("redirect_uri"), "https://calligraphy.example/");
        assert.ok(body.get("code_verifier"));
        return jsonResponse({ access_token: "oidc-token" });
      }
      if (path === "/api/v1/calligraphy/auth/me") {
        const authHeader = options.headers.Authorization;
        assert.ok(["Bearer identity-token", "Bearer oidc-token"].includes(authHeader));
        return jsonResponse({ user_id: "nebula-user-1", username: "nebula-user-1", created_at: "" });
      }
      if (String(path).startsWith("/api/v1/calligraphy/artworks/drafts")) return jsonResponse({ items: [] });
      if (String(path).startsWith("/api/v1/calligraphy/users/nebula-user-1/learning")) {
        return jsonResponse({ favorites: [], recent_practice: [], favorite_count: 0, practice_count: 0 });
      }
      throw new Error(`unexpected fetch ${path}`);
    },
  };
  context.globalThis = context;
  return { context, elements, localStorage, sessionStorage, calls, redirects, historyReplacements };
}

async function loadApp(env) {
  vm.createContext(env.context);
  vm.runInContext(fs.readFileSync("web/app/app.js", "utf8"), env.context, { filename: "web/app/app.js" });
  await new Promise((resolve) => setTimeout(resolve, 0));
  await new Promise((resolve) => setTimeout(resolve, 0));
}

const directEnv = makeContext({
  runtimeConfig: {
    runtime_profile: "managed",
    auth_mode: "nebula-direct",
    identity_login_endpoint: "https://identity.example/api/v1/auth/login",
  },
});
await loadApp(directEnv);
await directEnv.context.loginWithNebulaIdentity();

assert.equal(directEnv.sessionStorage.values.get("calligraphy.auth_token"), "identity-token");
assert.equal(directEnv.localStorage.values.has("calligraphy.auth_token"), false);
assert.equal(directEnv.elements.get("ownerUserId").value, "nebula-user-1");
assert.equal(directEnv.elements.get("authMode").textContent, "认证模式：Nebula Identity");
assert.equal(directEnv.elements.get("registerButton").hidden, true);
assert.equal(directEnv.elements.get("loginButton").textContent, "Identity 登录");

const pkceStartEnv = makeContext({
  runtimeConfig: {
    runtime_profile: "managed",
    auth_mode: "oidc-pkce",
    identity_client_id: "nebula-calligraphy-web",
    identity_authorization_endpoint: "https://identity.example/api/v1/auth/authorize",
    identity_token_endpoint: "https://identity.example/api/v1/auth/token",
  },
});
await loadApp(pkceStartEnv);
await pkceStartEnv.context.startOIDCLogin();

assert.equal(pkceStartEnv.redirects.length, 1);
const authURL = new URL(pkceStartEnv.redirects[0]);
assert.equal(authURL.origin + authURL.pathname, "https://identity.example/api/v1/auth/authorize");
assert.equal(authURL.searchParams.get("response_type"), "code");
assert.equal(authURL.searchParams.get("client_id"), "nebula-calligraphy-web");
assert.equal(authURL.searchParams.get("redirect_uri"), "https://calligraphy.example/");
assert.equal(authURL.searchParams.get("code_challenge_method"), "S256");
assert.ok(authURL.searchParams.get("code_challenge"));
assert.ok(pkceStartEnv.sessionStorage.values.get("calligraphy.pkce_verifier"));

const callbackState = "expected-state";
const pkceCallbackEnv = makeContext({
  runtimeConfig: {
    runtime_profile: "managed",
    auth_mode: "oidc-pkce",
    identity_client_id: "nebula-calligraphy-web",
    identity_authorization_endpoint: "https://identity.example/api/v1/auth/authorize",
    identity_token_endpoint: "https://identity.example/api/v1/auth/token",
  },
  locationSearch: `?code=auth-code&state=${callbackState}`,
});
pkceCallbackEnv.sessionStorage.setItem("calligraphy.oidc_state", callbackState);
pkceCallbackEnv.sessionStorage.setItem("calligraphy.pkce_verifier", "stored-verifier");
await loadApp(pkceCallbackEnv);

assert.equal(pkceCallbackEnv.sessionStorage.values.get("calligraphy.auth_token"), "oidc-token");
assert.equal(pkceCallbackEnv.elements.get("ownerUserId").value, "nebula-user-1");
assert.deepEqual(pkceCallbackEnv.historyReplacements, ["https://calligraphy.example/"]);
