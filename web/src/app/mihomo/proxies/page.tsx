"use client";

import { type ReactNode, useCallback, useEffect, useMemo, useState } from "react";
import {
  ChevronDown,
  ChevronUp,
  Copy,
  Gauge,
  GitBranch,
  Loader2,
  Plus,
  RefreshCw,
  RotateCcw,
  RotateCw,
  Save,
  Search,
  Settings2,
  SlidersHorizontal,
  Trash2,
  X,
} from "lucide-react";
import { AppShell } from "@/components/AppShell";
import { useToaster, ToastStack } from "@/components/Toaster";
import { cn } from "@/lib/utils";
import { api, apiData, apiList, formatBytes } from "@/lib/api";

const ICON = (n: string) =>
  `https://raw.githubusercontent.com/Koolson/Qure/refs/heads/master/IconSet/Color/${n}.png`;

const FALLBACK_ICONS: Record<string, string> = {
  direct: ICON("Direct"),
  global: ICON("Global"),
  filter: ICON("Filter"),
  netflix: ICON("Netflix"),
  microsoft: ICON("Microsoft"),
  telegram: ICON("Telegram_X"),
  spotify: ICON("Spotify"),
  apple: ICON("Apple"),
  tiktok: ICON("TikTok"),
  game: ICON("Game"),
  ai: ICON("AI"),
};

const CHAIN_PROXY_EXAMPLE = `# 链式代理示例：让订阅里的节点通过“前置代理”拨出
proxy-providers:
  mysub:
    type: http
    url: https://example.com/sub.yaml
    path: ./proxy_providers/mysub.yaml
    override:
      dialer-proxy: 前置代理

proxy-groups:
  - name: 前置代理
    type: select
    proxies:
      - DIRECT
      - transit-node
`;

interface Node {
  name: string;
  delay: number;
  type: string;
  alive: boolean;
  icon: string;
  hidden: boolean;
}

interface Group {
  name: string;
  type: string;
  icon: string;
  hidden: boolean;
  order?: number;
  now: number;
  selected: string;
  selectedIcon: string;
  nodes: Node[];
  delay: number;
}

interface Provider {
  name: string;
  now: number;
  total: number;
  alive: number;
  nodes: Node[];
  used: string;
  quota: string;
  percent: number;
  expire?: string;
  extra?: string;
  updated: string;
  chainDialerProxy?: string;
  chainSource?: string;
}

interface ProviderSource {
  name: string;
  url: string;
}

interface ProxyGroupDraft {
  id: string;
  name: string;
  type: string;
  proxies: string;
  extra: string;
}

interface RuntimeStats {
  connections: number;
  uploadSpeed: number;
  downloadSpeed: number;
  uploadTotal: number;
  downloadTotal: number;
  mode: string;
}

interface ChainRecord {
  kind: "provider" | "proxy";
  name: string;
  dialerProxy: string;
  source: string;
  type?: string;
}

interface ChainRecords {
  providerChains: ChainRecord[];
  proxyChains: ChainRecord[];
}

interface ChainTarget {
  label: string;
  tone: "ok" | "warning";
}

interface ProxyPageSettings {
  hideUnavailable: boolean;
  showHiddenProxies: boolean;
  autoDisconnectOnSwitch: boolean;
  doubleColumn: boolean;
  delayTestUrl: string;
  delayTimeoutMs: number;
  delayLowMs: number;
  delayHighMs: number;
  sortBy: string;
  nodeNameDisplay: "truncate" | "wrap";
}

const EMPTY_STATS: RuntimeStats = {
  connections: 0,
  uploadSpeed: 0,
  downloadSpeed: 0,
  uploadTotal: 0,
  downloadTotal: 0,
  mode: "-",
};

const PROXIES_SETTINGS_KEY = "msf-mihomo-proxies.settings";
const PROXIES_TAB_KEY = "msf-mihomo-proxies.tab";
const COLLAPSE_PREFIX = "msf-mihomo-proxies.collapse";
const AUTO_REFRESH_INTERVAL_MS = 30000;

const DEFAULT_PROXY_SETTINGS: ProxyPageSettings = {
  hideUnavailable: false,
  showHiddenProxies: false,
  autoDisconnectOnSwitch: true,
  doubleColumn: true,
  delayTestUrl: "https://www.gstatic.com/generate_204",
  delayTimeoutMs: 5000,
  delayLowMs: 400,
  delayHighMs: 800,
  sortBy: "default",
  nodeNameDisplay: "truncate",
};

const FALLBACK_HIDDEN_GROUP_NAMES = new Set([
  "高级节点",
  "游戏节点",
  "香港节点",
  "新加坡节点",
  "韩国节点",
  "台湾节点",
  "日本节点",
  "美国节点",
  "省流节点",
]);

const REGION_HIDDEN_GROUP_RE =
  /^(香港|澳门|台湾|新加坡|日本|韩国|美国|英国|德国|法国|荷兰|加拿大|澳大利亚|俄罗斯|印度|土耳其|越南|泰国|菲律宾|马来西亚|印尼|巴西|阿根廷|墨西哥|西班牙|意大利|波兰|瑞士|瑞典|挪威|芬兰|丹麦)节点$/;

const DEFAULT_GROUP_ORDER = [
  "节点选择",
  "手动切换",
  "漏网之鱼",
  "网络测试",
  "人工智能",
  "游戏平台",
  "Netflix",
  "苹果服务",
  "微软服务",
  "TikTok",
  "Spotify",
  "PT站点",
  "Telegram",
  "高级节点",
  "游戏节点",
  "香港节点",
  "新加坡节点",
  "韩国节点",
  "台湾节点",
  "日本节点",
  "美国节点",
  "省流节点",
  "机场节点",
  "全球直连",
];

function stringValue(value: unknown) {
  return value == null ? "" : String(value);
}

function readProxySettings(): ProxyPageSettings {
  if (typeof window === "undefined") return DEFAULT_PROXY_SETTINGS;
  try {
    const raw = window.localStorage.getItem(PROXIES_SETTINGS_KEY);
    if (!raw) return DEFAULT_PROXY_SETTINGS;
    const parsed = JSON.parse(raw) as Partial<ProxyPageSettings>;
    return {
      ...DEFAULT_PROXY_SETTINGS,
      ...parsed,
      delayTimeoutMs: Number(parsed.delayTimeoutMs) || DEFAULT_PROXY_SETTINGS.delayTimeoutMs,
      delayLowMs: Number(parsed.delayLowMs) || DEFAULT_PROXY_SETTINGS.delayLowMs,
      delayHighMs: Number(parsed.delayHighMs) || DEFAULT_PROXY_SETTINGS.delayHighMs,
      nodeNameDisplay: parsed.nodeNameDisplay === "wrap" ? "wrap" : "truncate",
    };
  } catch {
    return DEFAULT_PROXY_SETTINGS;
  }
}

function normalizeProxyGroupDrafts(payload: any): ProxyGroupDraft[] {
  const data = apiData<any>(payload, payload || {});
  const rows = apiList<any>(data, ["proxy-groups", "proxy_groups", "groups"]);
  return rows.map((row, index) => {
    const extra: Record<string, unknown> = {};
    Object.entries(row || {}).forEach(([key, value]) => {
      if (key === "name" || key === "type" || key === "proxies") return;
      extra[key] = value;
    });
    const proxies = Array.isArray(row?.proxies) ? row.proxies.map(String).join(", ") : stringValue(row?.proxies);
    return {
      id: `${stringValue(row?.name) || "group"}-${index}`,
      name: stringValue(row?.name),
      type: stringValue(row?.type || "select"),
      proxies,
      extra: Object.keys(extra).length > 0 ? JSON.stringify(extra, null, 2) : "",
    };
  });
}

function serializeProxyGroupDrafts(rows: ProxyGroupDraft[]) {
  return rows
    .map((row) => {
      const name = row.name.trim();
      if (!name) return null;
      const item: Record<string, unknown> = {
        name,
        type: row.type.trim() || "select",
        proxies: row.proxies.split(",").map((part) => part.trim()).filter(Boolean),
      };
      if (row.extra.trim()) {
        const extra = JSON.parse(row.extra) as Record<string, unknown>;
        Object.assign(item, extra);
      }
      return item;
    })
    .filter(Boolean);
}

function readSavedTab(): "groups" | "providers" {
  if (typeof window === "undefined") return "groups";
  return window.localStorage.getItem(PROXIES_TAB_KEY) === "providers" ? "providers" : "groups";
}

function collapseKey(kind: "group" | "provider", name: string) {
  return `${COLLAPSE_PREFIX}.${kind}.${name}`;
}

function readCollapsed(kind: "group" | "provider", name: string) {
  if (typeof window === "undefined") return false;
  const value = window.localStorage.getItem(collapseKey(kind, name));
  if (value == null) return true;
  return value === "1" || value === "true";
}

function numberValue(value: unknown) {
  const numeric = Number(value);
  return Number.isFinite(numeric) ? numeric : 0;
}

function boolValue(value: unknown, fallback = true) {
  if (value == null) return fallback;
  if (typeof value === "boolean") return value;
  if (typeof value === "number") return value !== 0;
  const normalized = String(value).toLowerCase();
  return !["false", "0", "dead", "timeout"].includes(normalized);
}

