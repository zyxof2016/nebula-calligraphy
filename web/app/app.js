const state = {
  layout: null,
  draft: null,
  svg: "",
  practiceSvg: "",
  glyphDetail: null,
  practiceTemplate: null,
  learningProfile: null,
};

const els = {
  ownerUserId: document.querySelector("#ownerUserId"),
  textInput: document.querySelector("#textInput"),
  styleInput: document.querySelector("#styleInput"),
  formatInput: document.querySelector("#formatInput"),
  widthInput: document.querySelector("#widthInput"),
  heightInput: document.querySelector("#heightInput"),
  marginInput: document.querySelector("#marginInput"),
  signatureInput: document.querySelector("#signatureInput"),
  sealInput: document.querySelector("#sealInput"),
  glyphInput: document.querySelector("#glyphInput"),
  glyphStyleInput: document.querySelector("#glyphStyleInput"),
  previewButton: document.querySelector("#previewButton"),
  saveButton: document.querySelector("#saveButton"),
  exportButton: document.querySelector("#exportButton"),
  refreshButton: document.querySelector("#refreshButton"),
  glyphSearchButton: document.querySelector("#glyphSearchButton"),
  glyphDetail: document.querySelector("#glyphDetail"),
  previewSurface: document.querySelector("#previewSurface"),
  status: document.querySelector("#status"),
  layoutMeta: document.querySelector("#layoutMeta"),
  artworkTitle: document.querySelector("#artworkTitle"),
  draftList: document.querySelector("#draftList"),
  glyphResults: document.querySelector("#glyphResults"),
  downloadLink: document.querySelector("#downloadLink"),
  exportList: document.querySelector("#exportList"),
  presetRefreshButton: document.querySelector("#presetRefreshButton"),
  presetGroups: document.querySelector("#presetGroups"),
  learningRefreshButton: document.querySelector("#learningRefreshButton"),
  learningStats: document.querySelector("#learningStats"),
  favoriteList: document.querySelector("#favoriteList"),
  practiceList: document.querySelector("#practiceList"),
};

function layoutRequest() {
  return {
    text: els.textInput.value,
    style: els.styleInput.value,
    paper: {
      format: els.formatInput.value,
      width_cm: Number(els.widthInput.value),
      height_cm: Number(els.heightInput.value),
    },
    direction: "vertical_rtl",
    margin_cm: Number(els.marginInput.value),
    signature: { text: els.signatureInput.value },
    seal_count: Number(els.sealInput.value),
  };
}

function validateForm() {
  if (!els.ownerUserId.value.trim()) {
    throw new Error("请输入用户");
  }
  if (!els.textInput.value.trim()) {
    throw new Error("请输入正文");
  }
  for (const [label, el] of [
    ["宽度", els.widthInput],
    ["高度", els.heightInput],
    ["边距", els.marginInput],
  ]) {
    const value = Number(el.value);
    if (!Number.isFinite(value) || value <= 0) {
      throw new Error(`${label}必须大于 0`);
    }
  }
  if (Number(els.marginInput.value) * 2 >= Math.min(Number(els.widthInput.value), Number(els.heightInput.value))) {
    throw new Error("边距过大");
  }
}

async function api(path, options = {}) {
  const response = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  const payload = await response.json();
  if (!response.ok) {
    throw new Error(payload.message || "request failed");
  }
  return payload;
}

function setStatus(text, type = "") {
  els.status.textContent = text;
  els.status.className = type ? `status ${type}` : "status";
}

