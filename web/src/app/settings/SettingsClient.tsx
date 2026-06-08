"use client";

import { useEffect, useMemo, useRef, useState, type ReactNode, type SelectHTMLAttributes } from "react";
import {
  Check,
  Code2,
  Download,
  Eye,
  EyeOff,
  FileText,
  Languages,
  Menu,
  Monitor,
  Moon,
  Palette,
  Plus,
  RefreshCw,
  RotateCcw,
  Save,
  Settings,
  ShieldAlert,
  Share2,
  Sun,
  Trash2,
  TriangleAlert,
  Upload,
  User,
  X,
  type LucideIcon,
} from "lucide-react";
import { AppShell } from "@/components/AppShell";
import { ToastStack, useToaster } from "@/components/Toaster";
import { api, apiData, apiList } from "@/lib/api";
import { cn } from "@/lib/utils";

type TabId = "profile" | "system" | "appearance" | "update" | "reset";
type ThemeMode = "light" | "dark" | "system";
type NodeEditMode = "share" | "yaml";

interface SubscriptionRow {
  id: string;
  name: string;
  url: string;
}

interface InitConfigState {
  iface: string;
  proxyCore: "Mihomo" | "无";
  mihomoType: "Meta" | "Alpha";
  autoDns: boolean;
  dnsEnable: string;
  dnsDisable: string;
  ipv6: boolean;
  subscriptions: SubscriptionRow[];
  nodeMode: NodeEditMode;
  shareNodes: string[];
  yamlNodes: string;
}

interface VersionInfo {
  version?: string;
  go_version?: string;
  platform?: string;
  build_time?: string;
}

interface UpdateStatus {
  current_version?: string;
  latest_version?: string;
  has_update?: boolean;
  can_install?: boolean;
  status?: string;
  progress?: number;
  error_message?: string;
  download_url?: string;
  effective_download_url?: string;
  release_notes?: string;
  last_check_time?: string;
}

interface ReleaseAsset {
  name?: string;
  browser_download_url?: string;
}

interface ReleaseItem {
  tag_name?: string;
  name?: string;
  body?: string;
  html_url?: string;
  published_at?: string;
  assets?: ReleaseAsset[];
}

interface ComponentUpdateState {
  component?: string;
  current_version?: string;
  current_version_detail?: string;
  latest_version?: string;
  has_update?: boolean;
  can_update?: boolean;
  status?: string;
  progress?: number;
  error_message?: string;
  download_url?: string;
  last_check_time?: string;
}

interface UpdateConfigState {
  auto_check: boolean;
  check_interval: number;
  auto_update: boolean;
  notify: boolean;
  mosdns_upgrade_mode: "full" | "incremental" | "reset";
  mihomo_upgrade_mode: "skip" | "full";
}

interface ComponentUpdateConfigState {
  component: string;
  auto_check: boolean;
  check_interval: number;
  auto_update: boolean;
}

interface ProfileInfo {
  username?: string;
  display_name?: string;
  email?: string;
  role?: string;
}

interface NetworkInterfaceInfo {
  name: string;
  ip?: string;
  primary_ip?: string;
  addresses?: string[];
}

const tabs: Array<{ id: TabId; label: string; Icon: LucideIcon }> = [
  { id: "profile", label: "个人中心", Icon: User },
  { id: "system", label: "系统管理", Icon: Settings },
  { id: "appearance", label: "外观设置", Icon: Palette },
  { id: "update", label: "系统更新", Icon: RefreshCw },
  { id: "reset", label: "重置系统", Icon: ShieldAlert },
];

const themeOptions: Array<{ id: ThemeMode; label: string; Icon: LucideIcon }> = [
  { id: "light", label: "浅色模式", Icon: Sun },
  { id: "dark", label: "深色模式", Icon: Moon },
  { id: "system", label: "跟随系统", Icon: Monitor },
];

const RELEASE_REPO_OWNER = "scoltzero";
const RELEASE_REPO_NAME = "msf";
const RELEASE_REPO = `${RELEASE_REPO_OWNER}/${RELEASE_REPO_NAME}`;
const RELEASE_REPO_URL = `https://github.com/${RELEASE_REPO}`;

function toDate(value?: string) {
  if (!value) return null;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

function formatDate(value?: string) {
  const date = toDate(value);
  if (!date) return "-";
  return date.toLocaleDateString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).replace(/\//g, "-");
}

function formatDateTime(value?: string) {
  const date = toDate(value);
  if (!date) return "-";
  return date.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  }).replace(/\//g, "-");
}

function formatRelativeDate(value?: string) {
  const date = toDate(value);
  if (!date) return "-";
  const seconds = Math.round((date.getTime() - Date.now()) / 1000);
  const ranges: Array<[Intl.RelativeTimeFormatUnit, number]> = [
    ["year", 60 * 60 * 24 * 365],
    ["month", 60 * 60 * 24 * 30],
    ["day", 60 * 60 * 24],
    ["hour", 60 * 60],
    ["minute", 60],
  ];
  const formatter = new Intl.RelativeTimeFormat("zh-CN", { numeric: "auto" });
  for (const [unit, unitSeconds] of ranges) {
    if (Math.abs(seconds) >= unitSeconds || unit === "minute") {
      return formatter.format(Math.round(seconds / unitSeconds), unit);
    }
  }
  return "刚刚";
}

function releaseTitle(release?: ReleaseItem) {
  if (!release) return "-";
  return release.tag_name || release.name || "未命名版本";
}

