const state = {
  projects: [],
  selectedProject: "",
  selectedBranch: "",
  selectedRequirement: "",
  selectedSkill: "",
  selectedRequirements: new Set(),
  newRequirementStatus: "draft",
  boardTab: "kanban",
  tailscaleDevices: [],
  tailscaleError: "",
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
  renderTodoRatio();
  loadTailscaleDevices().catch((error) => {
    state.tailscaleError = error.message;
    renderOpenClawStrip();
  });
}

function showView(viewId) {
  document.querySelectorAll(".view").forEach((node) => node.classList.remove("activeView"));
  document.querySelectorAll("[data-view]").forEach((node) => node.classList.toggle("active", node.dataset.view === viewId));
  el(viewId).classList.add("activeView");
}

function renderProjects() {
  const list = el("projectsList");
  list.innerHTML = "";
  if (state.selectedProject && !state.projects.some((project) => project.id === state.selectedProject)) {
    state.selectedProject = "";
    state.selectedBranch = "";
  }
  if (!state.projects.length) {
    list.className = "cardsGrid empty";
    list.textContent = "No projects yet";
    renderSideProjects();
    return;
  }
  list.className = "cardsGrid";
  for (const project of state.projects) {
    const card = document.createElement("button");
    card.className = "card";
    card.innerHTML = `<strong>${escapeHtml(project.name)}</strong><span>${escapeHtml(project.description || "")}</span><p class="muted">${(project.requirements || []).length} requirement point(s)</p>`;
    card.addEventListener("click", () => {
      state.selectedProject = project.id;
      state.selectedRequirements.clear();
      state.selectedBranch = firstBranch(project).id;
      state.boardTab = "kanban";
      pickLatestRequirement();
      renderBranches();
      renderSideProjects();
      renderBoard();
      showView("boardView");
    });
    list.append(card);
  }
  renderBranches();
  pickLatestRequirement();
  renderSideProjects();
  renderTodoRatio();
}

function renderSideProjects() {
  const list = el("sideProjectLinks");
  el("allProjectsBtn").classList.toggle("active", !state.selectedProject);
  list.innerHTML = "";
  if (!state.projects.length) {
    list.className = "sideProjectLinks empty";
    list.textContent = "No projects";
    return;
  }
  list.className = "sideProjectLinks";
  for (const project of state.projects) {
    const group = document.createElement("div");
    group.className = "projectTree";
    const button = document.createElement("button");
    button.className = `sideLink projectLink ${project.id === state.selectedProject ? "active" : ""}`;
    button.textContent = project.name;
    button.addEventListener("click", () => {
      state.selectedProject = project.id;
      state.selectedRequirements.clear();
      state.selectedBranch = firstBranch(project).id;
      state.selectedRequirement = "";
      state.boardTab = "kanban";
      pickLatestRequirement();
      renderBranches();
      renderSideProjects();
      renderBoard();
      showView("boardView");
    });
    group.append(button);
    if (project.id === state.selectedProject) {
      const branches = document.createElement("div");
      branches.className = "branchTree";
      for (const branch of normalizedBranches(project)) {
        const branchButton = document.createElement("button");
        branchButton.className = `sideLink branchLink ${branch.id === state.selectedBranch ? "active" : ""}`;
        branchButton.textContent = branch.name;
        branchButton.addEventListener("click", () => selectBranch(project.id, branch.id));
        branches.append(branchButton);
      }
      group.append(branches);
    }
    list.append(group);
  }
}

function pickLatestRequirement() {
  const project = currentProject();
  const latest = project ? currentBranchRequirements(project).at(-1) : null;
  if (!state.selectedRequirement && latest) state.selectedRequirement = latest.id;
}