function renderLayout(layout) {
  state.layout = layout;
  const width = layout.paper.width_cm * 10;
  const height = layout.paper.height_cm * 10;
  const glyphs = layout.slots
    .map((slot) => `<text x="${slot.x_cm * 10}" y="${slot.y_cm * 10}" font-size="${slot.size_cm * 10}">${escapeHtml(slot.character)}</text>`)
    .join("");
  const signature = (layout.signature_slots || [])
    .map((slot) => `<text x="${slot.x_cm * 10}" y="${slot.y_cm * 10}" font-size="${slot.size_cm * 10}">${escapeHtml(slot.text)}</text>`)
    .join("");
  const seals = (layout.seal_slots || [])
    .map((slot) => {
      const size = slot.size_cm * 10;
      return `<rect x="${slot.x_cm * 10 - size / 2}" y="${slot.y_cm * 10 - size / 2}" width="${size}" height="${size}"></rect>`;
    })
    .join("");

  state.svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${layout.paper.width_cm * 10}mm" height="${layout.paper.height_cm * 10}mm" viewBox="0 0 ${width} ${height}">
    <rect width="100%" height="100%" fill="#fbf7ef"></rect>
    <g fill="#111" text-anchor="middle" dominant-baseline="central" font-family="serif">${glyphs}</g>
    <g fill="#333" text-anchor="middle" dominant-baseline="central" font-family="serif">${signature}</g>
    <g fill="none" stroke="#9f1d20" stroke-width="1.5">${seals}</g>
  </svg>`;

  els.previewSurface.innerHTML = state.svg;
  els.layoutMeta.textContent = `${layout.character_count} 字 · ${layout.columns} 列 x ${layout.rows} 行 · 字径 ${layout.glyph_size_cm}cm`;
  updateDownload(state.svg);
}

function updateDownload(svg) {
  if (!svg) {
    els.downloadLink.hidden = true;
    els.downloadLink.removeAttribute("href");
    return;
  }
  const blob = new Blob([svg], { type: "image/svg+xml" });
  const url = URL.createObjectURL(blob);
  const oldUrl = els.downloadLink.getAttribute("href");
  if (oldUrl) URL.revokeObjectURL(oldUrl);
  els.downloadLink.href = url;
  els.downloadLink.hidden = false;
}

function updateDownloadURL(url) {
  const oldUrl = els.downloadLink.getAttribute("href");
  if (oldUrl && oldUrl.startsWith("blob:")) URL.revokeObjectURL(oldUrl);
  els.downloadLink.href = url;
  els.downloadLink.hidden = false;
}

async function preview() {
  validateForm();
  setStatus("生成中");
  const layout = await api("/api/v1/calligraphy/layouts/preview", {
    method: "POST",
    body: JSON.stringify(layoutRequest()),
  });
  renderLayout(layout);
  state.draft = null;
  els.artworkTitle.textContent = "作品预览";
  setStatus("已预览", "ok");
}

async function saveDraft() {
  validateForm();
  setStatus("保存中");
  const draft = await api("/api/v1/calligraphy/artworks/drafts", {
    method: "POST",
    body: JSON.stringify({
      owner_user_id: els.ownerUserId.value,
      layout: layoutRequest(),
    }),
  });
  state.draft = draft;
  renderLayout(draft.layout);
  renderExports(draft.exports || []);
  els.artworkTitle.textContent = draft.artwork_id;
  setStatus("已保存", "ok");
  await loadDrafts();
}

async function exportSVG() {
  if (!state.draft) {
    await saveDraft();
  }
  setStatus("导出中");
  const exported = await api(`/api/v1/calligraphy/artworks/drafts/${state.draft.artwork_id}/exports`, {
    method: "POST",
    body: JSON.stringify({ format: "svg", template_type: "reference" }),
  });
  if (exported.inline_content) {
    updateDownload(exported.inline_content);
  } else if (exported.storage_key) {
    updateDownloadURL(`/artifacts/${exported.storage_key}`);
  }
  await reloadCurrentDraft();
  setStatus(exported.storage_key ? "已写入" : "已导出", "ok");
}

async function reloadCurrentDraft() {
  if (!state.draft) return;
  const draft = await api(`/api/v1/calligraphy/artworks/drafts/${state.draft.artwork_id}`);
  state.draft = draft;
  renderExports(draft.exports || []);
}

async function searchGlyphs() {
  const character = encodeURIComponent(els.glyphInput.value.trim());
  const style = encodeURIComponent(els.glyphStyleInput.value);
  if (!character) return;
  const payload = await api(`/api/v1/calligraphy/glyphs/search?character=${character}&style=${style}`);
  els.glyphResults.innerHTML = "";
  if (!payload.items.length) {
    const empty = document.createElement("li");
    empty.className = "empty";
    empty.textContent = "未找到已发布字形";
    els.glyphResults.append(empty);
    return;
  }
  for (const glyph of payload.items) {
    const item = document.createElement("li");
    item.innerHTML = `<strong>${escapeHtml(glyph.character)}</strong><span>${escapeHtml(glyph.calligrapher)} · ${escapeHtml(glyph.copybook_id)} · ${escapeHtml(glyph.license_status)}</span>`;
    item.addEventListener("click", () => loadGlyphDetail(glyph.glyph_id).catch(showError));
    els.glyphResults.append(item);
  }
}

async function loadGlyphDetail(glyphId) {
  const detail = await api(`/api/v1/calligraphy/glyphs/${encodeURIComponent(glyphId)}`);
  renderGlyphDetail(detail);
}

async function loadPresetGroups() {
  const style = encodeURIComponent(els.glyphStyleInput.value);
  const payload = await api(`/api/v1/calligraphy/glyphs/presets?style=${style}`);
  els.presetGroups.innerHTML = "";
  for (const group of payload.items) {
    const section = document.createElement("section");
    section.className = "preset-group";
    const chars = group.glyphs
      .slice(0, 36)
      .map((glyph) => `<button type="button" data-glyph-id="${escapeHtml(glyph.glyph_id)}" data-character="${escapeHtml(glyph.character)}">${escapeHtml(glyph.character)}</button>`)
      .join("");
    section.innerHTML = `<h2>${escapeHtml(group.title)}</h2><p>${escapeHtml(group.description)}</p><div class="char-grid">${chars}</div>`;
    section.querySelectorAll("[data-glyph-id]").forEach((button) => {
      button.addEventListener("click", async () => {
        els.glyphInput.value = button.dataset.character;
        await loadGlyphDetail(button.dataset.glyphId);
        await searchGlyphs();
        setStatus("已选字", "ok");
      });
    });
    els.presetGroups.append(section);
  }
}

function renderGlyphDetail(detail) {
  state.glyphDetail = detail;
  const structure = detail.structure_notes.map((note) => `<li>${escapeHtml(note)}</li>`).join("");
  const brushwork = detail.brushwork_notes.map((note) => `<li>${escapeHtml(note)}</li>`).join("");
  const tabs = detail.practice_templates
    .map((template, index) => `<button type="button" data-template-index="${index}">${escapeHtml(template.title)}</button>`)
    .join("");
  els.glyphDetail.className = "glyph-detail";
  els.glyphDetail.innerHTML = `
    <div class="glyph-card">
      <div class="glyph-large">${escapeHtml(detail.glyph.character)}</div>
      <div>
        <h2>${escapeHtml(detail.glyph.calligrapher)} · ${escapeHtml(detail.glyph.copybook_id)}</h2>
        <p class="meta">${escapeHtml(detail.glyph.style)} · ${escapeHtml(detail.glyph.license_status)}</p>
      </div>
    </div>
    <ul class="note-list">${structure}</ul>
    <ul class="note-list">${brushwork}</ul>
    <div class="detail-actions">
      <button type="button" id="favoriteGlyphButton">收藏</button>
      <button type="button" class="ghost" id="recordPracticeButton">标记已练习</button>
    </div>
    <div class="template-tabs">${tabs}</div>
    <div id="practiceTemplate" class="practice-template"></div>
  `;
  document.querySelector("#favoriteGlyphButton")?.addEventListener("click", () => addFavorite().catch(showError));
  document.querySelector("#recordPracticeButton")?.addEventListener("click", () => recordPractice().catch(showError));
  els.glyphDetail.querySelectorAll("[data-template-index]").forEach((button) => {
    button.addEventListener("click", () => {
      const template = detail.practice_templates[Number(button.dataset.templateIndex)];
      renderPracticeTemplate(detail.glyph.character, template);
    });
  });
  renderPracticeTemplate(detail.glyph.character, detail.practice_templates[0]);
}

function renderPracticeTemplate(character, template) {
  const target = document.querySelector("#practiceTemplate");
  if (!target) return;
  state.practiceTemplate = template;
  const grid = template.grid_type === "jiugong" ? jiugongGrid() : miGrid();
  const fill = template.template_type === "outline" ? "none" : "#111";
  const stroke = template.template_type === "outline" ? "#111" : "none";
  state.practiceSvg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 200" role="img" aria-label="${escapeHtml(template.title)}">
    <rect x="10" y="10" width="180" height="180" fill="#fbf7ef" stroke="#c8b8a4"></rect>
    ${grid}
    <text x="100" y="106" text-anchor="middle" dominant-baseline="central" font-family="serif" font-size="122" fill="${fill}" stroke="${stroke}" stroke-width="2">${escapeHtml(character)}</text>
  </svg>`;
  target.innerHTML = `${state.practiceSvg}<button type="button" class="ghost" id="practiceDownloadButton">下载练习模板</button>`;
  document.querySelector("#practiceDownloadButton")?.addEventListener("click", () => {
    downloadText(`${character}-${template.grid_type}-${template.template_type}.svg`, state.practiceSvg, "image/svg+xml");
  });
}