function statusLabel(status?: string) {
  switch (status) {
    case "checked":
      return "已检查";
    case "downloading":
      return "下载中";
    case "downloaded":
      return "已下载";
    case "installing":
      return "安装中";
    case "failed":
      return "失败";
    case "running":
      return "运行中";
    case "completed":
      return "已完成";
    case "idle":
    default:
      return "空闲";
  }
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

function isTransientRestartError(error: unknown) {
  const message = errorMessage(error).toLowerCase();
  return [
    "terminated",
    "aborted",
    "aborterror",
    "load failed",
    "failed to fetch",
    "networkerror",
    "network error",
    "connection refused",
    "connection reset",
  ].some((part) => message.includes(part));
}

const defaultInitConfig: InitConfigState = {
  iface: "enp1s0 (192.168.10.3)",
  proxyCore: "Mihomo",
  mihomoType: "Meta",
  autoDns: true,
  dnsEnable: "127.0.0.1",
  dnsDisable: "223.5.5.5",
  ipv6: false,
  subscriptions: [
    {
      id: "kur",
      name: "kur",
      url: "https://example.invalid/subscription.yaml",
    },
    {
      id: "imm",
      name: "imm",
      url: "https://example.invalid/subscription.yaml",
    },
  ],
  nodeMode: "share",
  shareNodes: [
    "vless://00000000-0000-4000-8000-000000000000@example.invalid:443?encryption=none&security=reality&type=tcp&sni=example.invalid&fp=chrome#placeholder-node-1",
    "vless://00000000-0000-4000-8000-000000000001@example.invalid:443?encryption=none&security=reality&type=tcp&sni=example.invalid&fp=chrome#placeholder-node-2",
  ],
  yamlNodes:
    "proxies:\n  - name: placeholder-node-1\n    type: vless\n    server: 198.51.100.10\n    port: 28578\n    uuid: 00000000-0000-4000-8000-000000000000\n    network: tcp\n    tls: true\n    reality-opts:\n      public-key: xxx\n  - name: placeholder-node-2\n    type: vless\n    server: 203.0.113.10\n    port: 11451",
};

const defaultUpdateConfig: UpdateConfigState = {
  auto_check: true,
  check_interval: 43200,
  auto_update: false,
  notify: true,
  mosdns_upgrade_mode: "full",
  mihomo_upgrade_mode: "skip",
};

function parseSubscriptionValue(value: unknown): SubscriptionRow[] {
  if (Array.isArray(value)) {
    return value.flatMap((item, index) => subscriptionRowFromUnknown(item, index));
  }
  if (typeof value !== "string") return [];
  const trimmed = value.trim();
  if (!trimmed) return [];
  try {
    const parsed = JSON.parse(trimmed);
    if (Array.isArray(parsed)) return parsed.flatMap((item, index) => subscriptionRowFromUnknown(item, index));
  } catch {
    // Fall back to line-based input below.
  }
  return trimmed.split(/\s+/).flatMap((line, index) => subscriptionRowFromUnknown(line, index));
}

function subscriptionRowFromUnknown(value: unknown, index: number): SubscriptionRow[] {
  if (!value) return [];
  if (typeof value === "object" && !Array.isArray(value)) {
    const data = value as Record<string, unknown>;
    const url = String(data.url || data.subscription_url || data.subscriptionURL || "").trim();
    if (!url) return [];
    return [{
      id: `sub-${index + 1}`,
      name: String(data.name || data.tag || data.label || `订阅${index + 1}`).trim() || `订阅${index + 1}`,
      url,
    }];
  }
  const token = String(value).trim();
  if (!token) return [];
  const [rawName, ...rest] = token.split("|");
  const hasName = rest.length > 0;
  const url = hasName ? rest.join("|").trim() : token;
  if (!url) return [];
  return [{
    id: `sub-${index + 1}`,
    name: hasName && rawName.trim() ? rawName.trim() : `订阅${index + 1}`,
    url,
  }];
}

function serializeSubscriptions(rows: SubscriptionRow[]) {
  return rows
    .map((item) => {
      const url = item.url.trim();
      if (!url) return "";
      const name = item.name.trim().replace(/\|/g, "-");
      return name ? `${name}|${url}` : url;
    })
    .filter(Boolean)
    .join("\n");
}

function setupToInitConfig(raw: any): InitConfigState {
  const data = raw && typeof raw === "object" ? raw : {};
  const subscriptions = parseSubscriptionValue(data.subscription_urls || data.subscriptionURLs);
  const mihomoProxies = String(data.mihomo_proxies || data.mihomoProxies || "");
  const shareNodes = mihomoProxies && !mihomoProxies.trim().startsWith("proxies:")
    ? mihomoProxies.split(/\r?\n/).map((line) => line.trim()).filter(Boolean)
    : [];
  return {
    iface: String(data.selected_interface || data.selectedInterface || ""),
    proxyCore: String(data.proxy_core || data.proxyCore || "mihomo").toLowerCase() === "none" ? "无" : "Mihomo",
    mihomoType: String(data.mihomo_core_type || data.mihomoCoreType || "meta").toLowerCase() === "alpha" ? "Alpha" : "Meta",
    autoDns: data.auto_set_dns ?? data.autoSetDNS ?? true,
    dnsEnable: String(data.dns_on || data.dnsOn || "127.0.0.1"),
    dnsDisable: String(data.dns_off || data.dnsOff || "223.5.5.5"),
    ipv6: Boolean(data.enable_ipv6 ?? data.enableIPv6),
    subscriptions,
    nodeMode: mihomoProxies.trim().startsWith("proxies:") ? "yaml" : "share",
    shareNodes,
    yamlNodes: mihomoProxies.trim().startsWith("proxies:") ? mihomoProxies : "",
  };
}

function initConfigToSetupPayload(config: InitConfigState) {
  return {
    selected_interface: config.iface,
    proxy_core: config.proxyCore === "无" ? "none" : "mihomo",
    mihomo_core_type: config.mihomoType.toLowerCase(),
    auto_set_dns: config.autoDns,
    dns_on: config.dnsEnable,
    dns_off: config.dnsDisable,
    enable_ipv6: config.ipv6,
    subscription_urls: serializeSubscriptions(config.subscriptions),
    mihomo_proxies: config.nodeMode === "yaml" ? config.yamlNodes : config.shareNodes.filter(Boolean).join("\n"),
  };
}

function Card({
  title,
  Icon = FileText,
  children,
  className,
}: {
  title: string;
  Icon?: LucideIcon;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section
      className={cn(
        "rounded-[12px] border bg-card text-card-foreground !border-border/20 !shadow-none transition-shadow duration-300 hover:!shadow-sm",
        className
      )}
    >
      <div className="flex flex-col p-6 pb-3">
        <div className="flex items-center gap-2">
          <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-muted/60 text-muted-foreground">
            <Icon className="h-4 w-4" />
          </span>
          <h3 className="text-sm font-semibold leading-5 tracking-tight text-foreground">{title}</h3>
        </div>
      </div>
      <div className="p-6 pt-3">{children}</div>
    </section>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block space-y-1.5">
      <span className="block text-xs font-medium text-foreground">{label}</span>
      {children}
    </label>
  );
}

const inputClass =
  "w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground transition-all focus:border-primary focus:outline-none focus:ring-2 focus:ring-primary/20";

function Toggle({ checked, onChange, disabled = false }: { checked: boolean; onChange: (checked: boolean) => void; disabled?: boolean }) {
  return (
    <button
      type="button"
      onClick={() => onChange(!checked)}
      disabled={disabled}
      aria-pressed={checked}
      className={cn(
        "inline-flex h-6 w-11 shrink-0 items-center rounded-full border p-0.5 transition-colors",
        checked ? "border-primary bg-primary" : "border-border bg-muted",
        disabled && "cursor-not-allowed opacity-50"
      )}
    >
      <span
        className={cn(
          "h-5 w-5 rounded-full bg-white shadow transition-transform",
          checked ? "translate-x-5" : "translate-x-0"
        )}
      />
    </button>
  );
}

function PasswordInput({
  value,
  onChange,
  placeholder,
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
}) {
  const [show, setShow] = useState(false);

  return (
    <div className="relative">
      <input
        value={value}
        type={show ? "text" : "password"}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        className={`${inputClass} h-[38px] pr-12`}
      />
      <button
        type="button"
        aria-label={show ? "隐藏密码" : "显示密码"}
        title={show ? "隐藏密码" : "显示密码"}
        onClick={() => setShow((current) => !current)}
        className="absolute right-0 top-1/2 flex h-11 w-11 -translate-y-1/2 items-center justify-center text-muted-foreground transition-colors hover:text-foreground md:h-8 md:w-8"
      >
        {show ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
      </button>
    </div>
  );
}

function PlainInfo({ label, value, badge }: { label: string; value: string; badge?: boolean }) {
  return (
    <div className="min-w-0">
      <div className="text-xs leading-4 text-muted-foreground">{label}</div>
      {badge ? (
        <span className="mt-1 inline-flex rounded-md bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary">{value}</span>
      ) : (
        <div className="mt-1 truncate text-sm font-medium leading-5 text-foreground">{value}</div>
      )}
    </div>
  );
}

function PrimaryButton({ children, className, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      {...props}
      className={cn(
        "inline-flex h-9 items-center justify-center gap-2 rounded-[10px] bg-primary px-4 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50",
        className
      )}
    >
      {children}
    </button>
  );
}

function OutlineButton({ children, className, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      {...props}
      className={cn(
        "inline-flex h-8 items-center justify-center gap-2 rounded-[10px] border border-border bg-background px-3 text-xs font-medium text-foreground transition-colors hover:bg-muted",
        className
      )}
    >
      {children}
    </button>
  );
}

function ProfileTab({ showToast }: { showToast: (message: string) => void }) {
  const [profile, setProfile] = useState<ProfileInfo>({});
  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [saving, setSaving] = useState(false);
  const profileDirty = displayName !== (profile.display_name || "") || email !== (profile.email || "");
  const passwordReady = currentPassword.length > 0 && newPassword.length >= 8 && newPassword === confirmPassword;

  useEffect(() => {
    api("/api/v1/profile")
      .then((payload) => {
        const data = apiData<ProfileInfo>(payload, {});
        setProfile(data);
        setDisplayName(data.display_name || "");
        setEmail(data.email || "");
      })
      .catch((error) => showToast(errorMessage(error)));
  }, [showToast]);

  const saveProfile = async () => {
    setSaving(true);
    try {
      const payload = await api("/api/v1/profile", {
        method: "PUT",
        body: JSON.stringify({ display_name: displayName, email }),
      });
      const data = apiData<ProfileInfo>(payload, {});
      setProfile(data);
      setDisplayName(data.display_name || "");
      setEmail(data.email || "");
      showToast("个人资料已保存");
    } catch (error) {
      showToast(errorMessage(error));
    } finally {
      setSaving(false);
    }
  };

  const changePassword = async () => {
    setSaving(true);
    try {
      await api("/api/v1/profile/password", {
        method: "POST",
        body: JSON.stringify({ old_password: currentPassword, new_password: newPassword }),
      });
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      showToast("密码已修改");
    } catch (error) {
      showToast(errorMessage(error));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-4">
      <Card title="个人信息" Icon={User}>
        <div className="grid gap-4 md:grid-cols-4">
          <PlainInfo label="用户名" value={profile.username || "-"} />
          <PlainInfo label="角色" value={profile.role || "-"} badge />
          <PlainInfo label="显示名称" value={profile.display_name || "未设置"} />
          <PlainInfo label="邮箱" value={profile.email || "未设置"} />
        </div>
      </Card>

      <Card title="编辑个人资料" Icon={FileText}>
        <div className="flex flex-col gap-4">
          <div className="grid max-w-[672px] gap-4 md:grid-cols-2">
            <Field label="显示名称">
              <input
                value={displayName}
                onChange={(event) => setDisplayName(event.target.value)}
                placeholder="请输入显示名称"
                className={`${inputClass} h-[38px]`}
              />
            </Field>
            <Field label="邮箱">
              <input
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                type="email"
                placeholder="user@example.com"
                className={`${inputClass} h-[38px]`}
              />
            </Field>
          </div>
          <div className="flex justify-end">
            <PrimaryButton disabled={!profileDirty || saving} onClick={saveProfile} className="h-8 px-4 md:h-8">
              <Save className="h-3.5 w-3.5" />
              保存
            </PrimaryButton>
          </div>
        </div>
      </Card>

      <Card title="修改密码" Icon={ShieldAlert}>
        <div className="grid gap-4 md:grid-cols-3 md:pr-[318px]">
          <Field label="当前密码">
            <PasswordInput value={currentPassword} onChange={setCurrentPassword} placeholder="请输入当前密码" />
          </Field>
          <Field label="新密码">
            <PasswordInput value={newPassword} onChange={setNewPassword} placeholder="请输入新密码" />
          </Field>
          <Field label="确认密码">
            <PasswordInput value={confirmPassword} onChange={setConfirmPassword} placeholder="请再次输入新密码" />
          </Field>
        </div>
        <div className="mt-4 flex items-center justify-between gap-3">
          <p className="text-xs text-muted-foreground">密码至少 8 位，区分大小写</p>
          <PrimaryButton
            disabled={!passwordReady || saving}
            onClick={changePassword}
            className="h-8 shrink-0 px-4"
          >
            <Save className="h-3.5 w-3.5" />
            修改密码
          </PrimaryButton>
        </div>
      </Card>
    </div>
  );
}

function InitConfigSummary({
  config,
  visibleSecrets,
  onReveal,
  onEdit,
}: {
  config: InitConfigState;
  visibleSecrets: Record<string, boolean>;
  onReveal: (id: string) => void;
  onEdit: () => void;
}) {
  return (
    <>
      <div className="mb-4 flex justify-end">
        <OutlineButton onClick={onEdit}>编辑配置</OutlineButton>
      </div>
      <div className="grid gap-x-4 gap-y-4 md:grid-cols-4">
        <PlainInfo label="选定网卡" value={config.iface.replace(" (192.168.10.3)", "")} />
        <PlainInfo label="代理核心类型" value={`${config.proxyCore} (${config.mihomoType.toLowerCase()})`} />
        <PlainInfo label="自动设置 DNS" value={config.autoDns ? "已启用" : "已禁用"} />
        <PlainInfo label="DNS 启用地址" value={config.dnsEnable} />
        <PlainInfo label="DNS 禁用地址" value={config.dnsDisable} />
        <PlainInfo label="启用 IPv6" value={config.ipv6 ? "已启用" : "DNS自动设置"} />
        <PlainInfo label="运行时DNS" value="" />
      </div>
      <div className="mt-5 grid gap-3 md:grid-cols-2">
        {config.subscriptions.map((item) => (
          <div key={item.id} className="flex items-center gap-2 text-sm">
            <span className="font-mono text-muted-foreground">{item.name}:</span>
            <span className="min-w-0 flex-1 truncate font-mono text-foreground">
              {visibleSecrets[item.id] ? item.url : "********"}
            </span>
            <button
              onClick={() => onReveal(item.id)}
              className="flex h-6 w-6 items-center justify-center text-muted-foreground hover:text-foreground"
              aria-label="显示订阅"
            >
              {visibleSecrets[item.id] ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
            </button>
          </div>
        ))}
      </div>
    </>
  );
}

function SectionBox({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("rounded-xl border border-border/60 bg-background/40 p-4 md:p-5", className)}>{children}</div>;
}

function InitConfigEditor({
  draft,
  setDraft,
  onCancel,
  onSave,
  networkInterfaces,
}: {
  draft: InitConfigState;
  setDraft: (updater: (current: InitConfigState) => InitConfigState) => void;
  onCancel: () => void;
  onSave: () => void;
  networkInterfaces: NetworkInterfaceInfo[];
}) {
  const updateSub = (id: string, patch: Partial<SubscriptionRow>) => {
    setDraft((current) => ({
      ...current,
      subscriptions: current.subscriptions.map((item) => (item.id === id ? { ...item, ...patch } : item)),
    }));
  };

  const removeSub = (id: string) => {
    setDraft((current) => ({ ...current, subscriptions: current.subscriptions.filter((item) => item.id !== id) }));
  };

  const addSub = () => {
    setDraft((current) => ({
      ...current,
      subscriptions: [...current.subscriptions, { id: `sub-${current.subscriptions.length + 1}`, name: "", url: "" }],
    }));
  };

  const updateNode = (index: number, value: string) => {
    setDraft((current) => ({ ...current, shareNodes: current.shareNodes.map((node, i) => (i === index ? value : node)) }));
  };

  const removeNode = (index: number) => {
    setDraft((current) => ({ ...current, shareNodes: current.shareNodes.filter((_, i) => i !== index) }));
  };

  const addNode = () => {
    setDraft((current) => ({ ...current, shareNodes: [...current.shareNodes, ""] }));
  };

  return (
    <div className="space-y-4 animate-slide-up">
      <SectionBox>
        <h3 className="text-lg font-semibold text-foreground">选定网卡</h3>
        <p className="mt-1 text-sm text-muted-foreground">网卡接口</p>
        <select
          value={draft.iface}
          onChange={(event) => setDraft((current) => ({ ...current, iface: event.target.value }))}
          className={`${inputClass} mt-4 h-12 text-base`}
        >
          <option>请选择网卡接口</option>
          {networkInterfaces.map((iface) => (
            <option key={iface.name} value={iface.name}>
              {iface.name}{iface.primary_ip || iface.ip ? ` (${iface.primary_ip || iface.ip})` : ""}
            </option>
          ))}
        </select>
      </SectionBox>

      <SectionBox>
        <h3 className="text-lg font-semibold text-foreground">代理核心配置</h3>
        <p className="mt-1 text-sm text-muted-foreground">选择代理服务核心及其类型</p>
        <div className="mt-4 space-y-4">
          <Field label="选择代理服务核心及其类型">
            <select
              value={draft.proxyCore}
              onChange={(event) => setDraft((current) => ({ ...current, proxyCore: event.target.value as InitConfigState["proxyCore"] }))}
              className={`${inputClass} h-12 text-base`}
            >
              <option>请选择代理核心</option>
              <option disabled>代理核心 *</option>
              <option>Mihomo</option>
              <option>无</option>
            </select>
          </Field>
          <div className="border-t border-border/60 pt-4">
            <Field label="Mihomo 核心类型">
              <select
                value={draft.mihomoType}
                onChange={(event) => setDraft((current) => ({ ...current, mihomoType: event.target.value as InitConfigState["mihomoType"] }))}
                className={`${inputClass} h-12 text-base`}
              >
                <option>请选择核心类型</option>
                <option>Meta</option>
                <option>Alpha</option>
              </select>
            </Field>
          </div>
        </div>
      </SectionBox>

      <SectionBox>
        <label className="flex items-center gap-3 text-lg font-semibold text-foreground">
          <input
            type="checkbox"
            checked={draft.autoDns}
            onChange={(event) => setDraft((current) => ({ ...current, autoDns: event.target.checked }))}
            className="h-5 w-5 accent-primary"
          />
          自动设置 DNS
        </label>
        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <Field label="DNS 启用地址">
            <input
              value={draft.dnsEnable}
              onChange={(event) => setDraft((current) => ({ ...current, dnsEnable: event.target.value }))}
              placeholder="例如: 127.0.0.1"
              className={`${inputClass} h-12 text-base`}
            />
            <p className="mt-1 text-xs text-muted-foreground">例如: 127.0.0.1</p>
          </Field>
          <Field label="DNS 禁用地址">
            <input
              value={draft.dnsDisable}
              onChange={(event) => setDraft((current) => ({ ...current, dnsDisable: event.target.value }))}
              placeholder="例如: 8.8.8.8"
              className={`${inputClass} h-12 text-base`}
            />
            <p className="mt-1 text-xs text-muted-foreground">例如: 127.0.0.1</p>
          </Field>
        </div>
      </SectionBox>

      <SectionBox>
        <label className="flex items-center gap-3 text-lg font-semibold text-foreground">
          <input
            type="checkbox"
            checked={draft.ipv6}
            onChange={(event) => setDraft((current) => ({ ...current, ipv6: event.target.checked }))}
            className="h-5 w-5 accent-primary"
          />
          启用 IPv6
        </label>
        <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
          开启后代理核心将支持 IPv6 流量处理，关闭则仅处理 IPv4 流量。如果您的网络不支持 IPv6，请务必关闭此选项
        </p>
      </SectionBox>

      <SectionBox>
        <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 className="text-lg font-semibold text-foreground">订阅链接</h3>
            <p className="mt-1 text-sm text-muted-foreground">订阅链接配置</p>
          </div>
          <OutlineButton onClick={addSub} className="h-10 border-dashed text-sm">
            <Plus className="h-4 w-4" />
            添加订阅
          </OutlineButton>
        </div>
        <div className="space-y-4">
          {draft.subscriptions.map((item, index) => (
            <div key={item.id} className="grid gap-3 md:grid-cols-[1fr_1.45fr_auto] md:items-end">
              <Field label={index === 0 ? "名称 (可选)" : "名称"}>
                <input
                  value={item.name}
                  onChange={(event) => updateSub(item.id, { name: event.target.value })}
                  placeholder={`✈️机场${index + 1}`}
                  className={`${inputClass} h-12 text-base`}
                />
              </Field>
              <Field label="订阅地址 *">
                <input
                  value={item.url}
                  onChange={(event) => updateSub(item.id, { url: event.target.value })}
                  placeholder="https://..."
                  className={`${inputClass} h-12 text-base`}
                />
              </Field>
              <button
                onClick={() => removeSub(item.id)}
                className="inline-flex h-12 items-center justify-center rounded-lg px-3 text-destructive hover:bg-destructive/10"
                aria-label="删除订阅"
              >
                <Trash2 className="h-5 w-5" />
              </button>
            </div>
          ))}
        </div>
        <p className="mt-3 text-sm text-muted-foreground">✈️机场1</p>
      </SectionBox>

      <SectionBox>
        <div className="mb-4 flex items-start gap-3">
          <Settings className="mt-0.5 h-5 w-5 text-muted-foreground" />
          <div>
            <h3 className="text-lg font-semibold text-foreground">自定义节点（可选）</h3>
            <p className="mt-1 text-sm text-muted-foreground">自定义节点配置</p>
          </div>
        </div>
        <div className="inline-flex rounded-xl border border-border bg-muted/40 p-1 shadow-sm">
          {[
            { id: "share" as const, label: "分享链接模式", Icon: Share2 },
            { id: "yaml" as const, label: "YAML 文本模式", Icon: Code2 },
          ].map(({ id, label, Icon }) => (
            <button
              key={id}
              onClick={() => setDraft((current) => ({ ...current, nodeMode: id }))}
              className={cn(
                "inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition-all",
                draft.nodeMode === id ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"
              )}
            >
              <Icon className="h-4 w-4" />
              {label}
            </button>
          ))}
        </div>
        <div className="mt-4 rounded-lg border border-blue-500/40 bg-blue-500/10 px-4 py-3 text-sm text-blue-700 dark:text-blue-200">
          {draft.nodeMode === "share"
            ? "分享链接模式：支持协议：ss、ssr、trojan、vmess、vless、hysteria、hysteria2、tuic"
            : "YAML/文本模式：按照官方文档格式编写，可配置所有高级参数"}
        </div>
        {draft.nodeMode === "share" ? (
          <div className="mt-4 space-y-3">
            {draft.shareNodes.map((node, index) => (
              <div key={index} className="flex gap-3 rounded-lg border border-border/50 bg-card p-3">
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 font-semibold text-primary">{index + 1}</div>
                <input
                  value={node}
                  onChange={(event) => updateNode(index, event.target.value)}
                  placeholder="ss:// / trojan:// / vmess:// / vless:// / hysteria2:// / tuic:// ..."
                  className={`${inputClass} h-10 min-w-0 flex-1`}
                />
                <button
                  onClick={() => removeNode(index)}
                  className="rounded-lg px-2 text-destructive hover:bg-destructive/10"
                  aria-label="删除此节点"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            ))}
            <div className="flex flex-wrap items-center gap-3">
              <OutlineButton onClick={addNode} className="h-10 border-dashed text-sm">
                <Plus className="h-4 w-4" />
                添加节点
              </OutlineButton>
              <span className="text-sm text-muted-foreground">已添加 {draft.shareNodes.length} 条</span>
            </div>
          </div>
        ) : (
          <div className="mt-4">
            <textarea
              value={draft.yamlNodes}
              onChange={(event) => setDraft((current) => ({ ...current, yamlNodes: event.target.value }))}
              placeholder={'proxies:\n  - name: "my-node"\n    type: trojan\n    server: example.com\n    port: 443\n    password: "xxx"\n    sni: example.com'}
              className={`${inputClass} min-h-56 font-mono leading-relaxed`}
            />
            <p className="mt-2 text-xs text-muted-foreground">💡 提示：可以直接从 Mihomo 配置文件复制 proxies 部分粘贴到这里</p>
          </div>
        )}
      </SectionBox>

      <div className="flex gap-3 pt-2">
        <OutlineButton onClick={onCancel} className="h-11 px-5 text-sm">
          取消
        </OutlineButton>
        <PrimaryButton onClick={onSave} className="h-11 px-5">
          <Save className="h-4 w-4" />
          保存配置
        </PrimaryButton>
      </div>
    </div>
  );
}

function SystemTab({ showToast }: { showToast: (message: string) => void }) {
  const [retention, setRetention] = useState(24);
  const [visibleSecrets, setVisibleSecrets] = useState<Record<string, boolean>>({});
  const [editingInit, setEditingInit] = useState(false);
  const [initConfig, setInitConfig] = useState(defaultInitConfig);
  const [draftConfig, setDraftConfig] = useState(defaultInitConfig);
  const [networkInterfaces, setNetworkInterfaces] = useState<NetworkInterfaceInfo[]>([]);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    Promise.allSettled([
      api("/api/v1/settings"),
      api("/api/v1/setup/config"),
      api("/api/v1/setup/network-interfaces"),
    ]).then(([settingsResult, configResult, interfacesResult]) => {
      if (settingsResult.status === "fulfilled") {
        const data = apiData<Record<string, string>>(settingsResult.value, {});
        const hours = Number(data.token_retention_hours || data.jwt_expire_hours || data.jwt_expiry_hours || data.token_ttl_hours || 24);
        if (Number.isFinite(hours) && hours > 0) setRetention(hours);
      }
      if (interfacesResult.status === "fulfilled") {
        setNetworkInterfaces(apiList<NetworkInterfaceInfo>(interfacesResult.value, ["data", "interfaces", "items"]));
      }
      if (configResult.status === "fulfilled") {
        const config = setupToInitConfig(apiData<any>(configResult.value, configResult.value));
        setInitConfig(config);
        setDraftConfig(config);
      } else {
        showToast(errorMessage(configResult.reason));
      }
    });
  }, [showToast]);

  const saveSystemSettings = async () => {
    setSaving(true);
    try {
      await api("/api/v1/settings", {
        method: "PUT",
        body: JSON.stringify({ token_retention_hours: retention }),
      });
      showToast("系统配置已保存");
    } catch (error) {
      showToast(errorMessage(error));
    } finally {
      setSaving(false);
    }
  };

  const saveInitConfig = async () => {
    setSaving(true);
    try {
      await api("/api/v1/setup/config", {
        method: "PUT",
        body: JSON.stringify(initConfigToSetupPayload(draftConfig)),
      });
      setInitConfig(draftConfig);
      setEditingInit(false);
      showToast("初始化配置已保存");
    } catch (error) {
      showToast(errorMessage(error));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-4">
      <Card title="系统配置" Icon={Settings}>
        <div className="grid gap-4 md:grid-cols-[460px_auto] md:items-center md:justify-start">
          <div>
            <div className="text-sm font-medium text-foreground">Token 有效期(小时)</div>
            <p className="mt-1 text-xs text-muted-foreground">范围: 1-720 小时,默认 24 小时</p>
          </div>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <input
              type="number"
              min={1}
              max={720}
              value={retention}
              onChange={(event) => setRetention(Number(event.target.value))}
              className={`${inputClass} h-[38px] sm:w-24`}
            />
            <PrimaryButton onClick={saveSystemSettings} disabled={saving} className="h-9 px-4">
              <Save className="h-3.5 w-3.5" />
              保存
            </PrimaryButton>
          </div>
        </div>
      </Card>

      <Card title="初始化配置" Icon={Settings}>
        {editingInit ? (
          <InitConfigEditor
            draft={draftConfig}
            setDraft={setDraftConfig}
            networkInterfaces={networkInterfaces}
            onCancel={() => {
              setDraftConfig(initConfig);
              setEditingInit(false);
            }}
            onSave={saveInitConfig}
          />
        ) : (
          <InitConfigSummary
            config={initConfig}
            visibleSecrets={visibleSecrets}
            onReveal={(id) => setVisibleSecrets((current) => ({ ...current, [id]: !current[id] }))}
            onEdit={() => {
              setDraftConfig(initConfig);
              setEditingInit(true);
            }}
          />
        )}
      </Card>
    </div>
  );
}

function AppearanceTab({ showToast }: { showToast: (message: string) => void }) {
  const [theme, setTheme] = useState<ThemeMode>("system");
  const [language, setLanguage] = useState("简体中文");

  const applyThemeMode = (mode: ThemeMode) => {
    setTheme(mode);
    const dark = mode === "system" ? window.matchMedia("(prefers-color-scheme: dark)").matches : mode === "dark";
    document.documentElement.classList.toggle("dark", dark);
    document.documentElement.classList.toggle("light", !dark);
    localStorage.setItem("msf-theme", mode);
  };

  useEffect(() => {
    api("/api/v1/settings/appearance")
      .then((payload) => {
        const data = apiData<Record<string, string>>(payload, {});
        const nextTheme = (data.theme === "light" || data.theme === "dark" || data.theme === "system" ? data.theme : "system") as ThemeMode;
        applyThemeMode(nextTheme);
        setLanguage(data.language === "en-US" || data.language === "en" ? "English" : "简体中文");
      })
      .catch((error) => showToast(errorMessage(error)));
  }, [showToast]);

  const saveAppearance = async (patch: Record<string, string>) => {
    try {
      await api("/api/v1/settings/appearance", {
        method: "PUT",
        body: JSON.stringify(patch),
      });
      showToast("外观设置已保存");
    } catch (error) {
      showToast(errorMessage(error));
    }
  };

  const setThemeMode = (mode: ThemeMode) => {
    applyThemeMode(mode);
    void saveAppearance({ theme: mode });
  };

  return (
    <div className="space-y-4">
      <Card title="主题模式" Icon={Palette}>
        <div className="grid max-w-[672px] gap-3 md:grid-cols-3">
          {themeOptions.map(({ id, label, Icon }) => (
            <button
              key={id}
              onClick={() => setThemeMode(id)}
              className={cn(
                "flex h-[84px] items-center justify-center gap-2 rounded-lg border-2 px-4 text-sm font-medium transition-all",
                theme === id ? "border-primary bg-primary/10 text-foreground" : "border-border bg-transparent text-foreground hover:bg-muted/40"
              )}
            >
              <Icon className="h-5 w-5" />
              {label}
            </button>
          ))}
        </div>
      </Card>

      <Card title="语言 / Language" Icon={Languages}>
        <div className="grid max-w-[576px] gap-3 md:grid-cols-2">
          {["简体中文", "English"].map((item) => (
            <button
              key={item}
              onClick={() => {
                setLanguage(item);
                void saveAppearance({ language: item === "English" ? "en-US" : "zh-CN" });
              }}
              className={cn(
                "flex h-[84px] items-center justify-center gap-2 rounded-lg border-2 px-4 text-sm font-medium transition-all",
                language === item ? "border-primary bg-primary/10 text-foreground" : "border-border bg-transparent text-foreground hover:bg-muted/40"
              )}
            >
              <Languages className="h-5 w-5" />
              {item}
            </button>
          ))}
        </div>
      </Card>

      <Card title="菜单顺序" Icon={Menu}>
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <p className="text-sm text-muted-foreground">重置菜单顺序到默认状态</p>
          <OutlineButton onClick={() => void saveAppearance({ menu_order: "" })} className="h-8 text-sm">
            <RotateCcw className="h-4 w-4" />
            菜单顺序已重置
          </OutlineButton>
        </div>
      </Card>
    </div>
  );
}

function UpdateInfoTile({ label, value, badge }: { label: string; value: string; badge?: string }) {
  return (
    <div className="flex min-h-7 items-center justify-between gap-4 border-b border-border/40 py-1.5 last:border-b-0">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="flex items-center gap-2 text-right text-xs font-semibold text-foreground">
        {value}
        {badge ? <span className="rounded-full bg-green-500/10 px-2 py-0.5 text-xs text-green-600">{badge}</span> : null}
      </div>
    </div>
  );
}

function UpdateConfigRow({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 rounded-lg border border-border/50 bg-muted/20 p-3 md:flex-row md:items-center md:justify-between">
      <div>
        <div className="text-sm font-medium text-foreground">{title}</div>
        <p className="mt-1 text-xs text-muted-foreground">{description}</p>
      </div>
      {children}
    </div>
  );
}

function SmallSelect({ children, className, disabled = false, ...props }: SelectHTMLAttributes<HTMLSelectElement> & { children: ReactNode }) {
  return (
    <select
      {...props}
      disabled={disabled}
      className={cn(inputClass, "h-[27px] w-24 py-1 text-xs disabled:opacity-50", className)}
    >
      {children}
    </select>
  );
}

const updateIntervals = [
  { value: 43200, label: "12 小时" },
  { value: 86400, label: "24 小时" },
  { value: 259200, label: "3 天" },
  { value: 604800, label: "7 天" },
];

function ComponentUpdateCard({
  name,
  component,
  item,
  config,
  busy,
  onUpload,
  onCheck,
  onUpdate,
  onConfigChange,
}: {
  name: string;
  component: string;
  item?: ComponentUpdateState;
  config?: ComponentUpdateConfigState;
  busy?: string;
  onUpload: (component: string, file: File) => void;
  onCheck: (component: string) => void;
  onUpdate: (component: string) => void;
  onConfigChange: (component: string, patch: Partial<ComponentUpdateConfigState>) => void;
}) {
  const fileRef = useRef<HTMLInputElement>(null);
  const current = item?.current_version || "-";
  const currentDetail = item?.current_version_detail || "";
  const latest = item?.latest_version || "-";
  const hasUpdate = Boolean(item?.has_update);
  const canUpdate = Boolean(item?.can_update || item?.has_update);
  const isBusy = busy === component;
  const progress = typeof item?.progress === "number" ? item.progress : 0;
  const effectiveConfig = config || { component, auto_check: true, check_interval: 43200, auto_update: false };

  return (
    <div className="rounded-xl border border-border/50 p-4">
      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <div className="flex items-center gap-2 font-semibold text-foreground">
            {name}
            {hasUpdate ? <span className="rounded-full bg-green-500/10 px-2 py-0.5 text-xs text-green-600">可更新</span> : null}
            {!hasUpdate && canUpdate ? <span className="rounded-full bg-blue-500/10 px-2 py-0.5 text-xs text-blue-600">可覆盖</span> : null}
          </div>
          <div className="mt-1 text-xs text-muted-foreground">
            状态: {statusLabel(item?.status)} · 最后检查: {formatDateTime(item?.last_check_time)}
          </div>
          {item?.error_message ? <div className="mt-1 text-xs text-destructive">{item.error_message}</div> : null}
        </div>
        <div className="flex shrink-0 items-center gap-2 sm:pt-0.5">
          <input
            ref={fileRef}
            type="file"
            className="hidden"
            accept={component === "zashboard" ? ".zip" : undefined}
            onChange={(event) => {
              const file = event.target.files?.[0];
              if (file) onUpload(component, file);
              event.currentTarget.value = "";
            }}
          />
          <OutlineButton disabled={isBusy} onClick={() => fileRef.current?.click()} className="h-8 min-w-[76px] px-3 text-xs">
            <Upload className="h-3.5 w-3.5" />
            本地上传
          </OutlineButton>
          <OutlineButton disabled={isBusy} onClick={() => onCheck(component)} className="h-8 min-w-[76px] px-3 text-xs">
            <RefreshCw className={cn("h-3.5 w-3.5", isBusy && "animate-spin")} />
            检查更新
          </OutlineButton>
          <PrimaryButton disabled={isBusy || !canUpdate} onClick={() => onUpdate(component)} className="h-8 min-w-[52px] px-3 text-xs">
            {hasUpdate ? "更新" : "覆盖更新"}
          </PrimaryButton>
        </div>
      </div>
      {progress > 0 ? (
        <div className="mb-3">
          <div className="mb-1 flex justify-between text-xs text-muted-foreground">
            <span>进度</span>
            <span>{progress}%</span>
          </div>
          <div className="h-1.5 rounded-full bg-muted">
            <div className="h-full rounded-full bg-primary" style={{ width: `${Math.min(100, Math.max(0, progress))}%` }} />
          </div>
        </div>
      ) : null}
      <div className="grid gap-2 text-sm md:grid-cols-2">
        <div className="rounded-lg bg-muted/20 p-3">
          <div className="text-xs text-muted-foreground">当前版本</div>
          <div className="mt-1 font-mono text-foreground">{current}</div>
          <div className="mt-1 break-all text-xs text-muted-foreground">{currentDetail ? `详情: ${currentDetail}` : "时间: -"}</div>
        </div>
        <div className="rounded-lg bg-muted/20 p-3">
          <div className="text-xs text-muted-foreground">最新版本</div>
          <div className="mt-1 font-mono text-foreground">{latest}</div>
          <div className="mt-1 text-xs text-muted-foreground">时间: {formatDateTime(item?.last_check_time)}</div>
        </div>
      </div>
      <div className="mt-4 space-y-3">
        <UpdateConfigRow title="自动检查" description="定期检查是否有新版本可用">
          <Toggle checked={effectiveConfig.auto_check} onChange={(checked) => onConfigChange(component, { auto_check: checked })} />
        </UpdateConfigRow>
        <UpdateConfigRow title="检查间隔" description="设置检查更新的时间间隔">
          <SmallSelect
            value={String(effectiveConfig.check_interval)}
            onChange={(event) => onConfigChange(component, { check_interval: Number(event.target.value) })}
          >
            {updateIntervals.map((option) => (
              <option key={option.value} value={option.value}>{option.label}</option>
            ))}
          </SmallSelect>
        </UpdateConfigRow>
        <UpdateConfigRow title="自动更新" description="检测到新版本后自动下载">
          <Toggle checked={effectiveConfig.auto_update} onChange={(checked) => onConfigChange(component, { auto_update: checked })} />
        </UpdateConfigRow>
      </div>
    </div>
  );
}

function UpdateTab({ showToast }: { showToast: (message: string) => void }) {
  const [checking, setChecking] = useState(false);
  const [loadingReleases, setLoadingReleases] = useState(false);
  const [updateConfig, setUpdateConfig] = useState<UpdateConfigState>(defaultUpdateConfig);
  const [releaseModal, setReleaseModal] = useState<string | null>(null);
  const [versionInfo, setVersionInfo] = useState<VersionInfo>({});
  const [updateStatus, setUpdateStatus] = useState<UpdateStatus>({});
  const [releases, setReleases] = useState<ReleaseItem[]>([]);
  const [componentUpdates, setComponentUpdates] = useState<ComponentUpdateState[]>([]);
  const [componentConfigs, setComponentConfigs] = useState<Record<string, ComponentUpdateConfigState>>({});
  const [componentBusy, setComponentBusy] = useState("");
  const [repoError, setRepoError] = useState("");
  const [restartPending, setRestartPending] = useState(false);
  const restartRefreshTimer = useRef<number | null>(null);
  const restartRefreshAttempts = useRef(0);

  const latestRelease = releases[0];
  const displayUpdateStatus = restartPending
    ? {
        ...updateStatus,
        status: "installing",
        progress: Math.max(typeof updateStatus.progress === "number" ? updateStatus.progress : 0, 95),
        error_message: "",
      }
    : updateStatus;
  const currentVersion = versionInfo.version || "";
  const latestVersion = latestRelease ? releaseTitle(latestRelease) : "";
  const hasUpdate = Boolean(latestVersion) && Boolean(displayUpdateStatus.has_update);
  const canInstallUpdate = !restartPending && Boolean(displayUpdateStatus.can_install || displayUpdateStatus.status === "downloaded");
  const installingUpdate = restartPending || displayUpdateStatus.status === "installing";
  const selectedRelease = useMemo(
    () =>
      releases.find((release, index) => {
        const key = release.tag_name || release.html_url || release.name || String(index);
        return key === releaseModal;
      }),
    [releaseModal, releases]
  );

  function markSelfUpdateRestarting() {
    setRestartPending(true);
    setRepoError("");
    setUpdateStatus((status) => ({
      ...status,
      status: "installing",
      progress: Math.max(typeof status.progress === "number" ? status.progress : 0, 95),
      error_message: "",
    }));
  }

  function scheduleRestartRefresh(delayMs = 2500) {
    if (restartRefreshTimer.current !== null) {
      window.clearTimeout(restartRefreshTimer.current);
    }
    restartRefreshTimer.current = window.setTimeout(() => {
      restartRefreshTimer.current = null;
      void loadUpdateData(false, true);
    }, delayMs);
  }

  const loadUpdateData = async (checkRemote = false, suppressRestartErrors = false) => {
    setChecking(checkRemote);
    if (checkRemote) setLoadingReleases(true);
    setRepoError("");
    const failures: string[] = [];
    let checkedStatus: UpdateStatus | null = null;
    let statusLoaded = false;
    let autoDownloadStarted = false;

    if (checkRemote) {
      try {
        const payload = await api<any>("/api/v1/update/check", { method: "POST" });
        const data = apiData<UpdateStatus>(payload, {});
        checkedStatus = data;
        setUpdateStatus(data);
        if (data.has_update && updateConfig.auto_update) {
          await api("/api/v1/update/download", { method: "POST" });
          autoDownloadStarted = true;
        }
      } catch (err) {
        failures.push(errorMessage(err));
      }
    }

    const [versionResult, statusResult, releasesResult, componentResult, configResult, componentConfigResult] = await Promise.allSettled([
      api<any>("/api/v1/version"),
      api<any>("/api/v1/update/status"),
      checkRemote ? api<any>("/api/v1/update/releases") : Promise.resolve(null),
      api<any>("/api/v1/component-updates"),
      api<any>("/api/v1/update/config"),
      Promise.all(["mosdns", "mihomo", "zashboard"].map((component) => api<any>(`/api/v1/component-updates/${component}/config`))),
    ]);

    if (versionResult.status === "fulfilled") {
      setVersionInfo(apiData<VersionInfo>(versionResult.value, {}));
    } else {
      failures.push(errorMessage(versionResult.reason));
    }

    if (statusResult.status === "fulfilled") {
      setUpdateStatus(apiData<UpdateStatus>(statusResult.value, {}));
      statusLoaded = true;
      if (restartPending || suppressRestartErrors) {
        setRestartPending(false);
        restartRefreshAttempts.current = 0;
      }
    } else {
      failures.push(errorMessage(statusResult.reason));
    }

    if (checkRemote) {
      if (releasesResult.status === "fulfilled") {
        setReleases(apiList<ReleaseItem>(releasesResult.value, ["data", "items", "releases"]));
      } else {
        failures.push(errorMessage(releasesResult.reason));
      }
    }

    if (componentResult.status === "fulfilled") {
      setComponentUpdates(apiList<ComponentUpdateState>(componentResult.value, ["data", "items", "components"]));
    } else {
      failures.push(errorMessage(componentResult.reason));
    }

    if (configResult.status === "fulfilled") {
      setUpdateConfig({ ...defaultUpdateConfig, ...apiData<UpdateConfigState>(configResult.value, defaultUpdateConfig) });
    } else {
      failures.push(errorMessage(configResult.reason));
    }

    if (componentConfigResult.status === "fulfilled") {
      const nextConfigs: Record<string, ComponentUpdateConfigState> = {};
      for (const payload of componentConfigResult.value) {
        const data = apiData<ComponentUpdateConfigState>(payload, {} as ComponentUpdateConfigState);
        if (data.component) nextConfigs[data.component] = data;
      }
      setComponentConfigs(nextConfigs);
    } else {
      failures.push(errorMessage(componentConfigResult.reason));
    }

    const restartFailures = suppressRestartErrors && failures.length > 0 && failures.every((message) => isTransientRestartError(message));
    if (restartFailures) {
      setRepoError("");
      if (!statusLoaded) {
        markSelfUpdateRestarting();
        restartRefreshAttempts.current += 1;
        if (restartRefreshAttempts.current <= 20) {
          scheduleRestartRefresh(2500);
        }
      }
    } else if (failures.length > 0) {
      setRepoError(failures[0]);
      if (checkRemote) showToast(`检查更新失败: ${failures[0]}`);
    } else if (checkRemote && autoDownloadStarted) {
      showToast("检测到新版本，已开始自动下载");
    } else if (checkRemote && checkedStatus?.has_update && updateConfig.notify) {
      showToast("检测到新版本可用");
    } else if (checkRemote) {
      showToast("已检查更新");
    }

    if (checkRemote) setLoadingReleases(false);
    setChecking(false);
  };

  useEffect(() => {
    return () => {
      if (restartRefreshTimer.current !== null) {
        window.clearTimeout(restartRefreshTimer.current);
      }
    };
  }, []);

  useEffect(() => {
    void loadUpdateData();
  }, []);

  useEffect(() => {
    if (!updateConfig.auto_check) return;
    const intervalSeconds = Math.max(60, updateConfig.check_interval || defaultUpdateConfig.check_interval);
    const timer = window.setInterval(() => {
      void loadUpdateData(true);
    }, intervalSeconds * 1000);
    return () => window.clearInterval(timer);
  }, [updateConfig.auto_check, updateConfig.auto_update, updateConfig.check_interval, updateConfig.notify]);

  const checkUpdates = () => {
    void loadUpdateData(true);
  };

  const saveUpdateConfig = async (patch: Partial<UpdateConfigState>) => {
    const next = { ...updateConfig, ...patch };
    setUpdateConfig(next);
    try {
      const payload = await api<any>("/api/v1/update/config", {
        method: "PUT",
        body: JSON.stringify(next),
      });
      setUpdateConfig({ ...defaultUpdateConfig, ...apiData<UpdateConfigState>(payload, next) });
      showToast("更新配置已保存");
    } catch (err) {
      setUpdateConfig(updateConfig);
      showToast(`保存更新配置失败: ${errorMessage(err)}`);
    }
  };

  const saveComponentConfig = async (component: string, patch: Partial<ComponentUpdateConfigState>) => {
    const current = componentConfigs[component] || { component, auto_check: true, check_interval: 43200, auto_update: false };
    const next = { ...current, ...patch, component };
    setComponentConfigs((items) => ({ ...items, [component]: next }));
    try {
      const payload = await api<any>(`/api/v1/component-updates/${component}/config`, {
        method: "PUT",
        body: JSON.stringify(next),
      });
      setComponentConfigs((items) => ({ ...items, [component]: apiData<ComponentUpdateConfigState>(payload, next) }));
      showToast(`${component} 更新配置已保存`);
    } catch (err) {
      setComponentConfigs((items) => ({ ...items, [component]: current }));
      showToast(`${component} 更新配置保存失败: ${errorMessage(err)}`);
    }
  };

  const downloadUpdate = async () => {
    try {
      await api("/api/v1/update/download", { method: "POST" });
      showToast("更新包已开始下载");
      void loadUpdateData();
    } catch (err) {
      showToast(`下载更新失败: ${errorMessage(err)}`);
    }
  };

  const installUpdate = async () => {
    if (!window.confirm("安装更新会重启 msf 服务，当前 WebUI 会短暂断开。是否继续？")) return;
    try {
      const payload = await api<any>("/api/v1/update/install", { method: "POST" });
      if (payload.success === false) {
        showToast(`安装更新失败: ${payload.error || "未知错误"}`);
        void loadUpdateData();
        return;
      }
      showToast(payload.message || "更新安装已开始，服务将自动重启");
      restartRefreshAttempts.current = 0;
      markSelfUpdateRestarting();
      scheduleRestartRefresh();
    } catch (err) {
      if (isTransientRestartError(err)) {
        showToast("更新安装已开始，服务正在重启");
        restartRefreshAttempts.current = 0;
        markSelfUpdateRestarting();
        scheduleRestartRefresh();
        return;
      }
      showToast(`安装更新失败: ${errorMessage(err)}`);
    }
  };

  const componentItem = (component: string) =>
    componentUpdates.find((item) => item.component === component);

  const reloadComponents = async () => {
    const payload = await api<any>("/api/v1/component-updates");
    setComponentUpdates(apiList<ComponentUpdateState>(payload, ["data", "items", "components"]));
  };

  const checkComponent = async (component: string) => {
    setComponentBusy(component);
    try {
      const payload = await api<any>(`/api/v1/component-updates/${component}/check`, { method: "POST" });
      if (payload.success === false) {
        showToast(`${component} 检查更新失败: ${payload.error || "未知错误"}`);
        await reloadComponents();
        return;
      }
      const data = apiData<ComponentUpdateState>(payload, {});
      const config = componentConfigs[component];
      if (data.has_update && config?.auto_update) {
        const updatePayload = await api<any>(`/api/v1/component-updates/${component}/update`, { method: "POST" });
        if (updatePayload.success === false) {
          showToast(`${component} 自动更新失败: ${updatePayload.error || "未知错误"}`);
          await reloadComponents();
          return;
        }
        showToast(`${component} 检测到更新，已自动执行更新任务`);
      } else {
        showToast(`${component} 已检查更新`);
      }
      await reloadComponents();
    } catch (err) {
      showToast(`${component} 检查更新失败: ${errorMessage(err)}`);
    } finally {
      setComponentBusy("");
    }
  };

  const updateComponent = async (component: string) => {
    setComponentBusy(component);
    try {
      const payload = await api<any>(`/api/v1/component-updates/${component}/update`, { method: "POST" });
      if (payload.success === false) {
        showToast(`${component} 更新失败: ${payload.error || "未知错误"}`);
        await reloadComponents();
        return;
      }
      showToast(`${component} 更新任务已执行`);
      await reloadComponents();
    } catch (err) {
      showToast(`${component} 更新失败: ${errorMessage(err)}`);
    } finally {
      setComponentBusy("");
    }
  };

  const uploadComponent = async (component: string, file: File) => {
    setComponentBusy(component);
    try {
      const body = new FormData();
      body.append("file", file);
      const payload = await api<any>(`/api/v1/component-updates/${component}/upload`, { method: "POST", body });
      if (payload.success === false) {
        showToast(`${component} 本地上传失败: ${payload.error || "未知错误"}`);
        return;
      }
      const restarted = payload.data?.restarted ? "，已重启服务" : "";
      showToast(`${component} 已通过本地文件安装${restarted}`);
      await reloadComponents();
    } catch (err) {
      showToast(`${component} 本地上传失败: ${errorMessage(err)}`);
    } finally {
      setComponentBusy("");
    }
  };

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
        <Card title="版本信息" Icon={FileText}>
          <div className="space-y-1">
            <UpdateInfoTile label="当前版本" value={currentVersion} />
            <UpdateInfoTile label="最新版本" value={latestVersion} badge={hasUpdate ? "可更新" : undefined} />
            <UpdateInfoTile label="最后检查" value={formatDateTime(displayUpdateStatus.last_check_time)} />
            <UpdateInfoTile label="Go 版本" value={versionInfo.go_version || "-"} />
            <UpdateInfoTile label="构建平台" value={versionInfo.platform || "-"} />
            <UpdateInfoTile label="构建时间" value={versionInfo.build_time || "-"} />
            <UpdateInfoTile label="源仓库" value={RELEASE_REPO} />
          </div>
        </Card>

        <Card title="更新状态" Icon={RefreshCw}>
          <div className="flex items-center justify-between gap-4">
            <span className="text-xs text-muted-foreground">当前状态</span>
            <span className="inline-flex h-6 items-center gap-1.5 rounded-full border border-border bg-background px-2.5 text-xs font-semibold text-foreground">
              <span className={cn("h-1.5 w-1.5 rounded-full", displayUpdateStatus.status === "failed" ? "bg-destructive" : checking || restartPending ? "bg-primary" : "bg-muted-foreground")} />
              {statusLabel(displayUpdateStatus.status)}
            </span>
          </div>
          {typeof displayUpdateStatus.progress === "number" && displayUpdateStatus.progress > 0 ? (
            <div className="mt-4">
              <div className="mb-1 flex justify-between text-xs text-muted-foreground">
                <span>进度</span>
                <span>{displayUpdateStatus.progress}%</span>
              </div>
              <div className="h-1.5 rounded-full bg-muted">
                <div className="h-full rounded-full bg-primary" style={{ width: `${Math.min(100, Math.max(0, displayUpdateStatus.progress))}%` }} />
              </div>
            </div>
          ) : null}
          {restartPending ? <div className="mt-3 text-xs text-muted-foreground">服务正在重启，页面短暂断开属于正常现象。</div> : null}
          {!restartPending && displayUpdateStatus.error_message ? <div className="mt-3 text-xs text-destructive">{displayUpdateStatus.error_message}</div> : null}
        </Card>

        <Card title="操作" Icon={Download}>
          <div className="space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <OutlineButton onClick={checkUpdates} disabled={checking} className="h-8 px-3 text-xs">
                <RefreshCw className={cn("h-3.5 w-3.5", checking && "animate-spin")} />
                检查更新
              </OutlineButton>
              <PrimaryButton onClick={downloadUpdate} disabled={checking || !hasUpdate || !latestVersion} className="h-8 px-3 text-xs">
                <Download className="h-3.5 w-3.5" />
                下载更新
              </PrimaryButton>
              <PrimaryButton onClick={installUpdate} disabled={checking || installingUpdate || !canInstallUpdate} className="h-8 px-3 text-xs">
                <RefreshCw className={cn("h-3.5 w-3.5", installingUpdate && "animate-spin")} />
                安装并重启
              </PrimaryButton>
            </div>
            {displayUpdateStatus.effective_download_url && displayUpdateStatus.effective_download_url !== displayUpdateStatus.download_url ? (
              <div className="rounded-lg border border-border/50 bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
                下载走加速: <span className="break-all font-mono text-foreground/80">{displayUpdateStatus.effective_download_url}</span>
              </div>
            ) : null}
            <div className="border-t border-border/40 pt-3 text-xs text-muted-foreground">
              最后检查: {formatDateTime(displayUpdateStatus.last_check_time)}
            </div>
          </div>
        </Card>
      </div>

      <Card title="更新日志" Icon={FileText}>
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>来源: {RELEASE_REPO}</span>
          <a href={`${RELEASE_REPO_URL}/releases`} target="_blank" rel="noreferrer" className="text-primary hover:underline">
            打开 GitHub Releases
          </a>
        </div>
        {loadingReleases ? (
          <div className="rounded-lg border border-border/50 bg-muted/20 p-3 text-xs text-muted-foreground">正在加载更新日志...</div>
        ) : latestRelease ? (
          <pre className="max-h-[460px] overflow-auto whitespace-pre-wrap break-words rounded-lg border border-border/50 bg-muted/10 p-3 text-xs leading-relaxed text-foreground/90">
            {latestRelease.body || "该版本未填写更新日志。"}
          </pre>
        ) : (
          <div className="rounded-lg border border-border/50 bg-muted/20 p-3 text-xs text-muted-foreground">
            {repoError || "当前仓库暂无发布记录。"}
          </div>
        )}
      </Card>

      <Card title="更新配置" Icon={Settings}>
        <div className="space-y-3">
          <UpdateConfigRow title="自动检查更新" description="定期检查是否有新版本可用">
            <Toggle checked={updateConfig.auto_check} onChange={(checked) => void saveUpdateConfig({ auto_check: checked })} />
          </UpdateConfigRow>
          <UpdateConfigRow title="检查间隔" description="设置检查更新的时间间隔">
            <SmallSelect value={String(updateConfig.check_interval)} onChange={(event) => void saveUpdateConfig({ check_interval: Number(event.target.value) })}>
              {updateIntervals.map((option) => (
                <option key={option.value} value={option.value}>{option.label}</option>
              ))}
            </SmallSelect>
          </UpdateConfigRow>
          <UpdateConfigRow title="自动下载更新" description="检测到新版本后自动下载">
            <Toggle checked={updateConfig.auto_update} onChange={(checked) => void saveUpdateConfig({ auto_update: checked })} />
          </UpdateConfigRow>
          <UpdateConfigRow title="更新通知" description="有新版本时通知用户">
            <Toggle checked={updateConfig.notify} onChange={(checked) => void saveUpdateConfig({ notify: checked })} />
          </UpdateConfigRow>
          <UpdateConfigRow title="MosDNS 升级方式" description="默认推荐全量升级。">
            <SmallSelect
              className="w-36"
              value={updateConfig.mosdns_upgrade_mode}
              onChange={(event) => void saveUpdateConfig({ mosdns_upgrade_mode: event.target.value as UpdateConfigState["mosdns_upgrade_mode"] })}
            >
              <option value="full">全量升级（推荐）</option>
              <option value="incremental">增量升级</option>
              <option value="reset">重置升级（谨慎）</option>
            </SmallSelect>
          </UpdateConfigRow>
          <div className="rounded-lg border border-border/50 bg-muted/20 p-3 text-xs leading-relaxed text-muted-foreground">
            全量升级：保留系统里已保存的关键设置，其余按新模板覆盖；升级完成后会在 MosDNS 启动时自动回写这些设置。增量升级：尽量保留当前配置与规则，按升级规则补齐新文件。重置升级：完全放弃现有改动，全部使用新模板；仅在需要彻底重置时使用。
          </div>
          <UpdateConfigRow title="Mihomo 升级方式" description="不升级将完全保留当前配置；全量升级仅保留机场与 VPS 节点相关改动。">
            <SmallSelect
              className="w-36"
              value={updateConfig.mihomo_upgrade_mode}
              onChange={(event) => void saveUpdateConfig({ mihomo_upgrade_mode: event.target.value as UpdateConfigState["mihomo_upgrade_mode"] })}
            >
              <option value="skip">不升级（保留自定义）</option>
              <option value="full">全量升级（仅保留节点）</option>
            </SmallSelect>
          </UpdateConfigRow>
          <div className="rounded-lg border border-border/50 bg-muted/20 p-3 text-xs leading-relaxed text-muted-foreground">
            已自定义配置时建议使用“不升级”。未自定义配置时建议使用“全量升级”。
          </div>
        </div>
      </Card>

      <Card title="组件更新" Icon={Download}>
        <p className="mb-4 text-sm text-muted-foreground">组件更新管理</p>
        <div className="grid gap-4 lg:grid-cols-2">
          <ComponentUpdateCard name="MosDNS" component="mosdns" item={componentItem("mosdns")} config={componentConfigs.mosdns} busy={componentBusy} onUpload={uploadComponent} onCheck={checkComponent} onUpdate={updateComponent} onConfigChange={(item, patch) => void saveComponentConfig(item, patch)} />
          <ComponentUpdateCard name="Mihomo" component="mihomo" item={componentItem("mihomo")} config={componentConfigs.mihomo} busy={componentBusy} onUpload={uploadComponent} onCheck={checkComponent} onUpdate={updateComponent} onConfigChange={(item, patch) => void saveComponentConfig(item, patch)} />
          <ComponentUpdateCard name="Zashboard" component="zashboard" item={componentItem("zashboard")} config={componentConfigs.zashboard} busy={componentBusy} onUpload={uploadComponent} onCheck={checkComponent} onUpdate={updateComponent} onConfigChange={(item, patch) => void saveComponentConfig(item, patch)} />
        </div>
      </Card>

      <Card title="版本历史" Icon={FileText} className="overflow-hidden">
        <p className="mb-3 text-sm text-muted-foreground">最近发布的主要更新</p>
        <div className="space-y-2">
          {releases.length === 0 ? (
            <div className="rounded-lg border border-border/50 bg-muted/20 p-3 text-xs text-muted-foreground">
              {loadingReleases ? "正在加载历史版本..." : repoError || "当前仓库暂无历史版本。"}
            </div>
          ) : (
            releases.map((release, index) => {
              const key = release.tag_name || release.html_url || release.name || String(index);
              const version = releaseTitle(release);
              return (
                <div
                  key={key}
                  className="group relative overflow-hidden rounded-xl border border-border/50 bg-gradient-to-br from-background to-muted/20 p-3 pl-4 transition-all duration-300 hover:border-border hover:from-muted/30 hover:to-muted/40 hover:shadow-md"
                >
                  <div className="absolute bottom-0 left-0 top-0 w-1 bg-gradient-to-b from-muted to-muted/50 group-first:from-green-500 group-first:to-emerald-500" />
                  <div className="flex items-center gap-3">
                    <div className="min-w-[72px] text-center">
                      <div className="text-xs font-semibold text-foreground">{formatRelativeDate(release.published_at)}</div>
                      <div className="mt-0.5 text-[10px] text-muted-foreground">{formatDate(release.published_at)}</div>
                    </div>
                    <div className="h-10 w-px bg-border/50" />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-sm font-bold text-foreground">{version}</span>
                        {index === 0 ? <span className="rounded-full bg-green-500 px-1.5 py-0.5 text-[10px] text-white">最新版本</span> : null}
                      </div>
                      {release.name && release.name !== version ? <div className="mt-0.5 truncate text-xs text-muted-foreground">{release.name}</div> : null}
                    </div>
                    <button
                      onClick={() => setReleaseModal(key)}
                      className="shrink-0 rounded-lg border border-border bg-background px-3 py-1.5 text-xs transition-colors hover:border-primary/30 hover:bg-primary/10 hover:text-primary"
                    >
                      {release.body ? "查看变更" : "查看详情"}
                    </button>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </Card>

      {selectedRelease ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm animate-fade-in">
          <div className="flex max-h-[90vh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl border border-border bg-background shadow-2xl animate-scale-in">
            <div className="flex items-center justify-between border-b border-border/50 bg-gradient-to-r from-purple-500/10 to-blue-500/10 p-4">
              <div>
                <h3 className="font-bold text-foreground">版本 {releaseTitle(selectedRelease)} 变更日志</h3>
                <p className="mt-1 text-xs text-muted-foreground">
                  发布于 {formatDateTime(selectedRelease.published_at)} · {RELEASE_REPO}
                </p>
              </div>
              <button onClick={() => setReleaseModal(null)} className="rounded-lg p-2 text-muted-foreground hover:bg-muted hover:text-foreground">
                <X className="h-5 w-5" />
              </button>
            </div>
            <div className="overflow-auto p-5 text-sm text-muted-foreground">
              <pre className="whitespace-pre-wrap break-words text-sm leading-6 text-foreground/90">
                {selectedRelease.body || "该版本未填写更新日志。"}
              </pre>
              {selectedRelease.assets?.length ? (
                <div className="mt-4 border-t border-border/50 pt-4">
                  <div className="mb-2 text-xs font-semibold text-foreground">发布资产</div>
                  <div className="space-y-1">
                    {selectedRelease.assets.map((asset) => (
                      <a
                        key={asset.browser_download_url || asset.name}
                        href={asset.browser_download_url}
                        target="_blank"
                        rel="noreferrer"
                        className="block truncate rounded-lg border border-border/50 px-3 py-2 text-xs text-primary hover:bg-muted/50"
                      >
                        {asset.name || asset.browser_download_url}
                      </a>
                    ))}
                  </div>
                </div>
              ) : null}
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function ResetTab({ showToast }: { showToast: (message: string) => void }) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [deleteBinaries, setDeleteBinaries] = useState(false);
  const [password, setPassword] = useState("");
  const [resetting, setResetting] = useState(false);

  const resetSystem = async () => {
    setResetting(true);
    try {
      await api("/api/v1/setup/reset", {
        method: "POST",
        body: JSON.stringify({ delete_binaries: deleteBinaries, delete_components: deleteBinaries, current_password: password }),
      });
      setConfirmOpen(false);
      setPassword("");
      showToast(deleteBinaries ? "系统已重置，组件二进制已删除" : "系统已重置");
    } catch (error) {
      showToast(errorMessage(error));
    } finally {
      setResetting(false);
    }
  };

  return (
    <>
      <Card title="重置所有设置" Icon={ShieldAlert} className="border-destructive/50">
        <p className="mb-4 text-sm text-muted-foreground">
          此操作将清除所有系统配置和用户数据,并重新进入初始化向导。
          <span className="font-semibold text-destructive">此操作不可撤销!</span>
        </p>
        <PrimaryButton onClick={() => setConfirmOpen(true)} className="bg-destructive hover:bg-destructive/90">
          <ShieldAlert className="h-4 w-4" />
          重置系统
        </PrimaryButton>
      </Card>

      {confirmOpen ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 animate-fade-in">
          <div className="w-full max-w-md rounded-[12px] border border-border bg-card text-card-foreground shadow-apple-xl animate-fade-in">
            <div className="border-b border-border/50 p-6 pb-3">
              <div className="flex items-center gap-2">
                <TriangleAlert className="h-3.5 w-3.5 text-destructive" />
                <h3 className="text-xs font-semibold tracking-tight text-destructive">⚠️ 重置系统确认</h3>
              </div>
            </div>
            <div className="space-y-3 p-6 pt-4">
              <div className="space-y-2">
                <p className="text-xs font-semibold text-foreground">此操作将:</p>
                <ul className="list-inside list-disc space-y-1 text-xs text-muted-foreground">
                  <li>清除所有系统配置</li>
                  <li>清除所有用户数据</li>
                  <li>重新进入初始化向导</li>
                  <li className="font-medium text-destructive">此操作不可撤销!</li>
                </ul>
              </div>
              <label className="flex items-start gap-3 rounded-lg border border-destructive/20 bg-destructive/5 px-3 py-2 text-xs transition-colors hover:border-destructive/40">
                <input
                  type="checkbox"
                  checked={deleteBinaries}
                  onChange={(event) => setDeleteBinaries(event.target.checked)}
                  className="peer sr-only"
                />
                <span className="mt-0.5 flex h-4 w-4 items-center justify-center rounded border border-destructive/50 bg-background shadow-sm transition peer-checked:border-destructive peer-checked:bg-destructive">
                  <Check className={cn("h-3 w-3 text-white transition-opacity", deleteBinaries ? "opacity-100" : "opacity-0")} />
                </span>
                <span className="space-y-0.5">
                  <span className="block font-medium text-foreground">删除组件二进制（MosDNS / Mihomo / Sing-box）</span>
                  <span className="block text-[11px] text-muted-foreground">开启后将删除已下载的组件，重置后会重新下载。</span>
                </span>
              </label>
              <Field label="请输入管理员密码以确认:">
                <PasswordInput value={password} onChange={setPassword} placeholder="请输入当前管理员密码" />
              </Field>
              <div className="flex gap-2 pt-2">
                <OutlineButton onClick={() => setConfirmOpen(false)} className="h-9 flex-1">
                  取消
                </OutlineButton>
                <PrimaryButton
                  disabled={!password || resetting}
                  onClick={resetSystem}
                  className="h-9 flex-1 bg-destructive hover:bg-destructive/90"
                >
                  确认重置
                </PrimaryButton>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}

export function SettingsClient({ initialTab }: { initialTab: TabId }) {
  const { toasts, showToast } = useToaster();
  const [activeTab, setActiveTab] = useState<TabId>(initialTab);

  const changeTab = (tab: TabId) => {
    setActiveTab(tab);
    const url = tab === "profile" ? "/settings?tab=profile" : `/settings?tab=${tab}`;
    window.history.replaceState(null, "", url);
  };

  return (
    <AppShell>
      <div className="pb-6 animate-fade-in">
        <div>
          <h1 className="text-2xl font-bold leading-7 text-foreground">系统设置</h1>
          <p className="text-sm leading-4 text-muted-foreground">个人设置与系统配置</p>
        </div>

        <div className="mt-4 inline-flex w-full items-center gap-1 overflow-x-auto rounded-lg bg-muted p-1 scrollbar-hide" role="tablist">
          {tabs.map(({ id, label, Icon }) => (
            <button
              key={id}
              onClick={() => changeTab(id)}
              className={cn(
                "inline-flex h-11 min-w-[67px] flex-1 items-center justify-center gap-1.5 whitespace-nowrap rounded-md px-2 py-1.5 text-xs font-medium transition-all duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 md:h-7 md:min-w-[92px] md:flex-none md:px-3",
                activeTab === id
                  ? "bg-background text-foreground shadow-sm"
                  : "text-muted-foreground hover:bg-background/50 hover:text-foreground"
              )}
              role="tab"
              aria-selected={activeTab === id}
            >
              <Icon className="hidden h-3.5 w-3.5 md:block" />
              {label}
            </button>
          ))}
        </div>

        <div role="tabpanel" className="mt-2 animate-slide-up">
          {activeTab === "profile" && <ProfileTab showToast={showToast} />}
          {activeTab === "system" && <SystemTab showToast={showToast} />}
          {activeTab === "appearance" && <AppearanceTab showToast={showToast} />}
          {activeTab === "update" && <UpdateTab showToast={showToast} />}
          {activeTab === "reset" && <ResetTab showToast={showToast} />}
        </div>
      </div>
      <ToastStack toasts={toasts} />
    </AppShell>
  );
}
