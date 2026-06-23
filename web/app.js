const state = {
  projects: [],
  selectedProject: "",
  selectedRequirement: "",
  selectedSkill: "",
  discordText: "",
};

const el = (id) => document.getElementById(id);

async function api(path, options = {}) {
  const response = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(data.error || response.statusText);
  return data;
}

async function loadState() {
  const data = await api("/api/state");
  state.projects = data.projects || [];
  const cfg = data.config || {};
  el("openclawUrl").value = cfg.openclaw_base_url || "";
  el("openclawToken").value = cfg.openclaw_token || "";
  el("githubRepo").value = cfg.github_repo || "";
  el("discordUrl").value = cfg.discord_url || "";
  el("discordWidgetUrl").value = cfg.discord_widget_url || "";
  el("discordBotId").value = cfg.discord_bot_id || "";
  el("clawhubUrl").value = cfg.clawhub_base_url || "";
  el("llmUrl").value = cfg.llm_base_url || "";
  renderDiscordWidget();
  renderProjects();
  renderBoard();
}

function showView(viewId) {
  document.querySelectorAll(".view").forEach((node) => node.classList.remove("activeView"));
  document.querySelectorAll(".navItem").forEach((node) => node.classList.toggle("active", node.dataset.view === viewId));
  el(viewId).classList.add("activeView");
}

function renderProjects() {
  const list = el("projectsList");
  const select = el("projectSelect");
  list.innerHTML = "";
  select.innerHTML = "";
  if (!state.projects.length) {
    list.className = "cardsGrid empty";
    list.textContent = "No projects yet";
    return;
  }
  list.className = "cardsGrid";
  for (const project of state.projects) {
    const option = document.createElement("option");
    option.value = project.id;
    option.textContent = project.name;
    select.append(option);

    const card = document.createElement("button");
    card.className = "card";
    card.innerHTML = `<strong>${escapeHtml(project.name)}</strong><span>${escapeHtml(project.description || "")}</span><p class="muted">${(project.requirements || []).length} requirement point(s)</p>`;
    card.addEventListener("click", () => {
      state.selectedProject = project.id;
      select.value = project.id;
      pickLatestRequirement();
      renderBoard();
      showView("boardView");
    });
    list.append(card);
  }
  if (!state.selectedProject) state.selectedProject = select.value || state.projects[0].id;
  select.value = state.selectedProject;
  pickLatestRequirement();
}

function pickLatestRequirement() {
  const project = currentProject();
  const latest = project ? (project.requirements || []).at(-1) : null;
  if (!state.selectedRequirement && latest) state.selectedRequirement = latest.id;
}

async function saveConfig() {
  await api("/api/config", {
    method: "PUT",
    body: JSON.stringify({
      openclaw_base_url: el("openclawUrl").value.trim(),
      openclaw_token: el("openclawToken").value.trim(),
      github_repo: el("githubRepo").value.trim(),
      discord_url: el("discordUrl").value.trim(),
      discord_widget_url: el("discordWidgetUrl").value.trim(),
      discord_bot_id: el("discordBotId").value.trim(),
      clawhub_base_url: el("clawhubUrl").value.trim(),
      llm_base_url: el("llmUrl").value.trim(),
    }),
  });
  await refreshConnections();
  renderDiscordWidget();
}

function renderDiscordWidget() {
  const iframe = el("discordWidget");
  const url = el("discordWidgetUrl").value.trim();
  if (!url) {
    iframe.removeAttribute("src");
    return;
  }
  iframe.src = url;
}

async function refreshConnections() {
  await Promise.allSettled([loadOpenClawStatus(), loadSkills(), loadRuntimes()]);
}

async function loadOpenClawStatus() {
  const dot = el("openclawDot");
  const text = el("openclawText");
  const meta = el("openclawMeta");
  try {
    const status = await api("/api/openclaw/status");
    dot.className = `dot ${status.online ? "ok" : "bad"}`;
    text.textContent = status.online ? "OpenClaw online" : "OpenClaw offline";
    meta.textContent = status.online
      ? `${status.name || "agent"} ${status.version || ""} ${status.agent_id || ""}`.trim()
      : status.error || "not reachable";
  } catch (error) {
    dot.className = "dot bad";
    text.textContent = "OpenClaw offline";
    meta.textContent = error.message;
  }
}

async function loadSkills() {
  const skills = await api("/api/clawhub/skills");
  renderSkills(skills);
}

