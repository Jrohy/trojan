const $ = (selector) => document.querySelector(selector);
const themeMedia = window.matchMedia("(prefers-color-scheme: dark)");
const statusClasses = new Set(["normal", "expiring_soon", "expired", "traffic_exhausted"]);
let portalTheme = localStorage.getItem("portalTheme") || "system";
let loading = false;
let refreshTimer = 0;

function resolvedTheme() {
  if (portalTheme === "system") return themeMedia.matches ? "dark" : "light";
  return portalTheme === "dark" ? "dark" : "light";
}

function applyTheme() {
  document.documentElement.dataset.theme = resolvedTheme();
  const option = document.querySelector(`input[name="portalTheme"][value="${portalTheme}"]`);
  if (option) option.checked = true;
  localStorage.setItem("portalTheme", portalTheme);
}

function bytefmt(value) {
  const units = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];
  let number = Number(value || 0);
  let index = 0;
  while (number >= 1024 && index < units.length - 1) {
    number /= 1024;
    index += 1;
  }
  return `${number.toFixed(index ? 2 : 0)} ${units[index]}`;
}

function formatExpiry(value) {
  if (!value) return "长期有效";
  const [year, month, day] = value.split("-").map(Number);
  if (!year || !month || !day) return value;
  return `${year} 年 ${String(month).padStart(2, "0")} 月 ${String(day).padStart(2, "0")} 日`;
}

function formatRemaining(days) {
  if (days === null || days === undefined) return "长期有效";
  if (days < 0) return "已过期";
  if (days === 0) return "今天到期";
  return `还有 ${days} 天`;
}

function setLoadingState(state) {
  $("#portalLoading").hidden = state !== "loading";
  $("#portalError").hidden = state !== "error";
  $("#portalOverview").hidden = state !== "ready";
}

function renderOverview(data) {
  const traffic = data.traffic || {};
  const status = statusClasses.has(data.status) ? data.status : "normal";
  $("#overviewUsername").textContent = data.username || "—";
  $("#headerUsername").textContent = data.username || "";
  $("#settingsUsername").textContent = data.username || "—";
  const pill = $("#accountStatus");
  pill.className = `statusPill ${status}`;
  pill.querySelector("b").textContent = data.statusText || "账号正常";

  $("#trafficUsed").textContent = bytefmt(traffic.used);
  $("#trafficUpload").textContent = bytefmt(traffic.upload);
  $("#trafficDownload").textContent = bytefmt(traffic.download);
  const track = $("#trafficTrack");
  const progress = $("#trafficProgress");
  if (traffic.unlimited) {
    $("#trafficQuota").textContent = "/ 不限流量";
    $("#trafficRatio").textContent = "不限流量";
    $("#trafficRemaining").textContent = "不限流量";
    track.classList.add("unlimited");
    track.removeAttribute("aria-valuenow");
    track.setAttribute("aria-valuetext", "不限流量");
    progress.style.width = "100%";
  } else {
    const ratio = Math.max(0, Math.min(100, Number(traffic.usageRatio || 0)));
    $("#trafficQuota").textContent = `/ ${bytefmt(traffic.quota)}`;
    $("#trafficRatio").textContent = `已用 ${ratio.toFixed(1)}%`;
    $("#trafficRemaining").textContent = bytefmt(traffic.remaining);
    track.classList.remove("unlimited");
    track.setAttribute("aria-valuenow", ratio.toFixed(1));
    track.removeAttribute("aria-valuetext");
    progress.style.width = `${ratio}%`;
  }
  $("#expiryDate").textContent = formatExpiry(data.expiryDate);
  $("#daysRemaining").textContent = formatRemaining(data.daysRemaining);
  const updated = data.updatedAt ? new Date(data.updatedAt) : new Date();
  $("#updatedAt").textContent = `数据更新于 ${updated.toLocaleString("zh-CN", { hour12: false })}`;
  setLoadingState("ready");
}

async function session() {
  const response = await fetch("/auth/loginUser", { credentials: "same-origin", cache: "no-store" });
  if (!response.ok) return null;
  const body = await response.json();
  return body.data || null;
}

async function loadOverview({ initial = false } = {}) {
  if (loading) return;
  loading = true;
  const refreshButton = $("#refreshPortal");
  refreshButton.disabled = true;
  refreshButton.classList.add("loading");
  if (initial) setLoadingState("loading");
  try {
    const response = await fetch("/portal/me/summary", { credentials: "same-origin", cache: "no-store", headers: { Accept: "application/json" } });
    if (response.status === 401 || response.status === 403) {
      location.replace("/");
      return;
    }
    const body = await response.json();
    if (!response.ok || (body.Msg && body.Msg !== "success")) throw new Error(body.Msg || body.message || "无法获取账号信息");
    renderOverview(body.Data || body.data || {});
  } catch (error) {
    $("#portalErrorMessage").textContent = error.message || "请稍后重试";
    setLoadingState("error");
  } finally {
    loading = false;
    refreshButton.disabled = false;
    refreshButton.classList.remove("loading");
  }
}

async function logout() {
  const button = $("#portalLogout");
  button.disabled = true;
  try {
    await fetch("/auth/logout", { method: "POST", credentials: "same-origin" });
  } finally {
    location.replace("/");
  }
}

async function boot() {
  applyTheme();
  const identity = await session().catch(() => null);
  if (!identity) {
    location.replace("/");
    return;
  }
  if (identity.role !== "user") {
    location.replace("/");
    return;
  }
  $("#headerUsername").textContent = identity.username || "";
  $("#settingsUsername").textContent = identity.username || "—";
  await loadOverview({ initial: true });
  clearInterval(refreshTimer);
  refreshTimer = window.setInterval(() => {
    if (document.visibilityState === "visible") loadOverview();
  }, 60000);
}

themeMedia.addEventListener("change", () => { if (portalTheme === "system") applyTheme(); });
document.querySelectorAll('input[name="portalTheme"]').forEach((input) => input.addEventListener("change", () => {
  portalTheme = input.value;
  applyTheme();
}));
$("#openSettings").addEventListener("click", () => $("#settingsDialog").showModal());
$("#closeSettings").addEventListener("click", () => $("#settingsDialog").close());
$("#settingsDialog").addEventListener("click", (event) => { if (event.target === $("#settingsDialog")) $("#settingsDialog").close(); });
$("#portalLogout").addEventListener("click", logout);
$("#refreshPortal").addEventListener("click", () => loadOverview());
$("#retryPortal").addEventListener("click", () => loadOverview({ initial: true }));
document.addEventListener("visibilitychange", () => { if (document.visibilityState === "visible") loadOverview(); });

boot();