function renderBranches() {
  const select = el("branchSelect");
  const project = currentProject();
  select.innerHTML = "";
  if (!project) {
    select.disabled = true;
    select.innerHTML = `<option value="">No project</option>`;
    renderBranchList();
    return;
  }
  select.disabled = false;
  const branches = normalizedBranches(project);
  if (!state.selectedBranch || !branches.some((branch) => branch.id === state.selectedBranch)) {
    state.selectedBranch = branches[0].id;
  }
  for (const branch of branches) {
    const option = document.createElement("option");
    option.value = branch.id;
    option.textContent = branch.name;
    select.append(option);
  }
  select.value = state.selectedBranch;
  renderBranchList();
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

function connectDiscordPanel() {
  const url = el("discordWidgetUrl").value.trim();
  el("discordPreviewShell").classList.add("hidden");
  el("discordEmbedShell").classList.remove("hidden");
  const iframe = el("taskDiscordWidget");
  const hint = el("discordConnectHint");
  if (!url) {
    iframe.classList.add("hidden");
    hint.classList.remove("hidden");
    return;
  }
  iframe.classList.remove("hidden");
  hint.classList.add("hidden");
  iframe.src = url;
}

function showDiscordPreview() {
  el("discordEmbedShell").classList.add("hidden");
  el("discordPreviewShell").classList.remove("hidden");
}

async function refreshConnections() {
  await Promise.allSettled([loadOpenClawStatus(), loadTailscaleDevices(), loadSkills(), loadRuntimes()]);
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

async function loadTailscaleDevices() {
  const status = await api("/api/tailscale/devices");
  state.tailscaleDevices = status.devices || [];
  state.tailscaleError = status.error || "";
  renderOpenClawStrip(status.tailnet || "");
}

function renderOpenClawStrip(tailnet = "") {
  const strip = el("openClawStrip");
  if (!strip) return;
  strip.innerHTML = "";
  const online = state.tailscaleDevices.filter((device) => device.online);
  if (!online.length) {
    strip.className = "openClawStrip empty";
    strip.textContent = state.tailscaleError ? `Tailscale unavailable: ${state.tailscaleError}` : "No online OpenClaw devices";
    return;
  }
  strip.className = "openClawStrip";
  const label = document.createElement("span");
  label.className = "stripLabel";
  label.textContent = tailnet ? `Online OpenClaw · ${tailnet}` : "Online OpenClaw";
  strip.append(label);
  for (const device of online) {
    const node = document.createElement("button");
    node.className = "openClawChip";
    node.title = device.openclaw_url || device.dns_name || device.ip || device.name;
    node.innerHTML = `<span class="onlineDot"></span><strong>${escapeHtml(shortDeviceName(device.name))}</strong><span>${escapeHtml(device.ip || "")}</span>`;
    if (device.openclaw_url) {
      node.addEventListener("click", () => {
        el("openclawUrl").value = device.openclaw_url;
        saveConfig().catch((error) => alert(error.message));
      });
    }
    strip.append(node);
  }
}

function shortDeviceName(name) {
  return String(name || "OpenClaw").replace(/的MacBook Pro|的Mac mini|MacBook Pro|Mac mini/g, "").trim() || name || "OpenClaw";
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
  state.selectedBranch = firstBranch(project).id;
  await loadState();
  showView("boardView");
}

async function createBranch() {
  const project = currentProject();
  if (!project) throw new Error("Create or select a project first");
  const name = el("branchName").value.trim();
  if (!name) throw new Error("Branch name is required");
  const branch = await api(`/api/projects/${project.id}/branches`, {
    method: "POST",
    body: JSON.stringify({ name }),
  });
  state.selectedBranch = branch.id;
  state.boardTab = "kanban";
  closeBranchModal();
  el("branchName").value = "";
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
      priority: selectedPriority(),
      status: state.newRequirementStatus || "draft",
      branch_id: state.selectedBranch || firstBranch(project).id,
      agent_id: "openclaw-a",
    }),
  });
  state.selectedRequirement = req.id;
  closeRequirementModal();
  el("reqTitle").value = "";
  el("reqDesc").value = "";
  await loadState();
  showView("boardView");
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
  renderDiscordPreview();
}

function renderDiscordPreview() {
  const preview = el("discordPreview");
  if (!preview) return;
  if (!state.discordText) {
    preview.innerHTML = `<div class="discordMessage muted">Generate a prompt, then copy it into Discord.</div>`;
    return;
  }
  preview.innerHTML = `<div class="discordMessage"><strong>MutiSolo</strong>${escapeHtml(state.discordText)}</div>`;
}

async function copyDiscordPrompt() {
  if (!state.discordText) await generatePrompt();
  await navigator.clipboard.writeText(state.discordText);
  alert("Copied Discord prompt");
}

function openDiscord() {
  connectDiscordPanel();
}

async function pushGitHub() {
  const result = await api("/api/github/push", { method: "POST", body: "{}" });
  alert(result.status);
}

function currentProject() {
  const selected = state.selectedProject;
  return state.projects.find((project) => project.id === selected) || null;
}

function currentRequirement(project) {
  const requirements = currentBranchRequirements(project);
  return requirements.find((req) => req.id === state.selectedRequirement) || requirements.at(-1);
}

function normalizedBranches(project) {
  return project.branches && project.branches.length ? project.branches : [{ id: "main", name: "Main" }];
}

function firstBranch(project) {
  return normalizedBranches(project)[0];
}

