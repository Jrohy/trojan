(function () {
  "use strict";

  const validRanges = new Set(["1h", "24h", "7d", "30d"]);
  const statusLabels = { healthy: "健康", degraded: "降级", critical: "严重", failed: "失败", unknown: "未知" };
  const state = { active: false, range: "24h", inflight: null, queued: false, api: null, controller: null };
  const el = (selector) => document.querySelector(selector);
  const html = (value) => String(value ?? "").replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[char]);
  const finite = (value) => typeof value === "number" && Number.isFinite(value);
  const clamp = (value, min, max) => Math.max(min, Math.min(max, value));

  function localTime(value, fallback = "未知") {
    if (!value) return fallback;
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? fallback : date.toLocaleString();
  }

  function percent(value, digits = 2) {
    return finite(value) ? `${(value * 100).toFixed(digits)}%` : "未知";
  }

  function latency(value) {
    return finite(value) ? `${Math.round(value).toLocaleString()} ms` : "未知";
  }

  function bytes(value) {
    if (!finite(value)) return "未知";
    const units = ["B", "KiB", "MiB", "GiB", "TiB"];
    let amount = Math.max(0, value), index = 0;
    while (amount >= 1024 && index < units.length - 1) { amount /= 1024; index += 1; }
    return `${amount.toFixed(index ? 1 : 0)} ${units[index]}`;
  }

  function rate(value) { return finite(value) ? `${bytes(value)}/s` : "未知"; }

  function renderStatus(status) {
    const key = statusLabels[status] ? status : "unknown";
    const node = el("#opsStatus");
    node.className = `opsOverallStatus ${key}`;
    node.querySelector("strong").textContent = `整体${statusLabels[key]}`;
  }

  function renderOverview(data) {
    const overview = data.overview || {};
    const expected = Number(overview.expected || 0);
    const effective = Number(overview.success || 0) + Number(overview.failed || 0);
    const hasSamples = expected > 0 || effective > 0 || Number(overview.unknown || 0) > 0;
    el("#opsAvailability").textContent = percent(overview.availability);
    el("#opsAvailabilitySub").textContent = finite(overview.availability) ? `成功 ${Number(overview.success || 0).toLocaleString()} / 有效 ${effective.toLocaleString()}` : "尚无有效样本";
    el("#opsCoverage").textContent = hasSamples ? percent(overview.coverage, 1) : "未知";
    el("#opsCoverageSub").textContent = hasSamples ? `有效 ${effective.toLocaleString()} / 预期 ${expected.toLocaleString()}` : "尚无调度样本";
    el("#opsP95").textContent = latency(overview.p95Ms);
    el("#opsLatencySub").textContent = `P50 ${latency(overview.p50Ms)} · P99 ${latency(overview.p99Ms)}`;
    el("#opsErrors").textContent = hasSamples ? `${Number(overview.failed || 0).toLocaleString()} / ${Number(overview.unknown || 0).toLocaleString()}` : "未知";
    el("#opsErrorsSub").textContent = hasSamples ? "失败 / 未知（含缺失与跳过）" : "等待采样";
    el("#opsBudget").textContent = percent(overview.errorBudgetRemaining, 1);
    el("#opsBudgetSub").textContent = `候选 SLO ${percent(overview.sloTarget, 1)}`;
    renderStatus(data.status);
  }

  function resourceRisk(resources) {
    if (!resources || resources.error) return { status: "unknown", label: "未知", detail: resources && resources.error ? resources.error : "等待系统采样" };
    const serviceState = String(resources.serviceState || "unknown").toLowerCase();
    if (serviceState === "unknown") return { status: "unknown", label: "未知", detail: "代理服务状态未知" };
    if (serviceState !== "active") return { status: "critical", label: "严重", detail: `代理服务 ${resources.serviceState}` };
    if (resources.pressureSkipped) return { status: "degraded", label: "采集退让", detail: "系统压力较高，本轮跳过部分采集" };
    const critical = resources.memoryPercent >= 92 || resources.diskPercent >= 92 || resources.swapPercent >= 90 || resources.cpuPercent >= 90;
    const degraded = resources.memoryPercent >= 80 || resources.diskPercent >= 80 || resources.swapPercent >= 70 || resources.cpuPercent >= 75 || resources.load1 >= 0.85;
    if (critical) return { status: "critical", label: "严重", detail: "至少一项资源接近上限" };
    if (degraded) return { status: "degraded", label: "需关注", detail: "至少一项资源压力偏高" };
    const known = [resources.cpuPercent, resources.memoryPercent, resources.diskPercent].some(finite);
    return known ? { status: "healthy", label: "正常", detail: "关键资源处于可接受范围" } : { status: "unknown", label: "未知", detail: "等待系统采样" };
  }

  function renderResources(resources) {
    const data = resources || {};
    const risk = resourceRisk(data);
    const stateNode = el("#opsResourceState");
    stateNode.textContent = risk.label;
    stateNode.dataset.status = risk.status;
    el("#opsResourceSub").textContent = risk.detail;
    el("#opsResourceSampledAt").textContent = `采样于 ${localTime(data.sampledAt)}`;
    const rows = [
      ["CPU", finite(data.cpuPercent) ? `${data.cpuPercent.toFixed(1)}%` : "未知", data.cpuPercent, 75, 90],
      ["内存", finite(data.memoryPercent) ? `${data.memoryPercent.toFixed(1)}%` : "未知", data.memoryPercent, 80, 92],
      ["Swap", finite(data.swapPercent) ? `${data.swapPercent.toFixed(1)}%` : "未知", data.swapPercent, 70, 90],
      ["磁盘", finite(data.diskPercent) ? `${data.diskPercent.toFixed(1)}%` : "未知", data.diskPercent, 80, 92],
    ];
    const meters = rows.map(([name, label, value, warning, critical]) => {
      const level = !finite(value) ? "unknown" : (value >= critical ? "critical" : (value >= warning ? "degraded" : "healthy"));
      return `<div class="opsResource"><div><span>${name}</span><b>${label}</b></div><div class="opsResourceMeter ${level}"><i style="width:${finite(value) ? clamp(value, 0, 100) : 0}%"></i></div></div>`;
    }).join("");
    const facts = `<div class="opsResourceFacts"><span>负载 <b>${finite(data.load1) ? data.load1.toFixed(2) : "未知"}</b></span><span>磁盘剩余 <b>${bytes(data.diskFreeBytes)}</b></span><span>实时上行 <b>${rate(data.networkUpBps)}</b></span><span>实时下行 <b>${rate(data.networkDownBps)}</b></span></div>`;
    const skip = data.pressureSkipped ? '<p class="opsResourceWarning">本轮采集因资源压力主动退让，相关数据可能不完整。</p>' : "";
    el("#opsResources").innerHTML = meters + facts + skip;
  }

  function segmentPaths(points, getter, xOf, yOf) {
    const segments = [];
    let current = [];
    points.forEach((point, index) => {
      const value = getter(point);
      if (!finite(value)) {
        if (current.length) segments.push(current);
        current = [];
        return;
      }
      current.push(`${xOf(index).toFixed(1)},${yOf(value).toFixed(1)}`);
    });
    if (current.length) segments.push(current);
    return segments;
  }

  function renderHistory(history, range) {
    const points = Array.isArray(history) ? history.slice().sort((a, b) => new Date(a.at).getTime() - new Date(b.at).getTime()).slice(-240) : [];
    el("#opsHistoryRange").textContent = `${range || state.range} · ${points.length} 个时间点`;
    if (!points.length) {
      el("#opsHistoryChart").innerHTML = '<div class="opsEmpty">暂无历史数据</div>';
      return;
    }
    const width = 900, height = 250, left = 46, right = 54, top = 18, bottom = 32;
    const plotWidth = width - left - right, plotHeight = height - top - bottom;
    const maxLatency = Math.max(1000, ...points.map((item) => finite(item.p95Ms) ? item.p95Ms : 0));
    const xOf = (index) => left + (points.length === 1 ? plotWidth / 2 : index * plotWidth / (points.length - 1));
    const availabilityY = (value) => top + (1 - clamp(value, 0, 1)) * plotHeight;
    const latencyY = (value) => top + (1 - clamp(value / maxLatency, 0, 1)) * plotHeight;
    const completeValue = (point, key) => Number(point.unknown || 0) > 0 ? null : point[key];
    const availability = segmentPaths(points, (point) => completeValue(point, "availability"), xOf, availabilityY);
    const p95 = segmentPaths(points, (point) => completeValue(point, "p95Ms"), xOf, latencyY);
    const lines = [0, .25, .5, .75, 1].map((ratio) => {
      const y = top + ratio * plotHeight;
      return `<line x1="${left}" y1="${y}" x2="${width - right}" y2="${y}"/><text x="5" y="${y + 4}">${Math.round((1 - ratio) * 100)}%</text>`;
    }).join("");
    const gaps = points.map((point, index) => {
      if (Number(point.unknown || 0) <= 0 && finite(point.availability) && point.status !== "unknown") return "";
      const step = plotWidth / Math.max(1, points.length - 1);
      return `<rect class="opsGap" x="${clamp(xOf(index) - step / 2, left, width - right)}" y="${top}" width="${Math.max(2, step)}" height="${plotHeight}"/>`;
    }).join("");
    const pathMarkup = (segments, className) => segments.map((segment) => segment.length === 1
      ? `<circle class="${className}" cx="${segment[0].split(",")[0]}" cy="${segment[0].split(",")[1]}" r="2.5"/>`
      : `<polyline class="${className}" points="${segment.join(" ")}"/>`).join("");
    const first = localTime(points[0].at, "-");
    const last = localTime(points[points.length - 1].at, "-");
    el("#opsHistoryChart").innerHTML = `<svg viewBox="0 0 ${width} ${height}" role="img" aria-label="合成探测可用性与 P95 路径耗时趋势"><g class="opsGridLines">${lines}</g>${gaps}${pathMarkup(availability, "opsAvailabilityLine")}${pathMarkup(p95, "opsLatencyLine")}<text class="opsAxisText" x="${left}" y="${height - 8}">${html(first)}</text><text class="opsAxisText end" x="${width - right}" y="${height - 8}">${html(last)}</text><text class="opsAxisText end" x="${width - 5}" y="${top + 4}">${Math.round(maxLatency)} ms</text></svg>`;
  }

  function bucketLabel(leMs, index) {
    if (leMs === null) return "> 3000 ms";
    if (index === 0) return `≤ ${leMs} ms`;
    const lower = [200, 500, 1000, 3000][index - 1];
    return `${lower}–${leMs} ms`;
  }

  function renderBuckets(buckets) {
    const rows = Array.isArray(buckets) ? buckets.slice(0, 5) : [];
    const total = rows.reduce((sum, item) => sum + Number(item.count || 0), 0);
    const max = Math.max(1, ...rows.map((item) => Number(item.count || 0)));
    el("#opsLatencyTotal").textContent = `${total.toLocaleString()} 个样本`;
    el("#opsLatencyBuckets").innerHTML = rows.length ? rows.map((item, index) => {
      const count = Number(item.count || 0);
      return `<div class="opsBucket"><span>${bucketLabel(item.leMs, index)}</span><div><i style="width:${count / max * 100}%"></i></div><b>${count.toLocaleString()}</b></div>`;
    }).join("") : '<div class="opsEmpty">暂无耗时样本</div>';
  }

  function renderTargets(targets) {
    const rows = Array.isArray(targets) ? targets : [];
    el("#opsTargetRows").innerHTML = rows.length ? rows.map((target) => {
      const status = statusLabels[target.status] ? target.status : "unknown";
      const detail = target.error || (finite(target.httpStatus) ? `目标响应状态 ${target.httpStatus}` : "-");
      return `<tr><td><strong>${html(target.name || target.id || "未命名目标")}</strong><small>${html(target.host || "-")}</small></td><td><span class="opsBadge ${status}">${statusLabels[status]}</span></td><td>${latency(target.latencyMs)}</td><td>${localTime(target.lastCheckedAt)}</td><td class="opsTargetDetail">${html(detail)}</td></tr>`;
    }).join("") : '<tr><td colspan="5" class="empty">暂无探测目标数据</td></tr>';
    const latest = rows.map((item) => item.lastCheckedAt).filter(Boolean).sort().pop();
    el("#opsTargetsUpdated").textContent = latest ? `最近检查 ${localTime(latest)}` : "尚未检查";
  }

  function renderEvents(events) {
    const rows = Array.isArray(events) ? events.slice().sort((a, b) => new Date(b.at).getTime() - new Date(a.at).getTime()).slice(0, 50) : [];
    el("#opsEvents").innerHTML = rows.length ? rows.map((event) => {
      const severity = ["critical", "warning", "info"].includes(event.severity) ? event.severity : "info";
      return `<article class="opsEvent ${severity}"><span></span><div><strong>${html(event.target || "系统")}</strong><p>${html(event.message || "状态发生变化")}</p><time>${localTime(event.at)}</time></div></article>`;
    }).join("") : '<div class="opsEmpty">当前范围内没有事件</div>';
  }

  function renderSources(data) {
    const sources = Array.isArray(data.sources) ? data.sources : [];
    el("#opsSources").innerHTML = sources.length ? sources.map((source) => `<article class="${source.available ? "available" : "unavailable"}"><span></span><div><strong>${html(source.label || source.id)}</strong><p>${html(source.scope || "来源范围未知")}${source.message ? ` · ${html(source.message)}` : ""}</p></div></article>`).join("") : '<div class="opsEmpty">暂无数据来源信息</div>';
    const freshness = data.freshness || {};
    const interval = Number(freshness.sampleIntervalSeconds || 60);
    const stale = freshness.stale === true;
    const message = freshness.lastSampleAt
      ? `最后采样 ${localTime(freshness.lastSampleAt)} · 采样周期 ${interval} 秒${freshness.nextExpectedAt ? ` · 下次预计 ${localTime(freshness.nextExpectedAt)}` : ""}`
      : "尚未产生有效采样；所有 SLA 指标保持未知。";
    const node = el("#opsFreshness");
    node.className = `opsFreshness ${stale ? "stale" : ""}`;
    node.textContent = stale ? `数据已过期 · ${message}` : message;
    el("#opsUpdatedAt").textContent = data.updatedAt ? `接口更新 ${localTime(data.updatedAt)}` : "更新时间未知";
  }

  function render(data) {
    renderOverview(data || {});
    renderResources(data && data.resources);
    renderHistory(data && data.history, data && data.range);
    renderBuckets(data && data.latencyBuckets);
    renderTargets(data && data.targets);
    renderEvents(data && data.events);
    renderSources(data || {});
    const notice = el("#opsNotice");
    notice.hidden = true;
    notice.textContent = "";
  }

  async function run(signal) {
    if (!state.active || document.visibilityState === "hidden" || !state.api) return;
    const requestedRange = state.range;
    const data = await state.api(`/common/ops?range=${encodeURIComponent(requestedRange)}`, { signal });
    if (state.active && requestedRange === state.range) render(data);
  }

  async function refresh(api, force = false) {
    if (api) state.api = api;
    if (!state.active || document.visibilityState === "hidden") return;
    if (state.inflight) {
      state.queued = true;
      return state.inflight;
    }
    state.controller = new AbortController();
    state.inflight = run(state.controller.signal);
    try {
      await state.inflight;
    } catch (error) {
      if (error && error.name !== "AbortError") throw error;
    } finally {
      state.inflight = null;
      state.controller = null;
      if (state.queued) {
        state.queued = false;
        await refresh(state.api, false);
      }
    }
  }

  function showError(message) {
    const notice = el("#opsNotice");
    notice.hidden = false;
    notice.textContent = `运维监控数据加载失败：${message || "未知错误"}`;
  }

  function setActive(active) {
    state.active = active;
    if (!active && state.controller) state.controller.abort();
  }

  document.querySelectorAll("[data-ops-range]").forEach((button) => button.addEventListener("click", () => {
    const range = button.dataset.opsRange;
    if (!validRanges.has(range) || range === state.range) return;
    state.range = range;
    document.querySelectorAll("[data-ops-range]").forEach((item) => item.classList.toggle("active", item === button));
    refresh(state.api, true).catch((error) => showError(error.message));
  }));
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "hidden") {
      if (state.controller) state.controller.abort();
      return;
    }
    if (state.active) refresh(state.api, false).catch((error) => showError(error.message));
  });

  window.OpsMonitor = { refresh, setActive, showError };
}());