async function addFavorite() {
  if (!state.glyphDetail) return;
  await api(`/api/v1/calligraphy/users/${encodeURIComponent(currentOwner())}/favorites`, {
    method: "POST",
    body: JSON.stringify({ glyph_id: state.glyphDetail.glyph.glyph_id }),
  });
  await loadLearningProfile();
  setStatus("已收藏", "ok");
}

async function recordPractice() {
  if (!state.glyphDetail) return;
  const template = state.practiceTemplate || state.glyphDetail.practice_templates[0];
  await api(`/api/v1/calligraphy/users/${encodeURIComponent(currentOwner())}/practice`, {
    method: "POST",
    body: JSON.stringify({
      glyph_id: state.glyphDetail.glyph.glyph_id,
      template_type: template.template_type,
      grid_type: template.grid_type,
    }),
  });
  await loadLearningProfile();
  setStatus("已记录练习", "ok");
}

function downloadText(filename, content, type) {
  const blob = new Blob([content], { type });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

function miGrid() {
  return `
    <line x1="100" y1="10" x2="100" y2="190" stroke="#d6c7b3" stroke-dasharray="5 5"></line>
    <line x1="10" y1="100" x2="190" y2="100" stroke="#d6c7b3" stroke-dasharray="5 5"></line>
    <line x1="10" y1="10" x2="190" y2="190" stroke="#d6c7b3" stroke-dasharray="5 5"></line>
    <line x1="190" y1="10" x2="10" y2="190" stroke="#d6c7b3" stroke-dasharray="5 5"></line>
  `;
}

function jiugongGrid() {
  return `
    <line x1="70" y1="10" x2="70" y2="190" stroke="#d6c7b3"></line>
    <line x1="130" y1="10" x2="130" y2="190" stroke="#d6c7b3"></line>
    <line x1="10" y1="70" x2="190" y2="70" stroke="#d6c7b3"></line>
    <line x1="10" y1="130" x2="190" y2="130" stroke="#d6c7b3"></line>
  `;
}

async function loadDrafts() {
  const owner = encodeURIComponent(currentOwner());
  const payload = await api(`/api/v1/calligraphy/artworks/drafts?owner_user_id=${owner}`);
  els.draftList.innerHTML = "";
  if (!payload.items.length) {
    const empty = document.createElement("li");
    empty.className = "empty";
    empty.textContent = "暂无草稿";
    els.draftList.append(empty);
    return;
  }
  for (const draft of payload.items) {
    const item = document.createElement("li");
    const row = document.createElement("div");
    row.className = "draft-row";
    const loadButton = document.createElement("button");
    loadButton.type = "button";
    loadButton.className = "ghost";
    loadButton.textContent = `${draft.artwork_id} · ${draft.layout.normalized_text}`;
    loadButton.addEventListener("click", () => {
      state.draft = draft;
      renderLayout(draft.layout);
      renderExports(draft.exports || []);
      els.artworkTitle.textContent = draft.artwork_id;
      setStatus("已载入", "ok");
    });
    const deleteButton = document.createElement("button");
    deleteButton.type = "button";
    deleteButton.className = "danger";
    deleteButton.textContent = "删除";
    deleteButton.addEventListener("click", async () => {
      await deleteDraft(draft.artwork_id);
    });
    row.append(loadButton, deleteButton);
    item.append(row);
    els.draftList.append(item);
  }
}

async function loadLearningProfile() {
  const payload = await api(`/api/v1/calligraphy/users/${encodeURIComponent(currentOwner())}/learning`);
  state.learningProfile = payload;
  renderLearningProfile(payload);
}

function renderLearningProfile(profile) {
  els.learningStats.innerHTML = `
    <span>收藏 ${profile.favorite_count || 0}</span>
    <span>练习 ${profile.practice_count || 0}</span>
  `;
  renderFavorites(profile.favorites || []);
  renderPractice(profile.recent_practice || []);
}

function renderFavorites(favorites) {
  els.favoriteList.innerHTML = "";
  if (!favorites.length) {
    const empty = document.createElement("li");
    empty.className = "empty";
    empty.textContent = "暂无收藏";
    els.favoriteList.append(empty);
    return;
  }
  for (const favorite of favorites) {
    const item = document.createElement("li");
    const openButton = document.createElement("button");
    openButton.type = "button";
    openButton.className = "ghost compact-button";
    openButton.textContent = `${favorite.character} · ${styleLabel(favorite.style)}`;
    openButton.addEventListener("click", () => loadGlyphDetail(favorite.glyph_id).catch(showError));
    const removeButton = document.createElement("button");
    removeButton.type = "button";
    removeButton.className = "danger compact-remove";
    removeButton.textContent = "移除";
    removeButton.addEventListener("click", async () => {
      await deleteFavorite(favorite.glyph_id);
    });
    item.append(openButton, removeButton);
    els.favoriteList.append(item);
  }
}

function renderPractice(records) {
  els.practiceList.innerHTML = "";
  if (!records.length) {
    const empty = document.createElement("li");
    empty.className = "empty";
    empty.textContent = "暂无练习";
    els.practiceList.append(empty);
    return;
  }
  for (const record of records.slice(0, 8)) {
    const item = document.createElement("li");
    item.className = "practice-record";
    item.innerHTML = `<strong>${escapeHtml(record.character)}</strong><span>${escapeHtml(styleLabel(record.style))} · ${escapeHtml(record.grid_type)} · ${formatTime(record.created_at)}</span>`;
    item.addEventListener("click", () => loadGlyphDetail(record.glyph_id).catch(showError));
    els.practiceList.append(item);
  }
}

async function deleteFavorite(glyphId) {
  const response = await fetch(`/api/v1/calligraphy/users/${encodeURIComponent(currentOwner())}/favorites/${encodeURIComponent(glyphId)}`, { method: "DELETE" });
  if (!response.ok && response.status !== 404) {
    const payload = await response.json();
    throw new Error(payload.message || "delete favorite failed");
  }
  await loadLearningProfile();
  setStatus("已移除", "ok");
}

async function deleteDraft(artworkId) {
  await fetch(`/api/v1/calligraphy/artworks/drafts/${artworkId}`, { method: "DELETE" });
  if (state.draft?.artwork_id === artworkId) {
    state.draft = null;
    els.artworkTitle.textContent = "作品预览";
    els.previewSurface.innerHTML = '<span class="empty">草稿已删除</span>';
    els.layoutMeta.textContent = "尚未生成章法";
    renderExports([]);
    updateDownload("");
  }
  setStatus("已删除", "ok");
  await loadDrafts();
}

function renderExports(exports) {
  els.exportList.innerHTML = "";
  if (!exports.length) {
    const empty = document.createElement("li");
    empty.className = "empty";
    empty.textContent = "暂无导出";
    els.exportList.append(empty);
    return;
  }
  for (const item of exports) {
    const row = document.createElement("li");
    const href = item.storage_key ? `/artifacts/${item.storage_key}` : "";
    row.innerHTML = `<strong>${escapeHtml(item.format.toUpperCase())}</strong><span>${escapeHtml(item.template_type)} · ${escapeHtml(item.sha256.slice(0, 12))} · ${item.byte_size} bytes</span>`;
    if (href) {
      const link = document.createElement("a");
      link.href = href;
      link.download = `${item.artwork_id}-${item.export_id}.${item.format}`;
      link.textContent = "下载";
      row.append(link);
    }
    els.exportList.append(row);
  }
}

function currentOwner() {
  const owner = els.ownerUserId.value.trim();
  if (!owner) throw new Error("请输入用户");
  return owner;
}

function styleLabel(style) {
  return { ou: "欧体", yan: "颜体" }[style] || style;
}

function formatTime(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

for (const el of [els.textInput, els.styleInput, els.formatInput, els.widthInput, els.heightInput, els.marginInput, els.signatureInput, els.sealInput]) {
  el.addEventListener("change", () => {
    state.draft = null;
    setStatus("待预览");
  });
}

els.previewButton.addEventListener("click", () => preview().catch(showError));
els.saveButton.addEventListener("click", () => saveDraft().catch(showError));
els.exportButton.addEventListener("click", () => exportSVG().catch(showError));
els.refreshButton.addEventListener("click", () => loadDrafts().catch(showError));
els.glyphSearchButton.addEventListener("click", () => searchGlyphs().catch(showError));
els.presetRefreshButton.addEventListener("click", () => loadPresetGroups().catch(showError));
els.learningRefreshButton.addEventListener("click", () => loadLearningProfile().catch(showError));
els.ownerUserId.addEventListener("change", () => {
  Promise.all([loadDrafts(), loadLearningProfile()]).catch(showError);
});
els.glyphStyleInput.addEventListener("change", () => {
  searchGlyphs().then(loadPresetGroups).catch(showError);
});

function showError(error) {
  setStatus("失败", "error");
  els.layoutMeta.textContent = error.message;
}

els.previewSurface.innerHTML = '<span class="empty">输入正文后生成预览</span>';
renderExports([]);
renderLearningProfile({ favorites: [], recent_practice: [], favorite_count: 0, practice_count: 0 });
preview()
  .then(loadDrafts)
  .then(searchGlyphs)
  .then(loadPresetGroups)
  .then(loadLearningProfile)
  .catch(showError);