function renderSkills(skills) {
  const list = el("skillsList");
  list.innerHTML = "";
  if (!skills.length) {
    list.className = "cardsGrid empty";
    list.textContent = "No private ClawHub skills loaded";
    return;
  }
  list.className = "cardsGrid";
  for (const skill of skills) {
    const card = document.createElement("button");
    card.className = "card";
    card.innerHTML = `<strong>${escapeHtml(skill.name || skill.id)}</strong><span>${escapeHtml((skill.capabilities || []).join(", "))}</span><p class="muted">${escapeHtml(skill.version || "")}</p>`;
    card.addEventListener("click", () => selectSkill(skill.id));
    list.append(card);
  }
}

async function selectSkill(skillId) {
  state.selectedSkill = skillId;
  const skill = await api(`/api/clawhub/skills/${encodeURIComponent(skillId)}`);
  el("skillDetail").className = "";
  el("skillDetail").innerHTML = `<strong>${escapeHtml(skill.name || skill.id)}</strong><p>${escapeHtml(skill.description || "No description")}</p><p class="muted">${escapeHtml((skill.capabilities || []).join(", "))}</p><p class="muted">${escapeHtml(skill.runtime || "")} ${escapeHtml(skill.entrypoint || "")}</p>`;
}

async function installSelectedSkill() {
  if (!state.selectedSkill) throw new Error("Select a skill first");
  const result = await api(`/api/clawhub/skills/${encodeURIComponent(state.selectedSkill)}/install`, {
    method: "POST",
    body: JSON.stringify({ agent_id: el("skillAgentId").value.trim() }),
  });
  alert(result.result.sent ? "Install instruction sent to OpenClaw" : result.result.message || "Instruction not sent");
}

async function loadRuntimes() {
  const runtimes = await api("/api/plugin-runtimes");
  const list = el("runtimeList");
  list.innerHTML = "";
  list.className = "cardsGrid";
  for (const runtime of runtimes) {
    const card = document.createElement("div");
    card.className = "card";
    card.innerHTML = `<strong>${escapeHtml(runtime.name)}</strong><span>${escapeHtml((runtime.extensions || []).join(", "))}</span><p class="muted">${escapeHtml(runtime.command_hint || "")}</p>`;
    list.append(card);
  }
}

async function createProject() {
  const project = await api("/api/projects", {
    method: "POST",
    body: JSON.stringify({
      name: el("projectName").value.trim(),
      description: el("projectDesc").value.trim(),
      plan: el("projectPlan").value.trim(),
      docs: el("projectDocs").value.trim(),
    }),
  });
  state.selectedProject = project.id;
  await loadState();
  showView("boardView");
}

async function addRequirement() {
  const project = currentProject();
  if (!project) throw new Error("Create or select a project first");
  const req = await api(`/api/projects/${project.id}/requirements`, {
    method: "POST",
    body: JSON.stringify({
      title: el("reqTitle").value.trim(),
      description: el("reqDesc").value.trim(),
    }),
  });
  state.selectedRequirement = req.id;
  await loadState();
  showView("taskView");
}

async function generatePrompt() {
  const project = currentProject();
  if (!project) throw new Error("Create or select a project first");
  const requirement = currentRequirement(project);
  if (!requirement) throw new Error("Select or create a requirement first");
  const result = await api(`/api/projects/${project.id}/prompt`, {
    method: "POST",
    body: JSON.stringify({ requirement_id: requirement.id }),
  });
  el("artifactPath").textContent = result.artifact_path;
  state.discordText = result.discord_text || result.segments.join("\n\n");
  const segments = el("segments");
  segments.innerHTML = "";
  segments.className = "segments";
  result.segments.forEach((segment, index) => {
    const node = document.createElement("div");
    node.className = "segment";
    node.innerHTML = `<strong>Segment ${index + 1}</strong><pre>${escapeHtml(segment)}</pre>`;
    segments.append(node);
  });
}

async function copyDiscordPrompt() {
  if (!state.discordText) await generatePrompt();
  await navigator.clipboard.writeText(state.discordText);
  alert("Copied Discord prompt");
}

function openDiscord() {
  const url = el("discordUrl").value.trim() || "https://discord.com/app";
  window.open(url, "_blank", "noopener");
}

async function pushGitHub() {
  const result = await api("/api/github/push", { method: "POST", body: "{}" });
  alert(result.status);
}

function currentProject() {
  const selected = el("projectSelect").value || state.selectedProject;
  return state.projects.find((project) => project.id === selected) || state.projects[0];
}

function currentRequirement(project) {
  const requirements = project.requirements || [];
  return requirements.find((req) => req.id === state.selectedRequirement) || requirements.at(-1);
}