function currentBranchRequirements(project) {
  const branchID = state.selectedBranch || firstBranch(project).id;
  return (project.requirements || []).filter((req) => (req.branch_id || firstBranch(project).id) === branchID);
}

function renderBoard() {
  const list = el("boardList");
  const project = currentProject();
  if (!project) {
    list.innerHTML = "";
    list.className = "kanbanBoard empty";
    list.textContent = "Select a project to view its board";
    state.selectedRequirements.clear();
    renderSelectionToolbar();
    renderBranchList();
    renderBoardMode();
    return;
  }
  const requirements = project ? currentBranchRequirements(project) : [];
  list.innerHTML = "";
  if (!requirements.length) {
    list.className = "kanbanBoard empty";
    list.textContent = "No requirements";
    renderSelectionToolbar();
    renderBranchList();
    renderBoardMode();
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
    lane.dataset.status = column.id;
    lane.innerHTML = `<div class="columnHead">${column.title} <span>${reqs.length}</span></div><button class="addLane" data-add-status="${column.id}">+</button>`;
    lane.addEventListener("dragover", (event) => event.preventDefault());
    lane.addEventListener("drop", (event) => moveRequirement(event.dataTransfer.getData("text/plain"), column.id));
    for (const req of reqs) {
      lane.append(renderIssueCard(req, column.color));
    }
    list.append(lane);
  }
  document.querySelectorAll("[data-select-req]").forEach((node) => {
    node.addEventListener("change", () => {
      toggleRequirementSelection(node.dataset.selectReq, node.checked);
      node.closest(".issueWrap")?.classList.toggle("selected", node.checked);
    });
  });
  document.querySelectorAll("[data-add-status]").forEach((node) => {
    node.addEventListener("click", () => openRequirementModal(node.dataset.addStatus));
  });
  document.querySelectorAll("[data-agent-req]").forEach((node) => {
    node.addEventListener("change", () => assignRequirementAgent(node.dataset.agentReq, node.value));
  });
  renderSelectionToolbar();
  renderBranchList();
  renderBoardMode();
}

function renderIssueCard(req, color) {
  const wrap = document.createElement("div");
  wrap.className = `issueWrap ${state.selectedRequirements.has(req.id) ? "selected" : ""}`;
  wrap.innerHTML = `<div class="issueSelect"><input type="checkbox" data-select-req="${escapeHtml(req.id)}" ${state.selectedRequirements.has(req.id) ? "checked" : ""} /></div>`;
  const card = document.createElement("article");
  card.className = "issueCard";
  card.draggable = true;
  card.addEventListener("dragstart", (event) => event.dataTransfer.setData("text/plain", req.id));
  const status = req.status || "draft";
  const agentID = req.agent_id || "openclaw-a";
  card.innerHTML = `
    <div class="issueBody">
    <div class="issueTitle">${escapeHtml(req.title)}</div>
    <div class="badges">
      <span class="badge ${priorityClass(req.priority)}">${priorityLabel(req.priority)}</span>
      <span class="badge agent">${escapeHtml(agentLabel(agentID))}</span>
    </div>
    <div class="assigneeRow">
      <span class="avatar">A</span>
      <select class="agentSelect" data-agent-req="${escapeHtml(req.id)}">
        ${agentOptions(agentID)}
      </select>
    </div>
    ${req.commit_id ? `<div class="issueMeta"><span class="muted">${escapeHtml(req.commit_id)}</span></div>` : ""}
    </div>`;
  card.addEventListener("click", (event) => {
    if (event.target.closest("input,select")) return;
    const selected = !state.selectedRequirements.has(req.id);
    toggleRequirementSelection(req.id, selected);
    wrap.classList.toggle("selected", selected);
    const checkbox = wrap.querySelector("[data-select-req]");
    if (checkbox) checkbox.checked = selected;
  });
  card.addEventListener("dblclick", () => {
    state.selectedRequirement = req.id;
    showView("taskView");
  });
  wrap.append(card);
  return wrap;
}

function renderBranchList() {
  const list = el("branchList");
  const project = currentProject();
  list.innerHTML = "";
  if (!project) {
    list.className = "branchList empty";
    list.textContent = "No project selected";
    return;
  }
  list.className = "branchList";
  const requirements = project.requirements || [];
  for (const branch of normalizedBranches(project)) {
    const branchRequirements = requirements.filter((req) => (req.branch_id || firstBranch(project).id) === branch.id);
    const card = document.createElement("button");
    card.className = `branchCard ${branch.id === state.selectedBranch ? "active" : ""}`;
    card.innerHTML = `
      <strong>${escapeHtml(branch.name)}</strong>
      <span>${branchRequirements.length} requirement point(s)</span>
      <div class="branchStats">
        <span>Backlog ${countStatus(branchRequirements, "draft")}</span>
        <span>To do ${countStatus(branchRequirements, "sent")}</span>
        <span>Progress ${countStatus(branchRequirements, "in_progress")}</span>
      </div>`;
    card.addEventListener("click", () => selectBranch(project.id, branch.id));
    list.append(card);
  }
}