function arrayValue(value: unknown): any[] {
  return Array.isArray(value) ? value : [];
}

function objectValue(value: unknown): Record<string, any> {
  return value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, any> : {};
}

function chainDialerProxyFromProvider(row: any) {
  const override = objectValue(row?.override || row?.raw?.override);
  return stringValue(override["dialer-proxy"] || override.dialer_proxy).trim();
}

function normalizeChainRecord(row: any, kind: ChainRecord["kind"]): ChainRecord | null {
  const name = stringValue(row?.name).trim();
  const dialerProxy = stringValue(row?.dialer_proxy || row?.["dialer-proxy"]).trim();
  if (!name || !dialerProxy) return null;
  return {
    kind,
    name,
    dialerProxy,
    source: stringValue(row?.source || (kind === "provider" ? `proxy-providers.${name}.override.dialer-proxy` : "proxies[].dialer-proxy")),
    type: stringValue(row?.type || row?.provider_type),
  };
}

function normalizeChainRecords(payload: any): ChainRecords {
  const data = apiData<any>(payload, payload || {});
  const chains = data?.chains || data || {};
  const providerChains = apiList<any>(chains, ["provider_chains", "providerChains", "providers"])
    .map((row) => normalizeChainRecord(row, "provider"))
    .filter(Boolean) as ChainRecord[];
  const proxyChains = apiList<any>(chains, ["proxy_chains", "proxyChains", "proxies"])
    .map((row) => normalizeChainRecord(row, "proxy"))
    .filter(Boolean) as ChainRecord[];
  return { providerChains, proxyChains };
}

function fallbackIcon(name: string, type = "") {
  const key = `${name} ${type}`.toLowerCase();
  if (key.includes("direct")) return FALLBACK_ICONS.direct;
  if (key.includes("global")) return FALLBACK_ICONS.global;
  if (key.includes("netflix")) return FALLBACK_ICONS.netflix;
  if (key.includes("microsoft") || key.includes("微软")) return FALLBACK_ICONS.microsoft;
  if (key.includes("telegram")) return FALLBACK_ICONS.telegram;
  if (key.includes("spotify")) return FALLBACK_ICONS.spotify;
  if (key.includes("apple") || key.includes("苹果")) return FALLBACK_ICONS.apple;
  if (key.includes("tiktok")) return FALLBACK_ICONS.tiktok;
  if (key.includes("game") || key.includes("游戏")) return FALLBACK_ICONS.game;
  if (key.includes("ai") || key.includes("人工智能")) return FALLBACK_ICONS.ai;
  return FALLBACK_ICONS.filter;
}

function latestDelay(row: any) {
  const direct = numberValue(row.delay);
  if (direct > 0) return direct;
  const history = arrayValue(row.history);
  for (let i = history.length - 1; i >= 0; i -= 1) {
    const delay = numberValue(history[i]?.delay);
    if (delay > 0) return delay;
  }
  return 0;
}

function normalizeNode(row: any, nameFallback = ""): Node {
  if (typeof row === "string") {
    return {
      name: row,
      delay: 0,
      type: "",
      alive: true,
      icon: fallbackIcon(row),
      hidden: false,
    };
  }
  const name = stringValue(row?.name || nameFallback);
  const type = stringValue(row?.type || row?.raw?.type);
  const delay = latestDelay(row);
  return {
    name,
    delay,
    type,
    alive: boolValue(row?.alive, delay > 0 || type.toLowerCase() === "direct"),
    icon: stringValue(row?.icon || row?.raw?.icon) || fallbackIcon(name, type),
    hidden: boolValue(row?.hidden ?? row?.raw?.hidden, false),
  };
}

function inferredHiddenGroup(name: string) {
  return FALLBACK_HIDDEN_GROUP_NAMES.has(name) || REGION_HIDDEN_GROUP_RE.test(name);
}

function fallbackGroupOrder(name: string) {
  const index = DEFAULT_GROUP_ORDER.indexOf(name);
  return index >= 0 ? index : 100000;
}

function configuredGroupOrder(row: any, name: string) {
  const raw = Number(row?.order ?? row?.config_order);
  return Number.isFinite(raw) ? raw : fallbackGroupOrder(name);
}

function defaultGroupSort(a: Group, b: Group) {
  const rank = (a.order ?? fallbackGroupOrder(a.name)) - (b.order ?? fallbackGroupOrder(b.name));
  if (rank !== 0) return rank;
  return a.name.localeCompare(b.name, "zh");
}

function normalizeGroups(data: any): { groups: Group[]; nodes: Node[] } {
  const proxyMap = data?.proxies && typeof data.proxies === "object" ? data.proxies : {};
  const rawNodes = apiList<any>(data, ["proxy_list", "nodes"]);
  const nodeMap = new Map<string, Node>();
  for (const row of Object.values(proxyMap)) {
    const node = normalizeNode(row);
    if (node.name) nodeMap.set(node.name, node);
  }
  for (const row of rawNodes) {
    const node = normalizeNode(row);
    if (node.name) nodeMap.set(node.name, node);
  }

  const rawGroups = apiList<any>(data, ["groups", "proxy_groups"]);
  const groups = rawGroups.map((row) => {
    const allNames = arrayValue(row.all).map(stringValue).filter(Boolean);
    const selected = stringValue(row.now || row.selected || allNames[0] || "");
    const nodes = allNames.length
      ? allNames.map((name) => nodeMap.get(name) || normalizeNode(proxyMap[name], name))
      : [normalizeNode(row)];
    const selectedNode = nodeMap.get(selected) || nodes.find((node) => node.name === selected);
    const name = stringValue(row.name);
    const type = stringValue(row.type || row.raw?.type || "Selector");
    return {
      name,
      type,
      icon: stringValue(row.icon || row.raw?.icon) || fallbackIcon(name, type),
      hidden: boolValue(row.hidden ?? row.raw?.hidden, false) || inferredHiddenGroup(name),
      order: configuredGroupOrder(row, name),
      now: selected ? Math.max(0, allNames.indexOf(selected) + 1) : 0,
      selected: selected || "-",
      selectedIcon: selectedNode?.icon || fallbackIcon(selected || name, selectedNode?.type || type),
      nodes,
      delay: latestDelay(row),
    };
  });

  return { groups, nodes: [...nodeMap.values()] };
}

function providerSubscriptionInfo(row: any) {
  const info = row?.subscriptionInfo || row?.subscription_info || row?.runtime?.subscriptionInfo || row?.runtime?.subscription_info || {};
  const upload = numberValue(info.Upload ?? info.upload ?? info.uploadTotal);
  const download = numberValue(info.Download ?? info.download ?? info.downloadTotal);
  const total = numberValue(info.Total ?? info.total);
  const expire = numberValue(info.Expire ?? info.expire);
  const used = upload + download;
  return { used, total, expire };
}

function formatExpire(value: number) {
  if (!value) return "";
  const ts = value > 1_000_000_000_000 ? value : value * 1000;
  return `到期: ${new Date(ts).toLocaleDateString()}`;
}

function formatUpdated(value: unknown) {
  const raw = stringValue(value);
  if (!raw) return "未更新";
  const time = Date.parse(raw);
  if (!Number.isFinite(time)) return raw;
  const days = Math.floor((Date.now() - time) / 86400000);
  if (days <= 0) return "今天更新";
  return `更新于 ${days} 天前`;
}

function providerVehicleType(row: any) {
  return stringValue(row.vehicleType || row.vehicle_type || row["vehicle-type"] || row.runtime?.vehicleType || row.runtime?.vehicle_type).toLowerCase();
}

function providerRows(data: any) {
  const runtimeRows = apiList<any>(data, ["runtime_items", "runtime_providers"]);
  if (runtimeRows.length > 0) return runtimeRows;
  const providers = data?.providers;
  if (Array.isArray(providers)) return providers;
  if (providers && typeof providers === "object") {
    return Object.entries(providers).map(([name, row]) => ({ ...(row as any), name: stringValue((row as any)?.name || name) }));
  }
  return apiList<any>(data, ["items", "data"]);
}

function normalizeProviders(data: any): Provider[] {
  return providerRows(data).filter((row) => providerVehicleType(row) !== "compatible").map((item) => {
    const proxies = arrayValue(item.proxies || item.runtime?.proxies);
    const nodes = proxies.map((proxy) => normalizeNode(proxy)).filter((proxy) => proxy.name);
    const sub = providerSubscriptionInfo(item);
    const chainDialerProxy = chainDialerProxyFromProvider(item);
    const totalNodes = nodes.length;
    const alive = nodes.filter((proxy) => proxy.alive).length;
    const totalQuota = sub.total;
    const percent = totalQuota > 0 ? (sub.used / totalQuota) * 100 : 0;
    return {
      name: stringValue(item.name),
      now: 0,
      total: totalNodes,
      alive: totalNodes ? alive : numberValue(item.alive || item.available),
      nodes,
      used: totalQuota ? formatBytes(sub.used) : "-",
      quota: totalQuota ? formatBytes(totalQuota) : "-",
      percent,
      expire: formatExpire(sub.expire),
      updated: formatUpdated(item.updatedAt || item.updated_at || item.runtime?.updatedAt || item.runtime?.updated_at),
      chainDialerProxy: chainDialerProxy || undefined,
      chainSource: chainDialerProxy ? `proxy-providers.${stringValue(item.name)}.override.dialer-proxy` : undefined,
    };
  }).sort((a, b) => a.name.localeCompare(b.name, "zh"));
}