function renderBoard() {
  const list = el("boardList");
  const project = currentProject();
  const requirements = project ? project.requirements || [] : [];
  list.innerHTML = "";
  if (!requirements.length) {
    list.className = "kanbanBoard empty";
    list.textContent = "No requirements";
    renderAgentInbox([]);
    return;
  }
  list.className = "kanbanBoard";
  const columns = [
    { id: "draft", title: "BACKLOG", color: "low" },
    { id: "sent", title: "TO DO", color: "medium" },
    { id: "in_progress", title: "IN PROGRESS", color: "agent" },
    { id: "closed", title: "DONE", color: "done" },
  ];
  for (const column of columns) {
    const reqs = requirements.filter((req) => (req.status || "draft") === column.id || (column.id === "draft" && !req.status));
    const lane = document.createElement("section");
    lane.className = "kanbanColumn";
    lane.innerHTML = `<div class="columnHead">${column.title} <span>${reqs.length}</span></div><div class="addLane">+</div>`;
    for (const req of reqs) {
      lane.append(renderIssueCard(req, column.color));
    }
    list.append(lane);
  }
  renderAgentInbox(requirements);
  document.querySelectorAll("[data-open-req]").forEach((node) => {
    node.addEventListener("click", () => {
      state.selectedRequirement = node.dataset.openReq;
      showView("taskView");
    });
  });
}

function renderIssueCard(req, color) {
  const card = document.createElement("article");
  card.className = "issueCard";
  const status = req.status || "draft";
  const progress = status === "closed" ? "100%" : status === "in_progress" ? "55%" : status === "sent" ? "15%" : "0%";
  card.innerHTML = `
    <div class="issueTitle">${escapeHtml(req.title)}</div>
    <div class="badges">
      <span class="badge ${color}">${status === "closed" ? "Done" : status === "in_progress" ? "Medium" : "Low"}</span>
      <span class="badge agent">OpenClaw</span>
    </div>
    <div class="issueMeta">
      <span class="progressRing"></span>
      <span class="muted">${progress}</span>
      ${req.commit_id ? `<span class="muted">${escapeHtml(req.commit_id)}</span>` : ""}
    </div>
    <div class="avatars">
      <span class="avatar">A</span>
      <span class="muted">${escapeHtml(req.id.slice(0, 8))}</span>
    </div>
    <div class="itemRow">
      <input type="checkbox" data-req-id="${escapeHtml(req.id)}" />
      <button class="secondary" data-open-req="${escapeHtml(req.id)}">Open</button>
    </div>`;
  return card;
}

function renderAgentInbox(requirements) {
  const inbox = el("agentInbox");
  if (!inbox) return;
  inbox.innerHTML = "";
  const recent = requirements.slice(-6).reverse();
  if (!recent.length) {
    inbox.innerHTML = `<div class="empty">No agent activity</div>`;
    return;
  }
  for (const req of recent) {
    const item = document.createElement("div");
    item.className = "inboxItem";
    item.innerHTML = `<strong>OpenClaw A ${escapeHtml(req.status || "draft")}</strong><span class="muted">${escapeHtml(req.title)}</span>`;
    inbox.append(item);
  }
}

async function closeSelected() {
  const project = currentProject();
  if (!project) throw new Error("Create or select a project first");
  const ids = [...document.querySelectorAll("[data-req-id]:checked")].map((node) => node.dataset.reqId);
  if (!ids.length) throw new Error("Select at least one requirement");
  const commitId = el("commitId").value.trim();
  if (!commitId) throw new Error("Paste OpenClaw A commit sha first");
  await api(`/api/projects/${project.id}/board`, {
    method: "POST",
    body: JSON.stringify({ requirement_ids: ids, commit_id: commitId, status: "closed" }),
  });
  await loadState();
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function bind(id, fn) {
  el(id).addEventListener("click", async () => {
    try {
      await fn();
    } catch (error) {
      alert(error.message);
    }
  });
}

document.querySelectorAll("[data-view]").forEach((node) => {
  node.addEventListener("click", () => showView(node.dataset.view));
});
el("projectSelect").addEventListener("change", () => {
  state.selectedProject = el("projectSelect").value;
  state.selectedRequirement = "";
  pickLatestRequirement();
  renderBoard();
});

bind("collapseNavBtn", async () => el("navRail").classList.toggle("collapsed"));
bind("refreshBtn", async () => {
  await loadState();
  await refreshConnections();
});
bind("saveConfigBtn", saveConfig);
bind("createProjectBtn", createProject);
bind("addReqBtn", addRequirement);
bind("promptBtn", generatePrompt);
bind("copyDiscordBtn", copyDiscordPrompt);
bind("openDiscordBtn", async () => openDiscord());
bind("pushBtn", pushGitHub);
bind("closeSelectedBtn", closeSelected);
bind("loadSkillsBtn", loadSkills);
bind("installSkillBtn", installSelectedSkill);

loadState().then(refreshConnections).catch((error) => alert(error.message));