function renderBoardMode() {
  const showBranches = state.boardTab === "branch";
  el("boardList").classList.toggle("hidden", showBranches);
  el("branchList").classList.toggle("hidden", !showBranches);
  el("kanbanTabBtn").classList.toggle("active", !showBranches);
  el("branchTabBtn").classList.toggle("active", showBranches);
}

function showKanbanTab() {
  state.boardTab = "kanban";
  renderBoardMode();
}

function showBranchTab() {
  state.boardTab = "branch";
  renderBranchList();
  renderBoardMode();
}

function selectBranch(projectID, branchID) {
  state.selectedProject = projectID;
  state.selectedBranch = branchID;
  state.selectedRequirement = "";
  state.selectedRequirements.clear();
  state.boardTab = "kanban";
  pickLatestRequirement();
  renderBranches();
  renderSideProjects();
  renderBoard();
  showView("boardView");
}

function countStatus(requirements, status) {
  return requirements.filter((req) => (req.status || "draft") === status || (status === "draft" && !req.status)).length;
}

function toggleRequirementSelection(reqID, selected) {
  if (selected) {
    state.selectedRequirements.add(reqID);
  } else {
    state.selectedRequirements.delete(reqID);
  }
  renderSelectionToolbar();
}

async function closeSelected() {
  const project = currentProject();
  if (!project) throw new Error("Create or select a project first");
  const ids = [...state.selectedRequirements];
  if (!ids.length) throw new Error("Select at least one requirement");
  await api(`/api/projects/${project.id}/board`, {
    method: "POST",
    body: JSON.stringify({ requirement_ids: ids, status: "closed" }),
  });
  state.selectedRequirements.clear();
  await loadState();
}

async function moveRequirement(reqID, status) {
  if (!reqID || !status) return;
  const project = currentProject();
  if (!project) return;
  await api(`/api/projects/${project.id}/board`, {
    method: "POST",
    body: JSON.stringify({ requirement_ids: [reqID], status }),
  });
  await loadState();
}

async function moveRequirementToBranch(reqID, branchID) {
  if (!reqID || !branchID) return;
  const project = currentProject();
  if (!project) return;
  await api(`/api/projects/${project.id}/board`, {
    method: "POST",
    body: JSON.stringify({ requirement_ids: [reqID], branch_id: branchID, status: "draft" }),
  });
  state.selectedRequirements.delete(reqID);
  await loadState();
}

async function assignRequirementAgent(reqID, agentID) {
  if (!reqID || !agentID) return;
  const project = currentProject();
  if (!project) return;
  await api(`/api/projects/${project.id}/board`, {
    method: "POST",
    body: JSON.stringify({ requirement_ids: [reqID], agent_id: agentID }),
  });
  await loadState();
}

function renderSelectionToolbar() {
  const count = state.selectedRequirements.size;
  el("selectionToolbar").classList.toggle("hidden", count === 0);
  el("selectionCount").textContent = `${count} selected`;
  renderMoveBranchControls();
}

function renderMoveBranchControls() {
  const select = el("selectedBranchMoveSelect");
  const button = el("moveSelectedBranchBtn");
  select.innerHTML = `<option value="">Select branch...</option>`;
  select.disabled = true;
  button.disabled = true;
  const moveTarget = selectedBacklogRequirementForMove();
  if (!moveTarget) return;
  for (const branch of moveTarget.branches) {
    const option = document.createElement("option");
    option.value = branch.id;
    option.textContent = branch.name;
    select.append(option);
  }
  select.disabled = moveTarget.branches.length === 0;
  button.disabled = true;
}

function selectedBacklogRequirementForMove() {
  if (state.selectedRequirements.size !== 1) return null;
  const project = currentProject();
  if (!project) return null;
  const reqID = [...state.selectedRequirements][0];
  const requirement = currentBranchRequirements(project).find((req) => req.id === reqID);
  if (!requirement || (requirement.status || "draft") !== "draft") return null;
  const currentBranchID = requirement.branch_id || firstBranch(project).id;
  const branches = normalizedBranches(project).filter((branch) => branch.id !== currentBranchID);
  if (!branches.length) return null;
  return { requirement, branches };
}

function updateMoveButtonState() {
  el("moveSelectedBranchBtn").disabled = !el("selectedBranchMoveSelect").value;
}