function normalizeProviderSources(payload: any): ProviderSource[] {
  const data = apiData<any>(payload, payload);
  const items = apiList<any>(data, ["items", "data"]);
  return items
    .map((item) => ({ name: stringValue(item.name), url: stringValue(item.url) }))
    .filter((item) => item.name || item.url);
}

function normalizeStats(overviewPayload: any): RuntimeStats {
  const data = apiData<any>(overviewPayload, overviewPayload || {});
  const stats = data.stats || data;
  return {
    connections: numberValue(data.activeConnections ?? data.active_connections ?? stats.activeConnections ?? stats.active_connections),
    uploadSpeed: numberValue(data.uploadSpeed ?? data.upload_speed ?? stats.uploadSpeed ?? stats.upload_speed),
    downloadSpeed: numberValue(data.downloadSpeed ?? data.download_speed ?? stats.downloadSpeed ?? stats.download_speed),
    uploadTotal: numberValue(data.uploadTotal ?? data.upload_total ?? stats.uploadTotal ?? stats.upload_total),
    downloadTotal: numberValue(data.downloadTotal ?? data.download_total ?? stats.downloadTotal ?? stats.download_total),
    mode: stringValue(data.mode || data.config?.mode || stats.mode || "-"),
  };
}

function delayColor(d: number, settings = DEFAULT_PROXY_SETTINGS) {
  if (d === 0) return "text-muted-foreground";
  if (d < settings.delayLowMs) return "text-green-500";
  if (d < settings.delayHighMs) return "text-amber-500";
  return "text-red-500";
}

function dotColor(node: Node, settings = DEFAULT_PROXY_SETTINGS) {
  if (!node.alive || node.delay === 0) return "bg-muted-foreground/30";
  if (node.delay < settings.delayLowMs) return "bg-green-500";
  if (node.delay < settings.delayHighMs) return "bg-amber-500";
  return "bg-red-500";
}

function shouldShowGroup(group: Group, showHidden: boolean) {
  return showHidden || !group.hidden;
}

function groupDelay(g: Group) {
  const selected = g.nodes.find((n) => n.name === g.selected);
  if (selected && selected.delay > 0) return selected.delay;
  if (g.delay > 0) return g.delay;
  const ok = g.nodes.filter((n) => n.alive && n.delay > 0);
  return ok.length ? Math.min(...ok.map((n) => n.delay)) : 0;
}

const sorters: Record<string, (a: Group, b: Group) => number> = {
  default: defaultGroupSort,
  config: defaultGroupSort,
  "name-asc": (a, b) => a.name.localeCompare(b.name, "zh"),
  "name-desc": (a, b) => b.name.localeCompare(a.name, "zh"),
  "delay-asc": (a, b) => {
    const da = groupDelay(a) || Number.POSITIVE_INFINITY;
    const db = groupDelay(b) || Number.POSITIVE_INFINITY;
    return da - db;
  },
  "delay-desc": (a, b) => groupDelay(b) - groupDelay(a),
};

const modeLabels: Record<string, string> = {
  direct: "直连",
  rule: "规则",
  global: "全局",
};

const labelModes: Record<string, string> = {
  "直连": "direct",
  "规则": "rule",
  "全局": "global",
};

function SwitchControl({
  checked,
  onChange,
  disabled,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative inline-flex h-5 w-9 items-center rounded-full transition-colors disabled:opacity-50",
        checked ? "bg-primary" : "bg-muted"
      )}
      aria-pressed={checked}
    >
      <span
        className={cn(
          "inline-block h-4 w-4 rounded-full bg-background shadow transition-transform",
          checked ? "translate-x-4" : "translate-x-0.5"
        )}
      />
    </button>
  );
}

function SettingRow({
  label,
  description,
  children,
}: {
  label: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-xl border border-border/60 p-3">
      <div className="min-w-0">
        <div className="text-sm font-medium text-foreground">{label}</div>
        <div className="mt-1 text-xs text-muted-foreground">{description}</div>
      </div>
      {children}
    </div>
  );
}

