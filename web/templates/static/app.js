const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => Array.from(document.querySelectorAll(selector));
const state = { tab: "home", theme: localStorage.getItem("theme") || "light", users: [], domain: "", port: 443, version: {}, overview: null, overviewFetchedAt: 0, processMetrics: [], processFetchedAt: 0, cert: null, certFetchedAt: 0, service: {}, serviceBusy: "", refreshTimer: 0, loginTitle: "登录", refreshing: false, history: { groups: [], expanded: "", pages: {}, pageSizes: {}, sorts: {}, details: {}, meta: {}, datePicker: { active: "start", month: "" } } };
const titles = { home: ["仪表盘", "今日业务、账号健康与在线状态"], ops: ["运维监控", "合成探测、SLA 趋势与资源风险"], users: ["用户管理", "用户、流量、期限与分享"], history: ["访问历史", "用户访问目标统计"], service: ["Trojan管理", "服务控制、切换与日志"], system: ["系统监控", "月度容量、资源与进程压力"], settings: ["设置", "界面主题与偏好"] };

function rightRotate(v, n) { return (v >>> n) | (v << (32 - n)); }
function sha224(input) {
  const bytes = Array.from(new TextEncoder().encode(input)); const bits = bytes.length * 8; bytes.push(0x80);
  while ((bytes.length % 64) !== 56) bytes.push(0);
  for (let i = 7; i >= 0; i -= 1) bytes.push((bits / (2 ** (i * 8))) & 0xff);
  const k = [0x428a2f98,0x71374491,0xb5c0fbcf,0xe9b5dba5,0x3956c25b,0x59f111f1,0x923f82a4,0xab1c5ed5,0xd807aa98,0x12835b01,0x243185be,0x550c7dc3,0x72be5d74,0x80deb1fe,0x9bdc06a7,0xc19bf174,0xe49b69c1,0xefbe4786,0x0fc19dc6,0x240ca1cc,0x2de92c6f,0x4a7484aa,0x5cb0a9dc,0x76f988da,0x983e5152,0xa831c66d,0xb00327c8,0xbf597fc7,0xc6e00bf3,0xd5a79147,0x06ca6351,0x14292967,0x27b70a85,0x2e1b2138,0x4d2c6dfc,0x53380d13,0x650a7354,0x766a0abb,0x81c2c92e,0x92722c85,0xa2bfe8a1,0xa81a664b,0xc24b8b70,0xc76c51a3,0xd192e819,0xd6990624,0xf40e3585,0x106aa070,0x19a4c116,0x1e376c08,0x2748774c,0x34b0bcb5,0x391c0cb3,0x4ed8aa4a,0x5b9cca4f,0x682e6ff3,0x748f82ee,0x78a5636f,0x84c87814,0x8cc70208,0x90befffa,0xa4506ceb,0xbef9a3f7,0xc67178f2];
  let h = [0xc1059ed8,0x367cd507,0x3070dd17,0xf70e5939,0xffc00b31,0x68581511,0x64f98fa7,0xbefa4fa4];
  for (let o = 0; o < bytes.length; o += 64) {
    const w = new Array(64);
    for (let i = 0; i < 16; i += 1) { const j = o + i * 4; w[i] = ((bytes[j] << 24) | (bytes[j + 1] << 16) | (bytes[j + 2] << 8) | bytes[j + 3]) >>> 0; }
    for (let i = 16; i < 64; i += 1) { const s0 = rightRotate(w[i - 15], 7) ^ rightRotate(w[i - 15], 18) ^ (w[i - 15] >>> 3); const s1 = rightRotate(w[i - 2], 17) ^ rightRotate(w[i - 2], 19) ^ (w[i - 2] >>> 10); w[i] = (w[i - 16] + s0 + w[i - 7] + s1) >>> 0; }
    let [a,b,c,d,e,f,g,hh] = h;
    for (let i = 0; i < 64; i += 1) { const s1 = rightRotate(e, 6) ^ rightRotate(e, 11) ^ rightRotate(e, 25); const ch = (e & f) ^ ((~e) & g); const t1 = (hh + s1 + ch + k[i] + w[i]) >>> 0; const s0 = rightRotate(a, 2) ^ rightRotate(a, 13) ^ rightRotate(a, 22); const maj = (a & b) ^ (a & c) ^ (b & c); const t2 = (s0 + maj) >>> 0; hh = g; g = f; f = e; e = (d + t1) >>> 0; d = c; c = b; b = a; a = (t1 + t2) >>> 0; }
    h = h.map((v, i) => (v + [a,b,c,d,e,f,g,hh][i]) >>> 0);
  }
  return h.slice(0, 7).map((v) => v.toString(16).padStart(8, "0")).join("");
}
function escapeHTML(value) { return String(value ?? "").replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[char]); }
function bytefmt(value) { const u = ["B", "KiB", "MiB", "GiB", "TiB"]; let n = Number(value || 0), i = 0; while (n >= 1024 && i < u.length - 1) { n /= 1024; i += 1; } return `${n.toFixed(i ? 2 : 0)} ${u[i]}`; }
function quotaGiBValue(quota) { const value = Number(quota); if (value < 0) return "-1"; const gib = value / (1024 ** 3); return Number.isInteger(gib) ? String(gib) : String(Number(gib.toFixed(2))); }
function quotaBytesFromGiB(input) { const value = String(input).trim(); if (value === "-1") return -1; if (!/^(?:\d+(?:\.\d+)?|\.\d+)$/.test(value)) return null; const bytes = Math.round(Number(value) * (1024 ** 3)); return Number.isSafeInteger(bytes) ? bytes : null; }
function chinaDateValue(date) {
  const parts = new Intl.DateTimeFormat("en-US", { timeZone: "Asia/Shanghai", year: "numeric", month: "2-digit", day: "2-digit" }).formatToParts(date);
  const values = Object.fromEntries(parts.map((part) => [part.type, part.value]));
  return `${values.year}-${values.month}-${values.day}`;
}
function shiftHistoryDate(value, { days = 0, years = 0 } = {}) { const [year, month, day] = value.split("-").map(Number); const date = new Date(Date.UTC(year, month - 1, day)); if (years) date.setUTCFullYear(date.getUTCFullYear() + years); if (days) date.setUTCDate(date.getUTCDate() + days); return date.toISOString().slice(0, 10); }
function setDefaultHistoryDates() { const end = chinaDateValue(new Date()); if (!$("#historyStart").value) $("#historyStart").value = shiftHistoryDate(end, { days: -7 }); if (!$("#historyEnd").value) $("#historyEnd").value = end; }
function pct(used, limit) { return !limit ? 0 : Math.max(0, Math.min(100, (used / limit) * 100)); }
function b64(value) { return btoa(unescape(encodeURIComponent(value))); }
function passOf(user) { try { return decodeURIComponent(escape(atob(user.Password || ""))); } catch { return ""; } }
function form(data) { const body = new URLSearchParams(); Object.entries(data).forEach(([k, v]) => body.set(k, v)); return { method: "POST", credentials: "same-origin", headers: { "Content-Type": "application/x-www-form-urlencoded" }, body }; }
async function api(path, options) { const r = await fetch(path, { credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" }, ...options }); if (r.status === 401) { showLogin(false); throw new Error("未登录或登录已过期"); } if (r.status === 403) throw new Error("当前账号无权访问该功能"); const b = await r.json(); if (b.Msg && b.Msg !== "success") throw new Error(b.Msg); return b.Data || b.data || {}; }
function startAutoRefresh() {
  clearInterval(state.refreshTimer);
  const interval = state.tab === "ops" ? 15000 : (state.tab === "system" || state.tab === "home" || state.tab === "service" ? 5000 : 0);
  if (interval) state.refreshTimer = setInterval(() => { if (document.visibilityState !== "hidden") refresh(false); }, interval);
}

function formatUptime(value) {
  if (!value) return "-";
  const raw = String(value).trim();
  const daySplit = raw.split("-");
  const days = daySplit.length === 2 ? Number(daySplit[0] || 0) : 0;
  const timeParts = (daySplit.length === 2 ? daySplit[1] : daySplit[0]).split(":").map((i) => Number(i));
  if (timeParts.some((i) => Number.isNaN(i)) || timeParts.length < 2 || timeParts.length > 3 || Number.isNaN(days)) return raw;
  const hours = timeParts.length === 3 ? timeParts[0] : 0;
  const minutes = timeParts.length === 3 ? timeParts[1] : timeParts[0];
  const labels = [];
  if (days) labels.push(`${days}天`);
  if (hours || days) labels.push(`${hours}小时`);
  labels.push(`${minutes}分`);
  return labels.join("");
}
function applyTheme() { document.documentElement.dataset.theme = state.theme; const label = state.theme === "dark" ? "Light" : "暗黑"; $("#themeBtnInline").textContent = label; localStorage.setItem("theme", state.theme); }
function switchTheme() { state.theme = state.theme === "dark" ? "light" : "dark"; applyTheme(); }
function switchTab(tab) { state.tab = tab; $$(".tab").forEach((i) => i.classList.toggle("active", i.dataset.tab === tab)); $$(".view").forEach((i) => i.classList.toggle("active", i.dataset.view === tab)); $("#pageTitle").textContent = titles[tab][0]; $("#pageSub").textContent = titles[tab][1]; if (window.OpsMonitor) window.OpsMonitor.setActive(tab === "ops"); startAutoRefresh(); refresh(true); }
function showDashboard() { $("#authScreen").hidden = true; $("#appShell").hidden = false; $("#dashboard").hidden = false; setLoginNotice(""); startAutoRefresh(); refresh(true); }
function showLogin(isRegister, title) {
  clearInterval(state.refreshTimer);
  $("#appShell").hidden = true;
  $("#dashboard").hidden = true;
  $("#authScreen").hidden = false;
  $("#loginTitle").textContent = isRegister ? "初始化管理员" : (title || state.loginTitle || "登录");
  $("#loginBtn").textContent = isRegister ? "初始化" : "登录";
  $("#loginForm").dataset.mode = isRegister ? "register" : "login";
}
function setNotice(message) { const n = $("#notice"); if (!n) return; n.hidden = !message; n.textContent = message || ""; }
function setSystemNotice(message) { const n = $("#systemNotice"); if (!n) return; n.hidden = !message; n.textContent = message || ""; }
function setLoginNotice(message) { const n = $("#loginNotice"); if (!n) return; n.hidden = !message; n.textContent = message || ""; }
function setServiceNotice(message, type = "") {
  const n = $("#serviceNotice");
  if (!n) return;
  n.hidden = !message;
  n.textContent = message || "";
  n.className = `notice ${type}`.trim();
}

async function loadVersion() { state.version = await api("/common/version"); $("#uptimeCard").textContent = formatUptime(state.version.trojanUptime); }
function renderActive(data) {
  const users = data.onlineUsers || [];
  $("#onlineCount").textContent = users.length;
  if (!state.overview) $("#totalUsersCard").textContent = `共 ${(data.users || []).length} 个账号`;
  $("#onlineUpdatedAt").textContent = `更新于 ${new Date().toLocaleTimeString()} · 每 5 秒自动刷新`;
  setNotice(data.apiError || "");
  $("#userRows").innerHTML = users.length ? users.map((u) => `<tr><td>${escapeHTML(u.username)}</td><td><span class="badge online">在线</span></td><td>${u.ipCurrent}</td><td>${bytefmt(u.uploadSpeed)}/s</td><td>${bytefmt(u.downloadSpeed)}/s</td><td>${bytefmt(u.uploadTraffic)}</td><td>${bytefmt(u.downloadTraffic)}</td></tr>`).join("") : '<tr><td colspan="7" class="empty">暂无在线用户</td></tr>';
}
function formatCollectedAt(value) {
  if (!value) return "尚未收集";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
function renderDashboardOverview(data) {
  if (!data) return;
  state.overview = data;
  const service = data.service || {}, accounts = data.accounts || {}, traffic = data.traffic || {}, today = traffic.today || {}, cert = data.certificate || {};
  $("#serviceStateCard").textContent = service.running ? "运行中" : (service.state === "failed" ? "异常" : "已停止");
  $("#serviceStateCard").style.color = service.running ? "var(--ok)" : "var(--danger)";
  $("#uptimeCard").textContent = `${service.type || "Trojan"} · 已运行 ${formatUptime(service.uptime)}`;
  $("#totalUsersCard").textContent = `共 ${accounts.total ?? "-"} 个账号`;
  $("#todayTraffic").textContent = bytefmt(today.total);
  $("#todayTrafficSplit").textContent = `上传 ${bytefmt(today.upload)} · 下载 ${bytefmt(today.download)}`;
  $("#todayRequests").textContent = Number(today.requests || 0).toLocaleString();
  $("#trafficLastCollect").textContent = `统计至 ${formatCollectedAt(traffic.lastCollect)}`;
  $("#accountNormal").textContent = accounts.normal ?? "-";
  $("#accountExpiring").textContent = accounts.expiringSoon ?? "-";
  $("#accountExpired").textContent = accounts.expired ?? "-";
  $("#accountExhausted").textContent = accounts.trafficExhausted ?? "-";
  $("#dashboardUpdatedAt").textContent = `概览更新于 ${formatCollectedAt(data.updatedAt)}`;
  const alert = $("#dashboardAlert");
  if (cert.error) {
    alert.hidden = false;
    alert.textContent = `TLS 证书状态读取失败：${cert.error}`;
  } else if (cert.warning) {
    alert.hidden = false;
    alert.textContent = `TLS 证书将在 ${cert.daysLeft} 天后到期，请在系统监控中安排续签。`;
  } else {
    alert.hidden = true;
    alert.textContent = "";
  }
}
function shareFor(user) {
  const pass = passOf(user), remark = encodeURIComponent(`${state.domain}:${state.port}`);
  const token = b64(JSON.stringify({ user: user.Username, pass }));
  return {
    trojan: `trojan://${encodeURIComponent(pass)}@${state.domain}:${state.port}#${remark}`,
    clash: `${location.origin}/trojan/user/subscribe?token=${encodeURIComponent(token)}`,
    ss: `ss://${b64(`aes-128-gcm:${pass}@${state.domain}:${state.port}`)}#${remark}`,
  };
}
function renderUsers(data) {
  state.users = data.userList || []; state.domain = data.domain || location.hostname; state.port = data.port || 443;
  $("#allUserRows").innerHTML = state.users.length ? state.users.map((u) => `<tr><td>${u.ID}</td><td>${u.Username}</td><td>${passOf(u)}</td><td>${bytefmt(u.Upload)}</td><td>${bytefmt(u.Download)}</td><td>${bytefmt(Number(u.Upload || 0) + Number(u.Download || 0))}</td><td>${u.Quota < 0 ? "无限制" : bytefmt(u.Quota)}</td><td>${u.ExpiryDate || "无限制"}</td><td><div class="actionRow"><button data-user-action="edit" data-id="${u.ID}" class="ghost">编辑</button><button data-user-action="quota" data-id="${u.ID}" class="ghost">限流</button><button data-user-action="expire" data-id="${u.ID}" class="ghost">期限</button><button data-user-action="clean" data-id="${u.ID}" class="ghost">清流量</button><button data-user-action="share" data-id="${u.ID}" class="ghost">分享</button><button data-user-action="delete" data-id="${u.ID}" class="ghost">删除</button></div></td></tr>`).join("") : '<tr><td colspan="9" class="empty">暂无用户</td></tr>';
}
async function loadUsers() {
  const users = await api(`/trojan/user?_=${Date.now()}`);
  renderUsers(users);
}
function fillHistoryUsers(users) {
  const current = $("#historyUser").value;
  $("#historyUser").innerHTML = '<option value="">全部用户</option>' + users.map((u) => `<option value="${u.EncryptPass}">${u.Username}</option>`).join("");
  $("#historyUser").value = current;
}
function historyParams(extra = {}) {
  const params = new URLSearchParams(new FormData($("#historyFilter")));
  Object.entries(extra).forEach(([k, v]) => params.set(k, v));
  return params;
}
function historyDateParts(value) {
  const match = String(value || "").match(/^(\d{4})-(\d{2})-(\d{2})$/);
  return match ? { year: Number(match[1]), month: Number(match[2]), day: Number(match[3]) } : null;
}
function historyMonthValue(year, month) { return `${year}-${String(month).padStart(2, "0")}`; }
function closeHistoryDatePicker() {
  $("#historyDatePicker").hidden = true;
  $$('[data-history-date-input]').forEach((input) => input.setAttribute("aria-expanded", "false"));
}
function renderHistoryDatePicker() {
  const picker = state.history.datePicker;
  const [year, month] = picker.month.split("-").map(Number);
  if (!year || !month) return;
  $("#historyDatePickerTitle").textContent = picker.active === "end" ? "选择结束日期" : "选择开始日期";
  $("#historyDatePickerMonth").textContent = `${year}年${String(month).padStart(2, "0")}月`;
  const first = new Date(Date.UTC(year, month - 1, 1));
  const gridStart = new Date(first);
  gridStart.setUTCDate(1 - first.getUTCDay());
  const start = $("#historyStart").value;
  const end = $("#historyEnd").value;
  const today = chinaDateValue(new Date());
  const rangeStart = start && end && start <= end ? start : "";
  const rangeEnd = rangeStart ? end : "";
  const days = [];
  for (let index = 0; index < 42; index += 1) {
    const date = new Date(gridStart);
    date.setUTCDate(gridStart.getUTCDate() + index);
    const value = date.toISOString().slice(0, 10);
    const classes = ["historyDateDay"];
    if (date.getUTCMonth() !== month - 1) classes.push("outside");
    if (value === today) classes.push("today");
    if (value === start) classes.push("rangeStart");
    if (value === end) classes.push("rangeEnd");
    if (rangeStart && value > rangeStart && value < rangeEnd) classes.push("inRange");
    const selected = value === (picker.active === "end" ? end : start);
    days.push(`<button class="${classes.join(" ")}" type="button" data-picker-date="${value}" aria-label="${value}" aria-selected="${selected}">${date.getUTCDate()}</button>`);
  }
  $("#historyDateGrid").innerHTML = days.join("");
}
function openHistoryDatePicker(field) {
  const input = field === "end" ? $("#historyEnd") : $("#historyStart");
  const fallback = chinaDateValue(new Date());
  state.history.datePicker.active = field === "end" ? "end" : "start";
  state.history.datePicker.month = (historyDateParts(input.value) ? input.value : fallback).slice(0, 7);
  $$('[data-history-date-input]').forEach((item) => item.setAttribute("aria-expanded", String(item === input)));
  $("#historyDatePicker").hidden = false;
  renderHistoryDatePicker();
}
function moveHistoryDateMonth(offset) {
  const [year, month] = state.history.datePicker.month.split("-").map(Number);
  const date = new Date(Date.UTC(year, month - 1 + offset, 1));
  state.history.datePicker.month = historyMonthValue(date.getUTCFullYear(), date.getUTCMonth() + 1);
  renderHistoryDatePicker();
}
function setHistoryDate(value) {
  const active = state.history.datePicker.active;
  const input = active === "end" ? $("#historyEnd") : $("#historyStart");
  input.value = value;
  if (value && active === "start" && $("#historyEnd").value && value > $("#historyEnd").value) $("#historyEnd").value = value;
  if (value && active === "end" && $("#historyStart").value && value < $("#historyStart").value) $("#historyStart").value = value;
  input.dispatchEvent(new Event("change", { bubbles: true }));
  closeHistoryDatePicker();
}
function setHistoryRange(range) {
  const end = chinaDateValue(new Date());
  $("#historyEnd").value = end;
  $("#historyStart").value = range === "year" ? shiftHistoryDate(end, { years: -1 }) : (range === "30" ? shiftHistoryDate(end, { days: -29 }) : end);
  closeHistoryDatePicker();
  $("#historyFilter").requestSubmit();
}
function historyClientIPs(item) {
  const source = Array.isArray(item.clientIPs) && item.clientIPs.length ? item.clientIPs : [item.clientIP];
  return [...new Set(source.map((ip) => String(ip || "").trim()).filter(Boolean))];
}
function renderHistoryClientIPs(item) {
  const ips = historyClientIPs(item);
  if (!ips.length) return '<span class="historyIPValue">-</span>';
  if (ips.length === 1) return `<span class="historyIPValue">${escapeHTML(ips[0])}</span>`;
  const payload = encodeURIComponent(JSON.stringify(ips));
  return `<button class="historyIPSummary" type="button" data-client-ips="${payload}" aria-describedby="historyIPTooltip"><span>${escapeHTML(ips[0])}</span><em>+${ips.length - 1}</em></button>`;
}
function historySortState(userHash) { return state.history.sorts[userHash] || { by: "accessCount", dir: "desc" }; }
function renderHistorySortHeader(group, sortBy, label) {
  const sort = historySortState(group.userHash);
  const active = sort.by === sortBy;
  const direction = active ? sort.dir : "none";
  const arrow = active ? (sort.dir === "desc" ? "↓" : "↑") : "↕";
  return `<th aria-sort="${direction === "desc" ? "descending" : (direction === "asc" ? "ascending" : "none")}"><button class="historySortButton${active ? " active" : ""}" type="button" data-history-sort="${group.userHash}" data-sort-by="${sortBy}">${label}<span aria-hidden="true">${arrow}</span></button></th>`;
}
let historyIPTooltipTimer = 0;
function showHistoryIPTooltip(anchor) {
  let ips;
  try { ips = JSON.parse(decodeURIComponent(anchor.dataset.clientIps || "")); } catch { return; }
  if (!Array.isArray(ips) || ips.length < 2) return;
  const tooltip = $("#historyIPTooltip");
  clearTimeout(historyIPTooltipTimer);
  tooltip.innerHTML = `<strong>客户端 IP · ${ips.length}</strong><div>${ips.map((ip) => `<code>${escapeHTML(ip)}</code>`).join("")}</div>`;
  tooltip.hidden = false;
  tooltip.classList.remove("visible");
  if (!anchor.isConnected) return;
  const anchorRect = anchor.getBoundingClientRect();
  const tooltipRect = tooltip.getBoundingClientRect();
  const left = Math.max(12, Math.min(anchorRect.left, window.innerWidth - tooltipRect.width - 12));
  let top = anchorRect.bottom + 8;
  if (top + tooltipRect.height > window.innerHeight - 12) top = Math.max(12, anchorRect.top - tooltipRect.height - 8);
  tooltip.style.left = `${Math.round(left)}px`;
  tooltip.style.top = `${Math.round(top)}px`;
  tooltip.classList.add("visible");
}
function hideHistoryIPTooltip() {
  const tooltip = $("#historyIPTooltip");
  if (tooltip.hidden) return;
  tooltip.classList.remove("visible");
  clearTimeout(historyIPTooltipTimer);
  historyIPTooltipTimer = window.setTimeout(() => { tooltip.hidden = true; }, 150);
}
function renderHistoryDetail(group) {
  const detail = state.history.details[group.userHash];
  if (!detail) return "";
  const totalPages = Math.max(1, Math.ceil((detail.total || 0) / detail.pageSize));
  const rows = (detail.items || []).map((i) => `<tr><td>${escapeHTML(i.targetHost)}</td><td>${i.accessCount}</td><td class="historyTraffic">${bytefmt(i.totalTraffic ?? (Number(i.upload || 0) + Number(i.download || 0)))}</td><td>${renderHistoryClientIPs(i)}</td></tr>`).join("");
  return `<div class="historyDetail">
    <div class="historyDetailHead">
      <span>第 ${detail.page} / ${totalPages} 页，共 ${detail.total} 条目标记录</span>
      <select data-history-size="${group.userHash}"><option value="50"${detail.pageSize === 50 ? " selected" : ""}>每页 50</option><option value="100"${detail.pageSize === 100 ? " selected" : ""}>每页 100</option></select>
    </div>
    <div class="tableWrap"><table><thead><tr><th>目标域名/IP</th>${renderHistorySortHeader(group, "accessCount", "访问次数")}${renderHistorySortHeader(group, "totalTraffic", "总流量")}<th>客户端IP</th></tr></thead><tbody>${rows || '<tr><td colspan="4" class="empty">暂无访问记录</td></tr>'}</tbody></table></div>
    <div class="pager"><button class="ghost small" data-history-page="${group.userHash}" data-page="${Math.max(1, detail.page - 1)}" ${detail.page <= 1 ? "disabled" : ""}>上一页</button><button class="ghost small" data-history-page="${group.userHash}" data-page="${Math.min(totalPages, detail.page + 1)}" ${detail.page >= totalPages ? "disabled" : ""}>下一页</button></div>
  </div>`;
}
function renderHistory(data) {
  const groups = data.groups || [];
  state.history.groups = groups;
  state.history.meta = { startDate: data.startDate || state.history.meta.startDate, endDate: data.endDate || state.history.meta.endDate, lastCollect: data.lastCollect || state.history.meta.lastCollect };
  $("#historyMeta").textContent = `统计区间 ${data.startDate || "-"} 至 ${data.endDate || "-"}，最后收集 ${data.lastCollect ? new Date(data.lastCollect).toLocaleString() : "未执行"}`;
  $("#historyGroups").innerHTML = groups.length ? groups.map((g) => {
    const open = state.history.expanded === g.userHash;
    return `<section class="historyGroup">
      <button class="historyGroupHead" data-history-toggle="${g.userHash}" type="button">
        <span class="chevron">${open ? "v" : ">"}</span>
        <strong>${escapeHTML(g.username || g.userHash)}</strong>
        <span>${g.accessCount} 次访问</span>
        <span>${g.targetCount} 个目标</span>
        <span>${g.clientIPCount} 个客户端IP</span>
        <span class="historyTraffic">${bytefmt(g.totalTraffic)} 总流量</span>
      </button>
      ${open ? renderHistoryDetail(g) : ""}
    </section>`;
  }).join("") : '<div class="empty historyEmpty">暂无访问记录</div>';
}
async function loadHistory() {
  setDefaultHistoryDates();
  if (!state.users.length) {
    const users = await api("/trojan/user");
    state.users = users.userList || [];
  }
  fillHistoryUsers(state.users);
  const data = await api(`/trojan/access-history?${historyParams().toString()}`);
  state.history.expanded = "";
  state.history.details = {};
  renderHistory(data);
}
async function loadHistoryDetail(userHash, page) {
  const pageSize = state.history.pageSizes[userHash] || 50;
  const sort = historySortState(userHash);
  state.history.pages[userHash] = page;
  state.history.expanded = userHash;
  const params = historyParams({ detailUserHash: userHash, page, pageSize, sortBy: sort.by, sortDir: sort.dir });
  const data = await api(`/trojan/access-history?${params.toString()}`);
  if (data.detail && data.detail.sortBy) state.history.sorts[userHash] = { by: data.detail.sortBy, dir: data.detail.sortDir || "desc" };
  state.history.details[userHash] = data.detail;
  state.history.groups = data.groups || state.history.groups;
  renderHistory(data);
}
function renderService(data) {
  state.service = data || {};
  const mux = data.mux || {}, apiInfo = data.api || {};
  const running = data.trojanRunning === true || data.trojanState === "active";
  const stateText = running ? "代理服务运行中" : (data.trojanState === "failed" ? "代理服务异常" : "代理服务已停止");
  const stateSub = running ? `trojan.service active${data.trojanUptime ? `，已运行 ${formatUptime(data.trojanUptime)}` : ""}` : `trojan.service ${data.trojanState || "unknown"}，443 代理不可用`;
  $("#serviceStateText").textContent = stateText;
  $("#serviceStateSub").textContent = state.serviceBusy ? "正在执行操作，请等待状态刷新" : stateSub;
  $("#serviceDot").className = `statusDot ${state.serviceBusy ? "working" : (running ? "running" : (data.trojanState === "failed" ? "failed" : "stopped"))}`;
  $("#h2State").textContent = data.h2FullyEnabled ? "H2 已完整启用" : (data.h2Alpn ? "H2 ALPN 已启用" : "H2 未完整启用");
  $("#switchTypeBtn").textContent = `切换为 ${data.trojanType === "trojan-go" ? "trojan" : "trojan-go"}`;
  $("#switchTypeBtn").dataset.target = data.trojanType === "trojan-go" ? "trojan" : "trojan-go";
  $("#serviceInfo").innerHTML = [["服务状态", data.trojanState || "-"], ["运行时长", data.trojanUptime ? formatUptime(data.trojanUptime) : "-"], ["Trojan 类型", data.trojanType || "-"], ["ALPN", (data.alpn || []).join(", ") || "-"], ["H2 后端", data.h2Port ? `127.0.0.1:${data.h2Port} ${data.h2BackendReady ? "可用" : "不可用"}` : "-"], ["Mux", mux.Enabled || mux.enabled ? "已启用" : "未启用"], ["Mux 并发", mux.Concurrency || mux.concurrency || "-"], ["API", apiInfo.Enabled || apiInfo.enabled ? "已启用" : "未启用"], ["API 地址", `${apiInfo.APIAddr || apiInfo.api_addr || "127.0.0.1"}:${apiInfo.APIPort || apiInfo.api_port || 10000}`]].map(([k, v]) => `<div><span>${k}</span><b>${v}</b></div>`).join("");
  $$("[data-action]").forEach((btn) => {
    const action = btn.dataset.action;
    btn.disabled = Boolean(state.serviceBusy) || (action === "start" && running) || ((action === "stop" || action === "restart") && !running);
    btn.textContent = state.serviceBusy === action ? `${{ start: "启动", stop: "停止", restart: "重启" }[action]}中` : { start: running ? "已运行" : "启动", stop: "停止", restart: "重启" }[action];
  });
  $("#switchTypeBtn").disabled = Boolean(state.serviceBusy);
}
function pie(name, value, sub) { const safe = Math.max(0, Math.min(100, Number(value) || 0)); return `<div class="chartCard"><div class="pie" style="--value:${Math.round(safe)}"><span>${safe.toFixed(1)}%</span></div><h3>${name}</h3><p>${sub}</p></div>`; }
function trafficLevel(percent) {
  if (percent >= 100) return { key: "exhausted", text: "套餐已用尽" };
  if (percent >= 85) return { key: "warning", text: "接近套餐上限" };
  if (percent >= 70) return { key: "high", text: "使用偏高" };
  return { key: "normal", text: "用量正常" };
}
function renderMonthlyTraffic(data) {
  if (!data) return;
  const month = (data.traffic && data.traffic.month) || {}, capacity = data.capacity || {}, network = data.network || {};
  const percent = Number(capacity.usagePercent || 0), level = trafficLevel(percent), quota = Number(capacity.packageQuota || 0);
  $(".trafficPanel").dataset.level = level.key;
  $("#monthTrafficStatus").className = `trafficStatus ${level.key}`;
  $("#monthTrafficStatus").textContent = level.text;
  $("#monthTrafficPeriod").textContent = `${month.startDate || "-"} 至 ${month.endDate || "-"}`;
  $("#monthTrafficTotal").textContent = bytefmt(month.total);
  $("#monthTrafficMeter").style.width = `${Math.min(100, percent)}%`;
  $("#monthTrafficPercent").textContent = `${percent.toFixed(2)}%`;
  $("#monthTrafficRemaining").textContent = bytefmt(capacity.remaining);
  $("#monthTrafficAverage").textContent = `${bytefmt(capacity.dailyAverage)}/天`;
  $("#monthTrafficProjected").textContent = bytefmt(capacity.projectedTotal);
  $("#monthTrafficUpload").textContent = bytefmt(month.upload);
  $("#monthTrafficDownload").textContent = bytefmt(month.download);
  $("#monthTrafficUpdated").textContent = `按中国时区自然月统计；日均按已过 ${capacity.elapsedDays || "-"} 天计算。Trojan 数据统计至 ${formatCollectedAt(data.traffic && data.traffic.lastCollect)}。`;
  const networkPercent = quota ? (Number(network.total || 0) / quota) * 100 : 0;
  const coverageMode = network.coverageMode || "partial";
  const fullCoverage = coverageMode === "full";
  $("#networkTrafficLabel").textContent = "VPS 整机本月流量";
  $("#networkMonthTotal").textContent = network.ready ? bytefmt(network.total) : "等待采样";
  $("#networkMonthPercent").textContent = network.ready ? `占 2TB 套餐 ${networkPercent.toFixed(2)}%` : (network.error || "统计初始化中");
  $("#networkMonthUpload").textContent = bytefmt(network.upload);
  $("#networkMonthDownload").textContent = bytefmt(network.download);
  $(".trafficSecondary").classList.toggle("warning", networkPercent >= 85);
  const tracking = network.trackingSince ? formatCollectedAt(network.trackingSince) : "本功能上线后";
  const coverageText = fullCoverage ? "按公网网卡真实累计的完整自然月流量" : `自 ${tracking} 部署采集后按公网网卡真实累计；本月部署前的整机流量未纳入`;
  $("#networkTrackingSince").textContent = `${network.interface ? `${network.interface} · ` : ""}${coverageText}${network.error ? ` · ${network.error}` : ""}`;
}
function renderProcessMetrics(metrics) {
  const rows = Array.isArray(metrics) ? metrics : [];
  const tcp = rows.reduce((sum, item) => sum + Number(item.tcpConnections || 0), 0);
  const fd = rows.reduce((sum, item) => sum + Number(item.fdCount || 0), 0);
  const tcpLimit = rows.reduce((sum, item) => sum + Number(item.tcpLimit || 0), 0);
  const fdLimit = rows.reduce((sum, item) => sum + Number(item.fdLimit || 0), 0);
  const first = rows[0] || {};
  const tcpPercent = pct(tcp, tcpLimit), fdPercent = pct(fd, fdLimit);
  $("#tcpRatio").textContent = tcpLimit ? `${tcpPercent.toFixed(2)}%` : "-";
  $("#fdRatio").textContent = fdLimit ? `${fdPercent.toFixed(2)}%` : "-";
  $("#tcpMeter").style.width = `${tcpPercent}%`;
  $("#fdMeter").style.width = `${fdPercent}%`;
  $("#tcpLimitText").textContent = `${tcp.toLocaleString()} / ${(tcpLimit || 0).toLocaleString()}${first.tcpLimitSource ? ` · ${first.tcpLimitSource}` : ""}`;
  $("#fdLimitText").textContent = `${fd.toLocaleString()} / ${(fdLimit || 0).toLocaleString()} · PID ${rows.map((item) => item.pid).filter(Boolean).join(", ") || "-"}`;
  $("#kernelLimits").innerHTML = [["SYN backlog 上限", first.synBacklogLimit], ["TIME_WAIT bucket 上限", first.timeWaitBucketLimit], ["Conntrack 上限", first.conntrackLimit]].map(([key, value]) => `<div><span>${key}</span><b>${Number(value || 0).toLocaleString()}</b></div>`).join("");
}
function renderSystem(info, cert, overview, processMetrics) {
  const cpu = info.cpu && info.cpu[0] !== undefined ? info.cpu[0] : 0, mem = info.memory ? info.memory.usedPercent : 0, disk = info.disk ? info.disk.usedPercent : 0, swap = info.swap ? info.swap.usedPercent : 0;
  $("#chartGrid").innerHTML = pie("CPU", cpu, "处理器占用") + pie("内存", mem, info.memory ? `${bytefmt(info.memory.used)} / ${bytefmt(info.memory.total)}` : "-") + pie("磁盘", disk, info.disk ? `${bytefmt(info.disk.used)} / ${bytefmt(info.disk.total)}` : "-") + pie("Swap", swap, info.swap ? `${bytefmt(info.swap.used)} / ${bytefmt(info.swap.total)}` : "-");
  const load = info.load || {}, speed = info.speed || {}, net = info.netCount || {}, conntrack = info.conntrack || {};
  $("#realtimeInfo").innerHTML = [["当前上行", `${bytefmt(speed.Up)}/s`], ["当前下行", `${bytefmt(speed.Down)}/s`], ["系统负载", `${Number(load.load1 || 0).toFixed(2)} / ${Number(load.load5 || 0).toFixed(2)} / ${Number(load.load15 || 0).toFixed(2)}`], ["TCP / UDP", `${Number(net.tcp || 0).toLocaleString()} / ${Number(net.udp || 0).toLocaleString()}`], ["TIME_WAIT", Number(net.timeWait || 0).toLocaleString()], ["Conntrack", `${Number(conntrack.count || 0).toLocaleString()} / ${Number(conntrack.limit || 0).toLocaleString()}`]].map(([k, v]) => `<div><span>${k}</span><b>${v}</b></div>`).join("");
  renderMonthlyTraffic(overview);
  renderProcessMetrics(processMetrics);
  const certData = cert || {}, names = [...(certData.dnsNames || []), ...(certData.ipAddresses || [])].join(", ") || "-";
  $("#certInfo").innerHTML = [["服务运行时长", overview && overview.service ? formatUptime(overview.service.uptime) : "-"], ["证书到期", certData.notAfter ? new Date(certData.notAfter).toLocaleString() : "-"], ["剩余天数", certData.daysLeft !== undefined ? `${certData.daysLeft} 天` : "-"], ["签发域名", names], ["颁发者", certData.issuer || "-"], ["证书路径", certData.certPath || "-"]].map(([k, v]) => `<div><span>${k}</span><b>${v}</b></div>`).join("");
}
function setQR(img, text) {
  img.src = `/common/qrcode?data=${encodeURIComponent(text)}`;
}

async function refresh(force = false) {
  if (state.refreshing) return;
  state.refreshing = true;
  const tab = state.tab;
  try {
    if (tab === "home") {
      const needOverview = force || !state.overviewFetchedAt || Date.now() - state.overviewFetchedAt >= 60000;
      const [active, overview] = await Promise.all([api("/common/activeUsers"), needOverview ? api(`/common/overview${force ? "?refresh=1" : ""}`) : Promise.resolve(null)]);
      if (state.tab !== tab) return;
      if (overview) { state.overview = overview; state.overviewFetchedAt = Date.now(); renderDashboardOverview(overview); }
      renderActive(active);
    }
    if (tab === "ops") {
      await window.OpsMonitor.refresh(api, force);
      if (state.tab !== tab) return;
    }
    if (tab === "users") {
      await loadUsers();
      if (state.tab !== tab) return;
    }
    if (tab === "history") {
      await loadHistory();
      if (state.tab !== tab) return;
    }
    if (tab === "service") {
      const service = await api("/trojan/h2");
      if (state.tab !== tab) return;
      renderService(service);
    }
    if (tab === "system") {
      const needOverview = force || !state.overviewFetchedAt || Date.now() - state.overviewFetchedAt >= 60000;
      const needProcess = force || !state.processFetchedAt || Date.now() - state.processFetchedAt >= 15000;
      const needCert = force || !state.certFetchedAt || Date.now() - state.certFetchedAt >= 300000;
      const [info, overview, processMetrics, cert] = await Promise.all([
        api("/common/serverInfo"),
        needOverview ? api(`/common/overview${force ? "?refresh=1" : ""}`) : Promise.resolve(null),
        needProcess ? api("/common/processMetrics") : Promise.resolve(null),
        needCert ? api("/common/certInfo") : Promise.resolve(null),
      ]);
      if (state.tab !== tab) return;
      if (overview) { state.overview = overview; state.overviewFetchedAt = Date.now(); }
      if (processMetrics) { state.processMetrics = processMetrics; state.processFetchedAt = Date.now(); }
      if (cert) { state.cert = cert; state.certFetchedAt = Date.now(); }
      renderSystem(info, state.cert, state.overview, state.processMetrics);
      setSystemNotice("");
    }
  } catch (e) {
    if (tab === "ops" && window.OpsMonitor) window.OpsMonitor.showError(e.message);
    else if (tab === "system") setSystemNotice(e.message);
    else setNotice(e.message);
  } finally {
    state.refreshing = false;
  }
}
function openUserDialog(user) { $("#userDialogTitle").textContent = user ? "编辑用户" : "新增用户"; $("#userId").value = user ? user.ID : ""; $("#userNameInput").value = user ? user.Username : ""; $("#userPassInput").value = user ? passOf(user) : ""; $("#userDialog").showModal(); }
async function handleUserAction(action, id) {
  const user = state.users.find((u) => String(u.ID) === String(id)); if (!user) return;
  if (action === "edit") return openUserDialog(user);
  if (action === "quota") {
    const value = prompt("请输入流量限额（GiB，-1 表示无限制）", quotaGiBValue(user.Quota));
    if (value === null) return;
    const quota = quotaBytesFromGiB(value);
    if (quota === null) { alert("请输入有效的 GiB 数值，或输入 -1 表示无限制。"); return; }
    await api("/trojan/data", form({ id, quota }));
  }
  if (action === "expire") { const d = prompt("请输入可使用天数，0 表示取消期限", user.UseDays || 0); if (d !== null && Number(d) === 0) await api(`/trojan/user/expire?id=${id}`, { method: "DELETE" }); else if (d !== null) await api("/trojan/user/expire", form({ id, useDays: d })); }
  if (action === "clean" && confirm(`清空 ${user.Username} 的流量统计？`)) await api(`/trojan/data?id=${id}`, { method: "DELETE" });
  if (action === "delete" && confirm(`删除用户 ${user.Username}？`)) {
    const users = await api(`/trojan/user?id=${id}`, { method: "DELETE" });
    renderUsers(users);
    return;
  }
  if (action === "share") { const s = shareFor(user); $("#trojanShare").value = s.trojan; $("#clashShare").value = s.clash; $("#ssShare").value = s.ss; setQR($("#trojanQr"), s.trojan); setQR($("#clashQr"), s.clash); setQR($("#ssQr"), s.ss); $("#shareDialog").showModal(); return; }
  refresh();
}
function startLog() {
  const box = $("#logBox"); box.textContent = "连接日志中...\n";
  const ws = new WebSocket(`${location.protocol === "https:" ? "wss" : "ws"}://${location.host}/trojan/log?line=300`);
  ws.onmessage = (event) => { box.textContent += event.data; box.scrollTop = box.scrollHeight; };
  ws.onerror = () => { box.textContent += "\n日志连接失败\n"; };
}
async function boot() {
  applyTheme();
  try {
    const r = await fetch("/auth/check", { credentials: "same-origin" });
    if (r.status === 201) return showLogin(true);
    let title = "登录";
    if (r.ok) {
      const body = await r.json();
      title = (body.data && body.data.title) || title;
    } else {
      let message = "无法确认登录状态，请稍后重试";
      try {
        const body = await r.json();
        message = body.message || body.Msg || message;
      } catch {}
      showLogin(false, title);
      setLoginNotice(message);
      return;
    }
    state.loginTitle = title;
    const u = await fetch("/auth/loginUser", { credentials: "same-origin", cache: "no-store" }).catch(() => null);
    if (!u || !u.ok) return showLogin(false, title);
    const session = await u.json();
    if (session.data && session.data.role === "user") {
      location.replace("/portal");
      return;
    }
    return showDashboard();
  } catch {
    return showLogin(false);
  }
}

async function finishLogin() {
  const response = await fetch("/auth/loginUser", { credentials: "same-origin", cache: "no-store" });
  if (!response.ok) throw new Error("无法读取登录状态");
  const session = await response.json();
  if (session.data && session.data.role === "user") {
    location.replace("/portal");
    return;
  }
  showDashboard();
}
async function logout() {
  const btn = $("#logoutBtn");
  btn.disabled = true;
  try {
    await fetch("/auth/logout", { method: "POST", credentials: "same-origin" });
  } finally {
    btn.disabled = false;
    $("#password").value = "";
    showLogin(false);
  }
}

$("#loginForm").addEventListener("submit", async (e) => {
  e.preventDefault();
  setLoginNotice("");
  const isRegister = $("#loginForm").dataset.mode === "register";
  const f = new FormData($("#loginForm"));
  f.set("password", sha224(f.get("password") || ""));
  const r = await fetch(isRegister ? "/auth/register" : "/auth/login", { method: "POST", credentials: "same-origin", body: f });
  if (!r.ok) {
    setLoginNotice("用户名或密码错误");
    return;
  }
  if (isRegister) {
    $("#username").value = "admin";
    $("#password").value = "";
    showLogin(false);
    setLoginNotice("管理员初始化完成，请登录");
    return;
  }
  try { await finishLogin(); } catch (error) { setLoginNotice(error.message); }
});
$("#logoutBtn").addEventListener("click", logout);
$("#themeBtnInline").addEventListener("click", switchTheme);
$("#addUserBtn").addEventListener("click", () => openUserDialog());
$("#userForm").addEventListener("submit", async (e) => { e.preventDefault(); const id = $("#userId").value, username = $("#userNameInput").value.trim(), password = b64($("#userPassInput").value); await api(id ? "/trojan/user/update" : "/trojan/user", form(id ? { id, username, password } : { username, password })); $("#userDialog").close(); refresh(); });
$("#historyFilter").addEventListener("submit", async (e) => { e.preventDefault(); closeHistoryDatePicker(); state.history.expanded = ""; state.history.details = {}; await loadHistory(); });
$$('[data-history-date-input]').forEach((input) => {
  input.addEventListener("click", () => openHistoryDatePicker(input.dataset.historyDateInput));
  input.addEventListener("keydown", (e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); openHistoryDatePicker(input.dataset.historyDateInput); } });
});
$("#historyDatePicker").addEventListener("click", (e) => {
  const range = e.target.closest("[data-history-range]");
  if (range) return setHistoryRange(range.dataset.historyRange);
  const month = e.target.closest("[data-picker-month]");
  if (month) return moveHistoryDateMonth(Number(month.dataset.pickerMonth));
  const day = e.target.closest("[data-picker-date]");
  if (day) return setHistoryDate(day.dataset.pickerDate);
  const action = e.target.closest("[data-picker-action]");
  if (!action) return;
  if (action.dataset.pickerAction === "clear") setHistoryDate("");
});
document.addEventListener("click", (e) => { if (!e.target.closest(".historyDateRangeControl")) closeHistoryDatePicker(); });
document.addEventListener("keydown", (e) => { if (e.key === "Escape") { closeHistoryDatePicker(); hideHistoryIPTooltip(); } });
$("#collectHistoryBtn").addEventListener("click", async () => { const btn = $("#collectHistoryBtn"); btn.disabled = true; btn.textContent = "收集中"; try { await api("/trojan/access-history/collect", { method: "POST" }); await loadHistory(); } finally { btn.disabled = false; btn.textContent = "收集统计"; } });
$$("[data-close]").forEach((b) => b.addEventListener("click", () => b.closest("dialog").close()));
$("#allUserRows").addEventListener("click", (e) => { const b = e.target.closest("[data-user-action]"); if (b) handleUserAction(b.dataset.userAction, b.dataset.id); });
$("#historyGroups").addEventListener("click", async (e) => {
  const toggle = e.target.closest("[data-history-toggle]");
  if (toggle) {
    const userHash = toggle.dataset.historyToggle;
    if (state.history.expanded === userHash) {
      state.history.expanded = "";
      renderHistory({ groups: state.history.groups, ...state.history.meta });
    } else {
      await loadHistoryDetail(userHash, state.history.pages[userHash] || 1);
    }
    return;
  }
  const sortButton = e.target.closest("[data-history-sort]");
  if (sortButton) {
    const userHash = sortButton.dataset.historySort;
    const sortBy = sortButton.dataset.sortBy;
    const current = historySortState(userHash);
    state.history.sorts[userHash] = { by: sortBy, dir: current.by === sortBy && current.dir === "desc" ? "asc" : "desc" };
    await loadHistoryDetail(userHash, 1);
    return;
  }
  const pageBtn = e.target.closest("[data-history-page]");
  if (pageBtn && !pageBtn.disabled) await loadHistoryDetail(pageBtn.dataset.historyPage, Number(pageBtn.dataset.page || 1));
});
$("#historyGroups").addEventListener("change", async (e) => {
  const select = e.target.closest("[data-history-size]");
  if (!select) return;
  const userHash = select.dataset.historySize;
  state.history.pageSizes[userHash] = Number(select.value);
  await loadHistoryDetail(userHash, 1);
});
$("#historyGroups").addEventListener("pointerover", (e) => { const anchor = e.target.closest("[data-client-ips]"); if (anchor && !anchor.contains(e.relatedTarget)) showHistoryIPTooltip(anchor); });
$("#historyGroups").addEventListener("pointerout", (e) => { const anchor = e.target.closest("[data-client-ips]"); if (anchor && !anchor.contains(e.relatedTarget)) hideHistoryIPTooltip(); });
$("#historyGroups").addEventListener("focusin", (e) => { const anchor = e.target.closest("[data-client-ips]"); if (anchor) showHistoryIPTooltip(anchor); });
$("#historyGroups").addEventListener("focusout", (e) => { const anchor = e.target.closest("[data-client-ips]"); if (anchor && !anchor.contains(e.relatedTarget)) hideHistoryIPTooltip(); });
window.addEventListener("scroll", hideHistoryIPTooltip, { passive: true, capture: true });
window.addEventListener("resize", hideHistoryIPTooltip, { passive: true });
async function runServiceAction(action, button) {
  const labels = { start: "启动", stop: "停止", restart: "重启" };
  const running = state.service.trojanRunning === true || state.service.trojanState === "active";
  if (action === "start" && running) {
    setServiceNotice("trojan.service 已在运行，无需重复启动。", "ok");
    return;
  }
  if ((action === "stop" || action === "restart") && !running) {
    setServiceNotice("trojan.service 当前未运行，不能执行该操作。", "error");
    return;
  }
  if (action === "stop" && !confirm("停止会立即关闭 443 代理服务，确认停止 trojan.service？")) return;
  if (action === "restart" && !confirm("重启会短暂中断 443 代理连接，确认重启 trojan.service？")) return;
  state.serviceBusy = action;
  setServiceNotice(`正在${labels[action]} trojan.service...`);
  renderService(state.service);
  try {
    await api(`/trojan/${action}`, { method: "POST" });
    await refresh();
    setServiceNotice(`${labels[action]}操作已提交，当前状态已刷新。`, "ok");
  } catch (e) {
    setServiceNotice(`${labels[action]}失败：${e.message}`, "error");
  } finally {
    state.serviceBusy = "";
    button.blur();
    await refresh();
  }
}
$$("[data-action]").forEach((b) => b.addEventListener("click", () => runServiceAction(b.dataset.action, b)));
$("#switchTypeBtn").addEventListener("click", async () => {
  const target = $("#switchTypeBtn").dataset.target;
  if (!confirm(`切换会重新安装/重建代理程序并重启服务，确认切换为 ${target}？`)) return;
  state.serviceBusy = "switch";
  setServiceNotice(`正在切换为 ${target}，请等待...`);
  renderService(state.service);
  try {
    await api("/trojan/switch", form({ type: target }));
    await refresh();
    setServiceNotice(`已切换为 ${target}。`, "ok");
  } catch (e) {
    setServiceNotice(`切换失败：${e.message}`, "error");
  } finally {
    state.serviceBusy = "";
    await refresh();
  }
});
$("#logBtn").addEventListener("click", startLog);
$("#renewCertBtn").addEventListener("click", async () => { if (confirm("证书续签会临时重启 trojan-web，确认执行？")) { await api("/common/cert/renew", { method: "POST" }); alert("续签任务已启动，日志见 /tmp/trojan-cert-renew.log"); } });
$$(".tab").forEach((i) => i.addEventListener("click", () => switchTab(i.dataset.tab)));
boot();