function renderTodoRatio() {
  const all = state.projects.flatMap((project) => project.requirements || []);
  const denominator = all.filter((req) => ["draft", "sent", "in_progress", ""].includes(req.status || "")).length;
  const todo = all.filter((req) => (req.status || "") === "sent").length;
  el("todoRatio").textContent = `${denominator ? Math.round((todo / denominator) * 100) : 0}%`;
}

function openRequirementModal(status = "draft") {
  state.newRequirementStatus = status;
  const low = document.querySelector("[name=reqPriority][value=low]");
  if (low) low.checked = true;
  el("requirementModal").classList.remove("hidden");
  el("reqTitle").focus();
}

function closeRequirementModal() {
  el("requirementModal").classList.add("hidden");
}

function openBranchModal() {
  if (!currentProject()) {
    alert("Select a project first");
    return;
  }
  el("branchModal").classList.remove("hidden");
  el("branchName").focus();
}

function closeBranchModal() {
  el("branchModal").classList.add("hidden");
}

function selectedPriority() {
  return document.querySelector("[name=reqPriority]:checked")?.value || "low";
}

function priorityLabel(priority) {
  return {
    no_priority: "No priority",
    urgent: "Urgent",
    high: "High",
    medium: "Medium",
    low: "Low",
  }[priority || "low"] || "Low";
}

function priorityClass(priority) {
  return {
    no_priority: "none",
    urgent: "urgent",
    high: "high",
    medium: "medium",
    low: "low",
  }[priority || "low"] || "low";
}

function agentLabel(agentID) {
  return {
    "openclaw-a": "OpenClaw A",
    "openclaw-b": "OpenClaw B",
    "openclaw-c": "OpenClaw C",
  }[agentID || "openclaw-a"] || "OpenClaw A";
}

function agentOptions(selectedAgent) {
  return [
    ["openclaw-a", "OpenClaw A"],
    ["openclaw-b", "OpenClaw B"],
    ["openclaw-c", "OpenClaw C"],
  ]
    .map(([value, label]) => `<option value="${value}" ${value === selectedAgent ? "selected" : ""}>${label}</option>`)
    .join("");
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
  node.addEventListener("click", () => {
    if (node.dataset.view === "projectsView") {
      showProjectList();
      return;
    }
    showView(node.dataset.view);
  });
});
el("branchSelect").addEventListener("change", () => {
  state.selectedBranch = el("branchSelect").value;
  state.selectedRequirement = "";
  state.selectedRequirements.clear();
  state.boardTab = "kanban";
  pickLatestRequirement();
  renderSideProjects();
  renderBoard();
});

function showProjectList() {
  state.selectedProject = "";
  state.selectedBranch = "";
  state.selectedRequirement = "";
  state.selectedRequirements.clear();
  state.boardTab = "kanban";
  renderBranches();
  renderSideProjects();
  renderBoard();
  showView("projectsView");
}

async function moveSelectedRequirementToBranch() {
  const moveTarget = selectedBacklogRequirementForMove();
  const branchID = el("selectedBranchMoveSelect").value;
  if (!moveTarget || !branchID) return;
  await moveRequirementToBranch(moveTarget.requirement.id, branchID);
}

bind("refreshBtn", async () => {
  await loadState();
  await refreshConnections();
});
bind("allProjectsBtn", async () => showProjectList());
bind("saveConfigBtn", saveConfig);
bind("createProjectBtn", createProject);
bind("createBranchBtn", async () => openBranchModal());
bind("confirmBranchBtn", createBranch);
bind("addReqBtn", addRequirement);
bind("openRequirementModalBtn", async () => openRequirementModal("draft"));
bind("closeRequirementModalBtn", async () => closeRequirementModal());
bind("closeBranchModalBtn", async () => closeBranchModal());
bind("kanbanTabBtn", async () => showKanbanTab());
bind("branchTabBtn", async () => showBranchTab());
bind("promptBtn", generatePrompt);
bind("copyDiscordBtn", copyDiscordPrompt);
bind("openDiscordBtn", async () => openDiscord());
bind("closeDiscordEmbedBtn", async () => showDiscordPreview());
bind("pushBtn", pushGitHub);
bind("closeSelectedBtn", closeSelected);
bind("moveSelectedBranchBtn", moveSelectedRequirementToBranch);
bind("loadSkillsBtn", loadSkills);
bind("installSkillBtn", installSelectedSkill);
el("selectedBranchMoveSelect").addEventListener("change", updateMoveButtonState);

loadState().then(refreshConnections).catch((error) => alert(error.message));