function ProxySettingsModal({
  open,
  settings,
  onClose,
  onReset,
  onChange,
}: {
  open: boolean;
  settings: ProxyPageSettings;
  onClose: () => void;
  onReset: () => void;
  onChange: (settings: ProxyPageSettings) => void;
}) {
  if (!open) return null;

  const patch = <K extends keyof ProxyPageSettings>(key: K, value: ProxyPageSettings[K]) => {
    onChange({ ...settings, [key]: value });
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="w-full max-w-2xl max-h-[85vh] overflow-hidden rounded-xl border border-border bg-card shadow-apple-xl"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-center justify-between gap-3 border-b border-border bg-muted/30 px-5 py-3.5">
          <div>
            <h2 className="text-base font-semibold text-foreground">显示设置</h2>
            <p className="mt-0.5 text-xs text-muted-foreground">代理节点页面的显示、切换和延迟测试参数</p>
          </div>
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={onReset}
              className="inline-flex h-8 items-center gap-1.5 rounded-lg px-2.5 text-xs font-medium text-muted-foreground hover:bg-muted hover:text-foreground"
            >
              <RotateCcw className="h-3.5 w-3.5" />
              重置
            </button>
            <button
              type="button"
              onClick={onClose}
              className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground"
              title="关闭"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>

        <div className="max-h-[calc(85vh-112px)] space-y-5 overflow-y-auto px-5 py-4">
          <section className="space-y-2.5">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-foreground/80">显示选项</h3>
            <div className="space-y-1.5">
              <SettingRow label="隐藏不可用节点" description="开启后卡片中的超时或不可用节点不再显示在节点圆点中">
                <SwitchControl
                  checked={settings.hideUnavailable}
                  onChange={(checked) => patch("hideUnavailable", checked)}
                />
              </SettingRow>
              <SettingRow label="显示隐藏节点/点组" description="显示配置中标记为隐藏的节点和代理组">
                <SwitchControl
                  checked={settings.showHiddenProxies}
                  onChange={(checked) => patch("showHiddenProxies", checked)}
                />
              </SettingRow>
            </div>
          </section>

          <section className="space-y-2.5">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-foreground/80">布局选项</h3>
            <div className="space-y-1.5">
              <SettingRow label="双列布局" description="桌面端按左右两列显示策略组；关闭后单列显示">
                <SwitchControl
                  checked={settings.doubleColumn}
                  onChange={(checked) => patch("doubleColumn", checked)}
                />
              </SettingRow>
              <div className="rounded-xl border border-border/60 p-3">
                <div className="text-sm font-medium text-foreground">节点名称显示</div>
                <div className="mt-2 grid grid-cols-2 gap-2">
                  {[
                    ["truncate", "截断"] as const,
                    ["wrap", "换行"] as const,
                  ].map(([value, label]) => (
                    <button
                      key={value}
                      type="button"
                      onClick={() => patch("nodeNameDisplay", value)}
                      className={cn(
                        "rounded-lg border-2 px-3 py-2.5 text-xs font-medium transition-all",
                        settings.nodeNameDisplay === value
                          ? "border-primary bg-primary/10 text-primary"
                          : "border-border text-muted-foreground hover:border-primary/50 hover:bg-muted/50 hover:text-foreground"
                      )}
                    >
                      {label}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </section>

          <section className="space-y-2.5">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-foreground/80">行为选项</h3>
            <SettingRow label="切换后断开连接" description="切换策略组节点后调用连接清理接口，让新连接走新节点">
              <SwitchControl
                checked={settings.autoDisconnectOnSwitch}
                onChange={(checked) => patch("autoDisconnectOnSwitch", checked)}
              />
            </SettingRow>
          </section>

          <section className="space-y-2.5">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-foreground/80">延迟测试</h3>
            <div className="space-y-2.5">
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">测试 URL</label>
                <input
                  type="text"
                  value={settings.delayTestUrl}
                  onChange={(event) => patch("delayTestUrl", event.target.value)}
                  className="w-full rounded-lg border-2 border-border bg-background px-2.5 py-1.5 text-xs transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
                  placeholder={DEFAULT_PROXY_SETTINGS.delayTestUrl}
                />
              </div>
              <div className="grid grid-cols-3 gap-2.5">
                <div>
                  <label className="mb-1.5 block text-xs font-medium text-foreground">超时(ms)</label>
                  <input
                    type="number"
                    min={1000}
                    max={60000}
                    value={settings.delayTimeoutMs}
                    onChange={(event) => patch("delayTimeoutMs", Number(event.target.value) || 5000)}
                    className="w-full rounded-lg border-2 border-border bg-background px-2.5 py-1.5 text-xs transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-xs font-medium text-foreground">低延迟</label>
                  <input
                    type="number"
                    min={1}
                    max={5000}
                    value={settings.delayLowMs}
                    onChange={(event) => patch("delayLowMs", Number(event.target.value) || 400)}
                    className="w-full rounded-lg border-2 border-border bg-background px-2.5 py-1.5 text-xs transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-xs font-medium text-foreground">高延迟</label>
                  <input
                    type="number"
                    min={1}
                    max={5000}
                    value={settings.delayHighMs}
                    onChange={(event) => patch("delayHighMs", Number(event.target.value) || 800)}
                    className="w-full rounded-lg border-2 border-border bg-background px-2.5 py-1.5 text-xs transition-colors focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20"
                  />
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}

function chainTargetBadgeClass(target: ChainTarget) {
  return target.tone === "ok"
    ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
    : "bg-amber-500/10 text-amber-600 dark:text-amber-400";
}

function ChainRows({
  rows,
  emptyText,
  targetFor,
}: {
  rows: ChainRecord[];
  emptyText: string;
  targetFor: (name: string) => ChainTarget;
}) {
  if (rows.length === 0) {
    return <div className="rounded-lg border border-dashed border-border/70 p-4 text-center text-sm text-muted-foreground">{emptyText}</div>;
  }
  return (
    <div className="space-y-2">
      {rows.map((row) => {
        const target = targetFor(row.dialerProxy);
        return (
          <div key={`${row.kind}-${row.name}-${row.source}`} className="rounded-lg border border-border/60 bg-background p-3">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-medium text-sm text-foreground">{row.name}</span>
              {row.type && <span className="rounded-full bg-muted px-2 py-0.5 text-[11px] text-muted-foreground">{row.type}</span>}
              <span className={cn("rounded-full px-2 py-0.5 text-[11px] font-medium", chainTargetBadgeClass(target))}>{target.label}</span>
            </div>
            <div className="mt-2 grid gap-1 text-xs text-muted-foreground md:grid-cols-[8rem_1fr]">
              <span>前置代理</span>
              <span className="min-w-0 break-all font-mono text-foreground">{row.dialerProxy}</span>
              <span>配置来源</span>
              <span className="min-w-0 break-all font-mono">{row.source}</span>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function ChainProxyModal({
  providerChains,
  proxyChains,
  targetFor,
  onClose,
  onOpenConfig,
  onCopyExample,
}: {
  providerChains: ChainRecord[];
  proxyChains: ChainRecord[];
  targetFor: (name: string) => ChainTarget;
  onClose: () => void;
  onOpenConfig: () => void;
  onCopyExample: () => void;
}) {
  const empty = providerChains.length === 0 && proxyChains.length === 0;
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="w-full max-w-3xl max-h-[86vh] overflow-hidden rounded-xl border border-border bg-card shadow-apple-xl"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3 border-b border-border bg-muted/30 px-5 py-4">
          <div>
            <h2 className="text-lg font-semibold text-foreground">链式代理</h2>
            <p className="mt-1 text-xs text-muted-foreground">只读检测结果，配置请在专业配置中编辑</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground hover:bg-muted hover:text-foreground"
            title="关闭"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="max-h-[calc(86vh-132px)] space-y-5 overflow-y-auto px-5 py-4">
          {empty && (
            <div className="rounded-lg border border-dashed border-border/70 p-6 text-center text-sm text-muted-foreground">
              未检测到链式代理配置
            </div>
          )}
          <section className="space-y-2.5">
            <h3 className="text-sm font-semibold text-foreground">订阅批量链式代理</h3>
            <ChainRows rows={providerChains} emptyText="没有检测到 proxy-providers.override.dialer-proxy" targetFor={targetFor} />
          </section>
          <section className="space-y-2.5">
            <h3 className="text-sm font-semibold text-foreground">单节点链式代理</h3>
            <ChainRows rows={proxyChains} emptyText="没有检测到顶层 proxies[].dialer-proxy" targetFor={targetFor} />
          </section>
        </div>

        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border bg-muted/20 px-5 py-3">
          <button
            type="button"
            onClick={onCopyExample}
            className="inline-flex items-center gap-1.5 rounded-lg border border-input bg-background px-3 py-2 text-sm font-medium hover:bg-muted"
          >
            <Copy className="h-4 w-4" />
            复制链式代理示例
          </button>
          <button
            type="button"
            onClick={onOpenConfig}
            className="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            <Settings2 className="h-4 w-4" />
            打开专业配置
          </button>
        </div>
      </div>
    </div>
  );
}

function nodeDelayBadgeClass(node: Node, settings: ProxyPageSettings, active: boolean) {
  if (active) return "bg-primary-foreground/20 text-primary-foreground";
  if (!node.alive || node.delay <= 0) return "bg-muted text-muted-foreground";
  if (node.delay < settings.delayLowMs) return "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400";
  if (node.delay < settings.delayHighMs) return "bg-amber-500/15 text-amber-600 dark:text-amber-400";
  return "bg-red-500/15 text-red-600 dark:text-red-400";
}

function ProxyNodeTile({
  node,
  active,
  loading,
  settings,
  onClick,
  onTest,
}: {
  node: Node;
  active?: boolean;
  loading?: boolean;
  settings: ProxyPageSettings;
  onClick?: () => void;
  onTest?: () => void;
}) {
  const primaryAction = onClick || onTest;
  const badge = (
    <span className={cn("inline-flex h-5 min-w-8 shrink-0 items-center justify-center rounded-full px-1.5 text-[10px] font-semibold tabular-nums", nodeDelayBadgeClass(node, settings, !!active))}>
      {loading ? <Loader2 className="h-3 w-3 animate-spin" /> : node.delay > 0 ? node.delay : <Gauge className="h-3 w-3" />}
    </span>
  );

  return (
    <div
      role={primaryAction ? "button" : undefined}
      tabIndex={primaryAction ? 0 : undefined}
      onClick={primaryAction}
      onKeyDown={(event) => {
        if (!primaryAction) return;
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          primaryAction();
        }
      }}
      className={cn(
        "min-h-[4.25rem] w-[7.75rem] max-w-full min-w-0 rounded-lg border px-2.5 py-2 text-left transition-colors",
        "bg-muted/40 hover:bg-muted border-transparent",
        primaryAction && "cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary/30",
        active && "bg-primary text-primary-foreground hover:bg-primary/90"
      )}
      title={`${node.name}${node.delay > 0 ? ` · ${node.delay}ms` : ""}`}
    >
      <div className="flex w-full min-w-0 items-center gap-1.5 text-left">
        <img
          alt=""
          className="h-3.5 w-3.5 shrink-0 rounded-[3px] object-contain"
          referrerPolicy="no-referrer"
          src={node.icon || fallbackIcon(node.name, node.type)}
        />
        <span className={cn("min-w-0 text-xs font-medium", settings.nodeNameDisplay === "wrap" ? "break-all" : "truncate")}>
          {node.name}
        </span>
      </div>
      <div className="mt-1 flex items-center justify-between gap-2">
        <span
          className={cn("min-w-0 text-left text-[11px]", active ? "text-primary-foreground/75" : "text-muted-foreground", settings.nodeNameDisplay === "wrap" ? "break-all" : "truncate")}
        >
          {node.type || "-"}
        </span>
        {onTest ? (
          <button
            type="button"
            onClick={(event) => {
              event.stopPropagation();
              onTest();
            }}
            disabled={loading}
            className="shrink-0 rounded-full transition-transform hover:scale-110 disabled:opacity-60"
            aria-label={`测试 ${node.name} 延迟`}
            title={`测试 ${node.name} 延迟`}
          >
            {badge}
          </button>
        ) : (
          badge
        )}
      </div>
    </div>
  );
}

export default function MihomoProxiesPage() {
  const { toasts, showToast } = useToaster();
  const [groups, setGroups] = useState<Group[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [tab, setTab] = useState<"groups" | "providers">(readSavedTab);
  const [mode, setMode] = useState("规则");
  const [settings, setSettings] = useState<ProxyPageSettings>(readProxySettings);
  const [search, setSearch] = useState("");
  const [typeFilter, setTypeFilter] = useState("all");
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [testing, setTesting] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [showProviderModal, setShowProviderModal] = useState(false);
  const [showGroupModal, setShowGroupModal] = useState(false);
  const [showChainModal, setShowChainModal] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [collapseVersion, setCollapseVersion] = useState(0);
  const [sources, setSources] = useState<ProviderSource[]>([]);
  const [groupDrafts, setGroupDrafts] = useState<ProxyGroupDraft[]>([]);
  const [loadedSourceNames, setLoadedSourceNames] = useState<Set<string>>(new Set());
  const [runtimeStats, setRuntimeStats] = useState<RuntimeStats>(EMPTY_STATS);
  const [chainRecords, setChainRecords] = useState<ChainRecords>({ providerChains: [], proxyChains: [] });

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [proxiesPayload, overviewPayload, providerPayload] = await Promise.all([
        api<any>("/api/v1/mihomo/proxies"),
        api<any>("/api/v1/mihomo/overview").catch(() => null),
        api<any>("/api/v1/mihomo/proxy-providers").catch(() => null),
      ]);
      const data = apiData<any>(proxiesPayload, proxiesPayload);
      const providerData = providerPayload ? apiData<any>(providerPayload, providerPayload) : data;
      const normalized = normalizeGroups(data);
      setGroups(normalized.groups);
      setNodes(normalized.nodes);
      setProviders(normalizeProviders(providerData));
      setChainRecords(normalizeChainRecords(providerData));
      if (overviewPayload) {
        const stats = normalizeStats(overviewPayload);
        setRuntimeStats(stats);
        setMode(modeLabels[stats.mode] || stats.mode || "规则");
      }
    } catch (err) {
      showToast(err instanceof Error ? err.message : "加载代理失败");
    } finally {
      setLoading(false);
    }
  }, [showToast]);

  const loadProviderSources = useCallback(async () => {
    try {
      const payload = await api<any>("/api/v1/mihomo/proxy-providers");
      const nextSources = normalizeProviderSources(payload);
      setSources(nextSources);
      setLoadedSourceNames(new Set(nextSources.map((source) => source.name).filter(Boolean)));
    } catch (err) {
      showToast(err instanceof Error ? err.message : "加载供应商配置失败");
    }
  }, [showToast]);

  const loadGroupDrafts = useCallback(async () => {
    try {
      const payload = await api<any>("/api/v1/mihomo/proxy-groups-config");
      setGroupDrafts(normalizeProxyGroupDrafts(payload));
    } catch (err) {
      showToast(err instanceof Error ? err.message : "加载代理分组配置失败");
    }
  }, [showToast]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!autoRefresh) return;
    const timer = window.setInterval(() => void load(), AUTO_REFRESH_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [autoRefresh, load]);

  useEffect(() => {
    if (showProviderModal) void loadProviderSources();
  }, [showProviderModal, loadProviderSources]);

  useEffect(() => {
    if (showGroupModal) void loadGroupDrafts();
  }, [showGroupModal, loadGroupDrafts]);

  useEffect(() => {
    window.localStorage.setItem(PROXIES_SETTINGS_KEY, JSON.stringify(settings));
  }, [settings]);

  useEffect(() => {
    window.localStorage.setItem(PROXIES_TAB_KEY, tab);
  }, [tab]);

  const searchTerms = useMemo(() => search.trim().toLowerCase().split(/\s+/).filter(Boolean), [search]);

  const typeOptions = useMemo(() => {
    const types = new Set<string>();
    for (const group of groups) {
      if (!shouldShowGroup(group, settings.showHiddenProxies)) continue;
      if (group.type) types.add(group.type);
    }
    return ["all", ...Array.from(types).sort((a, b) => a.localeCompare(b, "zh"))];
  }, [groups, settings.showHiddenProxies]);

  const visibleGroups = useMemo(() => {
    const filtered = groups.filter((group) => {
      if (!shouldShowGroup(group, settings.showHiddenProxies)) return false;
      if (typeFilter !== "all" && group.type !== typeFilter) return false;
      if (searchTerms.length === 0) return true;
      const haystack = [group.name, group.type, group.selected, ...group.nodes.map((node) => node.name)].join(" ").toLowerCase();
      return searchTerms.every((term) => haystack.includes(term));
    });
    return [...filtered].sort(sorters[settings.sortBy] || sorters.default);
  }, [groups, searchTerms, settings.showHiddenProxies, settings.sortBy, typeFilter]);

  const visibleProviders = useMemo(() => {
    const filtered = providers.filter((provider) => {
      if (searchTerms.length === 0) return true;
      const haystack = [provider.name, provider.expire || "", provider.extra || "", provider.updated].join(" ").toLowerCase();
      return searchTerms.every((term) => haystack.includes(term));
    });
    if (settings.sortBy === "name-desc") return [...filtered].sort((a, b) => b.name.localeCompare(a.name, "zh"));
    return [...filtered].sort((a, b) => a.name.localeCompare(b.name, "zh"));
  }, [providers, searchTerms, settings.sortBy]);

  const groupByName = useMemo(() => new Map(groups.map((group) => [group.name, group])), [groups]);
  const nodeNameSet = useMemo(() => new Set(nodes.map((node) => node.name).filter(Boolean)), [nodes]);
  const providerChainByName = useMemo(
    () => new Map(chainRecords.providerChains.map((record) => [record.name, record])),
    [chainRecords.providerChains]
  );
  const chainTargetFor = useCallback((name: string): ChainTarget => {
    const target = name.trim();
    if (target === "DIRECT" || target === "REJECT") return { label: "内置目标", tone: "ok" };
    if (groupByName.has(target)) return { label: "策略组", tone: "ok" };
    if (nodeNameSet.has(target)) return { label: "节点", tone: "ok" };
    return { label: "未找到", tone: "warning" };
  }, [groupByName, nodeNameSet]);

  const currentCollapseNames = tab === "providers" ? visibleProviders.map((provider) => provider.name) : visibleGroups.map((group) => group.name);

  const allCurrentCollapsed = useMemo(() => {
    const kind = tab === "providers" ? "provider" : "group";
    return currentCollapseNames.length > 0 && currentCollapseNames.every((name) => readCollapsed(kind, name));
  }, [collapseVersion, currentCollapseNames, tab]);

  const setItemCollapsed = (kind: "group" | "provider", name: string, collapsed: boolean) => {
    window.localStorage.setItem(collapseKey(kind, name), collapsed ? "1" : "0");
    setCollapseVersion((version) => version + 1);
  };

  const setAllCurrentCollapsed = (collapsed: boolean) => {
    const kind = tab === "providers" ? "provider" : "group";
    for (const name of currentCollapseNames) {
      window.localStorage.setItem(collapseKey(kind, name), collapsed ? "1" : "0");
    }
    setCollapseVersion((version) => version + 1);
  };

  const stats = useMemo(() => {
    const liveNodes = nodes.filter((node) => node.alive).length;
    return [
      ["代理组", String(visibleGroups.length)],
      ["连接", String(runtimeStats.connections)],
      ["↑", `${formatBytes(runtimeStats.uploadSpeed)}/s`],
      ["↓", `${formatBytes(runtimeStats.downloadSpeed)}/s`],
      ["上传", formatBytes(runtimeStats.uploadTotal)],
      ["下载", formatBytes(runtimeStats.downloadTotal)],
      ["模式", mode],
      ["供应商", String(providers.length || liveNodes)],
    ];
  }, [mode, nodes, providers.length, runtimeStats, visibleGroups.length]);

  const fetchDelay = useCallback(async (name: string) => {
    if (!name || name === "REJECT") return 0;
    const timeout = Math.max(1000, settings.delayTimeoutMs || DEFAULT_PROXY_SETTINGS.delayTimeoutMs);
    const testUrl = settings.delayTestUrl.trim() || DEFAULT_PROXY_SETTINGS.delayTestUrl;
    const payload = await api<any>(
      `/api/v1/mihomo/proxies/${encodeURIComponent(name)}/delay?timeout=${encodeURIComponent(String(timeout))}&url=${encodeURIComponent(testUrl)}`
    );
    return numberValue(apiData<any>(payload, payload).delay);
  }, [settings.delayTestUrl, settings.delayTimeoutMs]);

  const collectDelayTargets = useCallback((groupName: string) => {
    const targets = new Set<string>();
    const visitedGroups = new Set<string>();

    const add = (name: string) => {
      if (name && name !== "REJECT") targets.add(name);
    };

    const visit = (name: string, depth: number) => {
      if (depth > 6) {
        add(name);
        return;
      }
      const group = groupByName.get(name);
      if (!group) {
        add(name);
        return;
      }
      if (visitedGroups.has(name)) return;
      visitedGroups.add(name);
      add(group.name);
      for (const node of group.nodes) {
        add(node.name);
        if (groupByName.has(node.name) && node.name !== group.name) {
          visit(node.name, depth + 1);
        }
      }
    };

    visit(groupName, 0);
    return Array.from(targets).slice(0, 260);
  }, [groupByName]);

  const runDelayBatch = useCallback(async (names: string[]) => {
    const results = new Map<string, number>();
    let index = 0;
    const workers = Array.from({ length: Math.min(8, Math.max(1, names.length)) }, async () => {
      while (index < names.length) {
        const name = names[index];
        index += 1;
        try {
          results.set(name, await fetchDelay(name));
        } catch {
          results.set(name, 0);
        }
      }
    });
    await Promise.all(workers);
    return results;
  }, [fetchDelay]);

  const aggregateGroupDelays = useCallback((startName: string, measured: Map<string, number>) => {
    const next = new Map(measured);
    const visiting = new Set<string>();

    const resolve = (name: string): number => {
      const direct = next.get(name) || 0;
      const group = groupByName.get(name);
      if (!group || visiting.has(name)) return direct;
      visiting.add(name);
      const childDelays = group.nodes
        .map((node) => {
          if (groupByName.has(node.name)) return resolve(node.name);
          return next.get(node.name) || node.delay || 0;
        })
        .filter((delay) => delay > 0);
      visiting.delete(name);
      const aggregate = direct > 0 ? direct : childDelays.length ? Math.min(...childDelays) : 0;
      if (aggregate > 0) next.set(name, aggregate);
      return aggregate;
    };

    resolve(startName);
    return next;
  }, [groupByName]);

  const applyDelayResults = useCallback((delays: Map<string, number>) => {
    const patchNode = (node: Node) => {
      if (!delays.has(node.name)) return node;
      const delay = delays.get(node.name) || 0;
      return { ...node, delay, alive: delay > 0 || node.type.toLowerCase() === "direct" };
    };

    setGroups((items) =>
      items.map((group) => ({
        ...group,
        delay: delays.has(group.name) ? delays.get(group.name) || 0 : group.delay,
        nodes: group.nodes.map(patchNode),
      }))
    );
    setNodes((items) => items.map(patchNode));
    setProviders((items) =>
      items.map((provider) => ({
        ...provider,
        nodes: provider.nodes.map(patchNode),
      }))
    );
  }, []);

  const testGroup = async (name: string) => {
    setTesting(name);
    try {
      const targets = collectDelayTargets(name);
      const measured = await runDelayBatch(targets);
      const delays = aggregateGroupDelays(name, measured);
      const delay = delays.get(name) || 0;
      applyDelayResults(delays);
      const successCount = Array.from(measured.values()).filter((value) => value > 0).length;
      showToast(delay > 0 ? `${name} 延迟 ${delay}ms，已测试 ${successCount}/${targets.length} 项` : `${name} 测试超时`);
    } catch (err) {
      showToast(err instanceof Error ? err.message : "延迟测试失败");
    } finally {
      setTesting(null);
    }
  };

  const testNode = async (nodeName: string) => {
    if (!nodeName || nodeName === "REJECT") return;
    if (groupByName.has(nodeName)) {
      await testGroup(nodeName);
      return;
    }
    setTesting(nodeName);
    try {
      const delay = await fetchDelay(nodeName);
      const patchNode = (node: Node) => (node.name === nodeName ? { ...node, delay, alive: delay > 0 } : node);
      setGroups((items) =>
        items.map((group) => ({
          ...group,
          nodes: group.nodes.map(patchNode),
          delay: group.name === nodeName ? delay : group.delay,
        }))
      );
      setProviders((items) =>
        items.map((provider) => ({
          ...provider,
          nodes: provider.nodes.map(patchNode),
        }))
      );
      showToast(delay > 0 ? `${nodeName} 延迟 ${delay}ms` : `${nodeName} 测试超时`);
    } catch (err) {
      showToast(err instanceof Error ? err.message : "延迟测试失败");
    } finally {
      setTesting(null);
    }
  };

  const selectNode = async (group: string, node: Node) => {
    setGroups((items) => items.map((item) => item.name === group ? { ...item, selected: node.name, selectedIcon: node.icon } : item));
    try {
      await api(`/api/v1/mihomo/proxies/${encodeURIComponent(group)}`, {
        method: "PUT",
        body: JSON.stringify({ name: node.name }),
      });
      if (settings.autoDisconnectOnSwitch) {
        await api("/api/v1/mihomo/connections", { method: "DELETE" }).catch(() => null);
      }
      showToast(`${group} → ${node.name}`);
      await load();
    } catch (err) {
      showToast(err instanceof Error ? err.message : "切换节点失败");
      await load();
    }
  };

  const switchMode = async (label: string) => {
    const next = labelModes[label] || label.toLowerCase();
    const previous = mode;
    setMode(label);
    try {
      await api("/api/v1/mihomo/controller/configs", {
        method: "PATCH",
        body: JSON.stringify({ mode: next }),
      });
      showToast(`已切换至${label}模式`);
      await load();
    } catch (err) {
      setMode(previous);
      showToast(err instanceof Error ? err.message : "切换模式失败");
    }
  };

  const openProfessionalConfig = () => {
    window.location.assign("/mihomo/config");
  };

  const copyChainExample = async () => {
    try {
      await navigator.clipboard.writeText(CHAIN_PROXY_EXAMPLE);
      showToast("链式代理示例已复制");
    } catch (err) {
      showToast(err instanceof Error ? err.message : "复制失败");
    }
  };

  const updateProvider = async (name: string) => {
    try {
      await api(`/api/v1/mihomo/proxy-providers/${encodeURIComponent(name)}/update`, { method: "POST" });
      showToast(`正在更新 ${name}...`);
      await load();
    } catch (err) {
      showToast(err instanceof Error ? err.message : "更新供应商失败");
    }
  };

  const saveSources = async (restart: boolean) => {
    const cleaned = sources.map((source) => ({ name: source.name.trim(), url: source.url.trim() })).filter((source) => source.name || source.url);
    if (cleaned.some((source) => !source.name || !source.url)) {
      showToast("供应商名称和 URL 不能为空");
      return;
    }
    const nextNames = new Set(cleaned.map((source) => source.name));
    const deleted = [...loadedSourceNames].filter((name) => !nextNames.has(name));
    try {
      await Promise.all([
        ...cleaned.map((source) =>
          api(`/api/v1/mihomo/proxy-providers/${encodeURIComponent(source.name)}`, {
            method: "PUT",
            body: JSON.stringify({ name: source.name, url: source.url, type: "http" }),
          })
        ),
        ...deleted.map((name) => api(`/api/v1/mihomo/proxy-providers/${encodeURIComponent(name)}`, { method: "DELETE" })),
      ]);
      if (restart) {
        await api("/api/v1/mihomo/restart", { method: "POST" }).catch(() => null);
      }
      setShowProviderModal(false);
      showToast(restart ? "已保存并重启" : "已保存");
      await load();
    } catch (err) {
      showToast(err instanceof Error ? err.message : "保存供应商失败");
    }
  };

  const saveGroupDrafts = async () => {
    try {
      const payload = serializeProxyGroupDrafts(groupDrafts);
      await api("/api/v1/mihomo/proxy-groups-config", {
        method: "PUT",
        body: JSON.stringify({ "proxy-groups": payload }),
      });
      setShowGroupModal(false);
      showToast("代理分组已保存并重启 Mihomo");
      await load();
    } catch (err) {
      showToast(err instanceof Error ? err.message : "保存代理分组失败，请检查高级 JSON");
    }
  };

  return (
    <AppShell>
      <div className="space-y-4 animate-fade-in">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <h1 className="text-2xl md:text-3xl font-bold text-foreground">代理</h1>
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-muted-foreground">
            {stats.map(([k, v], i) => (
              <span key={k} className="flex items-center gap-2">
                {i > 0 && <span className="text-border">•</span>}
                <span>
                  {k} <span className="font-semibold text-foreground">{v}</span>
                </span>
              </span>
            ))}
          </div>
          <div className="ml-auto flex items-center gap-2">
            <button
              type="button"
              onClick={() => {
                setAutoRefresh((value) => !value);
                showToast(autoRefresh ? "已关闭自动刷新" : "已开启自动刷新");
              }}
              className={cn(
                "inline-flex h-9 items-center rounded-lg px-3 text-sm font-medium transition-colors",
                autoRefresh ? "bg-muted text-foreground hover:bg-muted/80" : "border border-border bg-background text-muted-foreground hover:bg-muted hover:text-foreground"
              )}
            >
              自动刷新：{autoRefresh ? "开" : "关"}
            </button>
            <button
              type="button"
              onClick={() => setAllCurrentCollapsed(!allCurrentCollapsed)}
              disabled={currentCollapseNames.length === 0}
              className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-50"
              title={allCurrentCollapsed ? "展开列表" : "折叠列表"}
            >
              {allCurrentCollapsed ? <ChevronDown className="h-4 w-4" /> : <ChevronUp className="h-4 w-4" />}
            </button>
            <button
              type="button"
              onClick={() => void load()}
              disabled={loading}
              className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-50"
              title="刷新"
            >
              <RefreshCw className={cn("h-4 w-4", loading && "animate-spin")} />
            </button>
            <button
              type="button"
              onClick={() => setShowSettings(true)}
              className={cn(
                "inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-background text-muted-foreground transition-colors hover:bg-muted hover:text-foreground",
                showSettings && "bg-muted text-foreground"
              )}
              title="显示设置"
            >
              <SlidersHorizontal className="h-4 w-4" />
            </button>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <div className="flex gap-1 rounded-lg bg-muted/50 p-1">
            <button
              onClick={() => setTab("groups")}
              className={cn(
                "px-3 py-1.5 rounded-md text-sm font-medium transition-all",
                tab === "groups" ? "bg-card text-primary shadow-sm" : "text-muted-foreground hover:text-foreground"
              )}
            >
              策略组 ({visibleGroups.length})
            </button>
            <button
              onClick={() => setTab("providers")}
              className={cn(
                "px-3 py-1.5 rounded-md text-sm font-medium transition-all",
                tab === "providers" ? "bg-card text-primary shadow-sm" : "text-muted-foreground hover:text-foreground"
              )}
            >
              供应商 ({visibleProviders.length})
            </button>
          </div>

          <div className="flex gap-1 rounded-lg border border-border/60 p-0.5">
            {["直连", "规则", "全局"].map((m) => (
              <button
                key={m}
                onClick={() => void switchMode(m)}
                className={cn(
                  "px-3 py-1 rounded-md text-sm font-medium transition-all",
                  mode === m ? "bg-primary/10 text-primary" : "text-muted-foreground hover:text-foreground"
                )}
              >
                {m}
              </button>
            ))}
          </div>

          <div className="relative min-w-[260px] flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <input
              type="text"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="搜索｜多个关键词用空格分隔"
              className="h-9 w-full rounded-lg border border-border/60 bg-card pl-9 pr-9 text-sm text-foreground focus:border-primary/60 focus:outline-none"
            />
            {search.trim() !== "" && (
              <button
                type="button"
                onClick={() => setSearch("")}
                className="absolute right-2 top-1/2 inline-flex h-6 w-6 -translate-y-1/2 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
                title="清空搜索"
              >
                <X className="h-4 w-4" />
              </button>
            )}
          </div>

          <select
            value={settings.sortBy}
            onChange={(e) => setSettings((value) => ({ ...value, sortBy: e.target.value }))}
            className="h-9 px-3 rounded-lg border border-border/60 bg-card text-sm text-foreground focus:outline-none focus:border-primary/60"
            title="排序"
          >
            <option value="default">默认排序</option>
            <option value="config">按配置排序</option>
            <option value="name-asc">按名称升序</option>
            <option value="name-desc">按名称降序</option>
            <option value="delay-asc">按延迟升序</option>
            <option value="delay-desc">按延迟降序</option>
          </select>
          {tab === "groups" && (
            <select
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value)}
              className="h-9 px-3 rounded-lg border border-border/60 bg-card text-sm text-foreground focus:outline-none focus:border-primary/60"
              title="类型过滤"
            >
              {typeOptions.map((option) => (
                <option key={option} value={option}>
                  {option === "all" ? "全部" : option}
                </option>
              ))}
            </select>
          )}
        </div>

        {tab === "providers" ? (
          <div className="space-y-3">
            <div className="flex flex-wrap gap-2">
              <button
                onClick={() => setShowProviderModal(true)}
                className="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg border border-input bg-background text-sm font-medium text-foreground hover:bg-muted transition-colors"
              >
                <Settings2 className="h-4 w-4" />
                管理代理供应商
              </button>
              <button
                onClick={() => setShowChainModal(true)}
                className="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg border border-input bg-background text-sm font-medium text-foreground hover:bg-muted transition-colors"
              >
                <GitBranch className="h-4 w-4" />
                链式代理
              </button>
            </div>
            <div className={cn("grid grid-cols-1 gap-3", settings.doubleColumn && "2xl:grid-cols-2")}>
              {visibleProviders.length === 0 ? (
                <div className="rounded-lg border border-dashed border-border bg-card p-8 text-center text-sm text-muted-foreground 2xl:col-span-2">
                  {loading ? "正在加载供应商..." : "暂无代理供应商"}
                </div>
              ) : visibleProviders.map((p) => {
                const collapsed = readCollapsed("provider", p.name);
                const configuredChain = p.chainDialerProxy
                  ? { dialerProxy: p.chainDialerProxy, source: p.chainSource || `proxy-providers.${p.name}.override.dialer-proxy` }
                  : providerChainByName.get(p.name);
                const chainTarget = configuredChain ? chainTargetFor(configuredChain.dialerProxy) : null;
                return (
                  <div key={p.name} className="rounded-lg border border-border bg-card p-4">
                    <div className="flex items-center justify-between gap-2">
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <h3 className="font-semibold text-sm">
                            {p.name} <span className="text-xs text-muted-foreground font-normal">({p.now}/{p.total})</span>
                          </h3>
                          {configuredChain && (
                            <span
                              className={cn(
                                "rounded-full px-2 py-0.5 text-[11px] font-medium",
                                chainTarget?.tone === "warning"
                                  ? "bg-amber-500/10 text-amber-600 dark:text-amber-400"
                                  : "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                              )}
                              title={configuredChain.source}
                            >
                              {chainTarget?.tone === "warning" ? "链式目标未找到" : `链式 -> ${configuredChain.dialerProxy}`}
                            </span>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className="text-xs px-2 py-0.5 rounded-full bg-green-500/10 text-green-600 dark:text-green-400 font-medium">
                          存活 {p.alive}
                        </span>
                        <button onClick={() => void updateProvider(p.name)} className="p-1 rounded hover:bg-muted text-muted-foreground transition-colors" aria-label="更新">
                          <RotateCw className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => setItemCollapsed("provider", p.name, !collapsed)}
                          className="p-1 rounded hover:bg-muted text-muted-foreground transition-colors"
                          aria-label={collapsed ? "展开" : "折叠"}
                        >
                          {collapsed ? <ChevronDown className="h-4 w-4" /> : <ChevronUp className="h-4 w-4" />}
                        </button>
                      </div>
                    </div>
                    <div className="mt-3 flex items-center justify-between text-xs">
                      <span className="text-muted-foreground">
                        <span className="font-medium text-foreground">{p.used}</span> / {p.quota}
                      </span>
                      {p.extra && <span className="text-muted-foreground">{p.extra}</span>}
                      {p.expire && <span className="text-muted-foreground">{p.expire}</span>}
                    </div>
                    <div className="mt-1.5 h-1.5 w-full rounded-full bg-muted overflow-hidden">
                      <div className="h-full rounded-full bg-primary" style={{ width: `${Math.min(p.percent, 100)}%` }} />
                    </div>
                    <div className="mt-2 text-[11px] text-muted-foreground">{p.updated}</div>
                    {!collapsed && p.nodes.length > 0 && (
                      <div className="mt-3 flex flex-wrap items-start gap-2">
                        {p.nodes.map((node) => (
                          <ProxyNodeTile
                            key={node.name}
                            node={node}
                            loading={testing === node.name}
                            settings={settings}
                            onClick={() => void testNode(node.name)}
                            onTest={() => void testNode(node.name)}
                          />
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        ) : (
          <div className="space-y-3">
            <div className="flex">
              <button
                onClick={() => setShowGroupModal(true)}
                className="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg border border-input bg-background text-sm font-medium text-foreground hover:bg-muted transition-colors"
              >
                <Settings2 className="h-4 w-4" />
                管理代理分组
              </button>
            </div>
            <div className={cn("grid grid-cols-1 gap-3", settings.doubleColumn && "2xl:grid-cols-2")}>
            {visibleGroups.length === 0 ? (
              <div className="rounded-lg border border-dashed border-border bg-card p-8 text-center text-sm text-muted-foreground 2xl:col-span-2">
                {loading ? "正在加载代理组..." : "暂无代理组"}
              </div>
            ) : visibleGroups.map((g) => {
              const d = groupDelay(g);
              const isTesting = testing === g.name;
              const collapsed = readCollapsed("group", g.name);
              const isRelay = g.type.toLowerCase() === "relay";
              const displayNodes = settings.hideUnavailable
                ? g.nodes.filter((node) => node.alive && node.delay > 0)
                : g.nodes;
              return (
                <div
                  key={g.name}
                  className="rounded-lg border border-border bg-card transition-all duration-200 hover:shadow-sm hover:border-primary/30"
                >
                  <div className="px-3 py-2 select-none">
                    <div className="flex items-start gap-2">
                      <div className="shrink-0 self-center">
                        <img
                          alt={g.name}
                          className="h-11 w-11 rounded-lg object-cover"
                          referrerPolicy="no-referrer"
                          src={g.icon}
                        />
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <h3 className="font-semibold text-sm truncate">{g.name}</h3>
	                          <span className="text-xs text-muted-foreground">
	                            : {g.type || "Selector"} ({g.now}/{g.nodes.length})
	                          </span>
                          {isRelay && (
                            <span className="rounded-full bg-amber-500/10 px-2 py-0.5 text-[11px] font-medium text-amber-600 dark:text-amber-400">
                              relay 已弃用
                            </span>
                          )}
	                          {d > 0 && (
                            <span className={cn("text-sm font-medium ml-auto", delayColor(d, settings))}>{d}ms</span>
                          )}
                          <button
                            onClick={() => void testGroup(g.name)}
                            disabled={isTesting}
                            className={cn(
                              "p-1 rounded hover:bg-muted transition-colors disabled:opacity-50",
                              d > 0 ? "" : "ml-auto"
                            )}
                            aria-label="测试延迟"
                          >
                            {isTesting ? (
                              <Loader2 className="h-4 w-4 animate-spin text-primary" />
                            ) : (
                              <Gauge className="h-4 w-4" />
                            )}
                          </button>
                          <button
                            onClick={() => setItemCollapsed("group", g.name, !collapsed)}
                            className="p-1 rounded hover:bg-muted transition-colors"
                            aria-label={collapsed ? "展开" : "折叠"}
                          >
                            {collapsed ? <ChevronDown className="h-4 w-4 text-muted-foreground" /> : <ChevronUp className="h-4 w-4 text-muted-foreground" />}
                          </button>
                        </div>
                        <div className="mt-1">
                          <div className="flex items-center gap-3">
                            <div className="min-w-0 flex items-center gap-1.5">
                              <img
                                alt={g.selected}
                                className="h-4 w-4 rounded-sm shrink-0 object-contain"
                                referrerPolicy="no-referrer"
                                src={g.selectedIcon}
                              />
                              <span className={cn("text-xs text-muted-foreground", settings.nodeNameDisplay === "wrap" ? "break-all" : "truncate")}>{g.selected}</span>
                            </div>
                            <div className="flex flex-wrap gap-1 ml-auto">
                              {displayNodes.slice(0, 50).map((n) => (
                                <button
                                  key={n.name}
                                  onClick={() => void selectNode(g.name, n)}
                                  title={`${n.name} · ${n.delay === 0 ? "超时" : n.delay + "ms"}`}
                                  className={cn(
                                    "h-3 w-3 rounded-full transition-transform hover:scale-125",
                                    dotColor(n, settings),
                                    g.selected === n.name && "ring-2 ring-primary/60 ring-offset-1 ring-offset-card"
                                  )}
                                />
                              ))}
                              {displayNodes.length > 50 && (
                                <span className="text-[10px] text-muted-foreground">+{displayNodes.length - 50}</span>
                              )}
                            </div>
                          </div>
                          {!collapsed && displayNodes.length > 0 && (
                            <div className="mt-3 flex flex-wrap items-start gap-2">
                              {displayNodes.map((node) => (
                                <ProxyNodeTile
                                  key={node.name}
                                  node={node}
                                  active={g.selected === node.name}
                                  loading={testing === node.name}
                                  settings={settings}
                                  onClick={() => void selectNode(g.name, node)}
                                  onTest={() => void testNode(node.name)}
                                />
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
            </div>
          </div>
        )}
      </div>

      <ProxySettingsModal
        open={showSettings}
        settings={settings}
        onClose={() => setShowSettings(false)}
        onReset={() => setSettings(DEFAULT_PROXY_SETTINGS)}
        onChange={setSettings}
      />

      {showChainModal && (
        <ChainProxyModal
          providerChains={chainRecords.providerChains}
          proxyChains={chainRecords.proxyChains}
          targetFor={chainTargetFor}
          onClose={() => setShowChainModal(false)}
          onOpenConfig={openProfessionalConfig}
          onCopyExample={() => void copyChainExample()}
        />
      )}

      {showGroupModal && (
        <div
          className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4"
          onClick={() => setShowGroupModal(false)}
        >
          <div
            className="w-full max-w-[920px] max-h-[86vh] overflow-auto rounded-xl border border-border bg-card shadow-apple-xl p-5 animate-scale-in"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-start justify-between">
              <div>
                <h2 className="text-lg font-bold text-foreground">代理分组</h2>
                <p className="text-xs text-muted-foreground">保存后会切换为自定义配置并重启 Mihomo</p>
              </div>
              <button
                onClick={() => setShowGroupModal(false)}
                className="px-3 py-1.5 rounded-lg border border-input bg-background text-sm font-medium hover:bg-muted transition-colors flex items-center gap-1.5"
              >
                <X className="h-4 w-4" />
                关闭
              </button>
            </div>

            <div className="mt-4 space-y-3">
              {groupDrafts.length === 0 && (
                <div className="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">暂无代理分组</div>
              )}
              {groupDrafts.map((group, index) => (
                <div key={group.id} className="rounded-lg border border-border/60 bg-background p-3">
                  <div className="grid gap-2 md:grid-cols-[1fr_10rem_2fr_auto]">
                    <input
                      value={group.name}
                      onChange={(event) => setGroupDrafts((items) => items.map((item, i) => (i === index ? { ...item, name: event.target.value } : item)))}
                      placeholder="分组名称"
                      className="px-3 py-2 text-sm rounded-lg border border-border/60 bg-card focus:outline-none focus:border-primary/60"
                    />
                    <select
                      value={group.type}
                      onChange={(event) => setGroupDrafts((items) => items.map((item, i) => (i === index ? { ...item, type: event.target.value } : item)))}
                      className="px-3 py-2 text-sm rounded-lg border border-border/60 bg-card focus:outline-none focus:border-primary/60"
                    >
	                      {["select", "url-test", "fallback", "load-balance", "relay"].map((type) => (
	                        <option key={type} value={type}>{type === "relay" ? "relay（已弃用）" : type}</option>
	                      ))}
	                    </select>
                    <input
                      value={group.proxies}
                      onChange={(event) => setGroupDrafts((items) => items.map((item, i) => (i === index ? { ...item, proxies: event.target.value } : item)))}
                      placeholder="节点/分组，逗号分隔"
                      className="px-3 py-2 text-sm rounded-lg border border-border/60 bg-card font-mono focus:outline-none focus:border-primary/60"
                    />
                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => setGroupDrafts((items) => {
                          if (index === 0) return items;
                          const next = [...items];
                          [next[index - 1], next[index]] = [next[index], next[index - 1]];
                          return next;
                        })}
                        className="p-2 rounded-lg border border-border/60 text-muted-foreground hover:bg-muted"
                        aria-label="上移"
                      >
                        <ChevronUp className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() => setGroupDrafts((items) => {
                          if (index >= items.length - 1) return items;
                          const next = [...items];
                          [next[index + 1], next[index]] = [next[index], next[index + 1]];
                          return next;
                        })}
                        className="p-2 rounded-lg border border-border/60 text-muted-foreground hover:bg-muted"
                        aria-label="下移"
                      >
                        <ChevronDown className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() => setGroupDrafts((items) => items.filter((_, i) => i !== index))}
                        className="p-2 rounded-lg border border-border/60 text-muted-foreground hover:text-destructive hover:bg-destructive/10"
                        aria-label="删除"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
	                  <textarea
	                    value={group.extra}
                    onChange={(event) => setGroupDrafts((items) => items.map((item, i) => (i === index ? { ...item, extra: event.target.value } : item)))}
                    placeholder='高级 JSON，例如 {"url":"http://detectportal.firefox.com/success.txt","interval":120}'
	                    className="mt-2 min-h-20 w-full rounded-lg border border-border/60 bg-card px-3 py-2 font-mono text-xs focus:outline-none focus:border-primary/60"
	                  />
                    {group.type === "relay" && (
                      <div className="mt-2 rounded-lg bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-300">
                        relay 已被官方弃用，链式代理推荐使用 dialer-proxy 或 proxy-providers.override.dialer-proxy。
                      </div>
                    )}
	                </div>
	              ))}
            </div>

            <div className="mt-4 flex items-center justify-between gap-2">
              <button
                onClick={() => setGroupDrafts((items) => [...items, { id: `new-${Date.now()}`, name: "", type: "select", proxies: "DIRECT", extra: "" }])}
                className="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg border border-input bg-background text-sm font-medium hover:bg-muted transition-colors"
              >
                <Plus className="h-4 w-4" />
                新增分组
              </button>
              <button
                onClick={() => void saveGroupDrafts()}
                className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
              >
                <Save className="h-4 w-4" />
                保存并重启
              </button>
            </div>
          </div>
        </div>
      )}

      {showProviderModal && (
        <div
          className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4"
          onClick={() => setShowProviderModal(false)}
        >
          <div
            className="w-full max-w-[680px] rounded-xl border border-border bg-card shadow-apple-xl p-5 animate-scale-in"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-start justify-between">
              <div>
                <h2 className="text-lg font-bold text-foreground">代理供应商</h2>
                <p className="text-xs text-muted-foreground">当前配置文件</p>
              </div>
              <button
                onClick={() => setShowProviderModal(false)}
                className="px-3 py-1.5 rounded-lg border border-input bg-background text-sm font-medium hover:bg-muted transition-colors flex items-center gap-1.5"
              >
                <X className="h-4 w-4" />
                关闭
              </button>
            </div>

            <div className="mt-4 space-y-2">
              {sources.length === 0 && (
                <div className="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">暂无配置供应商</div>
              )}
              {sources.map((s, i) => (
                <div key={i} className="flex items-center gap-2">
                  <input
                    value={s.name}
                    onChange={(e) =>
                      setSources((arr) => arr.map((x, j) => (j === i ? { ...x, name: e.target.value } : x)))
                    }
                    placeholder="名称"
                    className="w-40 px-3 py-2 text-sm rounded-lg border border-border/60 bg-background focus:outline-none focus:border-primary/60"
                  />
                  <input
                    value={s.url}
                    onChange={(e) =>
                      setSources((arr) => arr.map((x, j) => (j === i ? { ...x, url: e.target.value } : x)))
                    }
                    placeholder="订阅链接（http/https）"
                    className="flex-1 px-3 py-2 text-sm rounded-lg border border-border/60 bg-background font-mono focus:outline-none focus:border-primary/60"
                  />
                  <button
                    onClick={() => setSources((arr) => arr.filter((_, j) => j !== i))}
                    className="p-2 rounded-lg border border-border/60 text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                    aria-label="删除"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>

            <div className="mt-4 flex items-center justify-between gap-2">
              <button
                onClick={() => setSources((arr) => [...arr, { name: "", url: "" }])}
                className="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg border border-input bg-background text-sm font-medium hover:bg-muted transition-colors"
              >
                <Plus className="h-4 w-4" />
                新增供应商
              </button>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => void saveSources(false)}
                  className="px-4 py-2 rounded-lg border border-input bg-background text-sm font-medium hover:bg-muted transition-colors"
                >
                  保存
                </button>
                <button
                  onClick={() => void saveSources(true)}
                  className="px-4 py-2 rounded-lg bg-primary text-primary-foreground text-sm font-medium hover:bg-primary/90 transition-colors"
                >
                  保存并重启
                </button>
              </div>
            </div>
            <p className="mt-2 text-[11px] text-muted-foreground">保存并重启会调用 Mihomo 重启接口以加载新的供应商配置。</p>
          </div>
        </div>
      )}

      <ToastStack toasts={toasts} />
    </AppShell>
  );
}
