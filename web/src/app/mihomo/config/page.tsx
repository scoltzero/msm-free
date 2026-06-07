"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Play,
  Square,
  RotateCw,
  Maximize2,
  Copy,
  X,
  Upload,
  Download,
  Save,
  CheckCircle2,
  FileCode,
  Cpu,
  FileText,
  RefreshCw,
  RotateCcw,
  TriangleAlert,
  Trash2,
} from "lucide-react";
import { AppShell } from "@/components/AppShell";
import { useToaster, ToastStack } from "@/components/Toaster";
import { YamlEditor } from "@/components/mihomo/YamlEditor";
import { api, apiList, formatBytes, formatPercent } from "@/lib/api";
import { cn } from "@/lib/utils";

const DEFAULT_PATH = "configs/mihomo/config.yaml";
const USER_CONFIG_DIR = "configs/mihomo/user_configs";

interface ConfigFile {
  name?: string;
  path?: string;
  size?: number;
  modified?: string;
  active?: boolean;
}

interface ServiceStatus {
  status?: string;
  state?: string;
  running?: boolean;
  installed?: boolean;
  version?: string;
  pid?: number | string;
  uptime?: string;
  uptime_text?: string;
  cpu?: number | string;
  cpu_percent?: number | string;
  memory?: number | string;
  memory_bytes?: number | string;
  memory_text?: string;
  config_path?: string;
  path?: string;
  log_path?: string;
}

interface ConfigModeInfo {
  mode?: "generated" | "custom";
  backup_path?: string;
  backup_exists?: boolean;
  protected_fields?: string[];
  protected_warning?: string;
}

interface ConfigValidation {
  valid?: boolean;
  error?: string;
  warnings?: string[];
}

function configPathFor(file: ConfigFile | string) {
  if (typeof file === "string") {
    return file.startsWith("configs/mihomo/") ? file : `${USER_CONFIG_DIR}/${file}`;
  }
  if (file.path) return file.path;
  return configPathFor(file.name || "");
}

function fileName(path: string) {
  return path.split("/").filter(Boolean).pop() || path;
}

function isUserConfigPath(path: string) {
  return path.startsWith(`${USER_CONFIG_DIR}/`);
}

function isReservedConfigName(name: string) {
  return ["config.yaml", "phone_config.yaml", "msf_generated.backup.yaml", "config.yaml.backup"].includes(name.trim().toLowerCase());
}

function normalizeClientConfigName(name: string) {
  let next = name.trim();
  if (!next) return "";
  if (next.includes("/") || next.includes("\\") || next.includes("..")) return "";
  if (!/\.(ya?ml)$/i.test(next)) next += ".yaml";
  if (isReservedConfigName(next)) return "";
  return next;
}

function isRunning(status: ServiceStatus | null) {
  if (!status) return false;
  if (typeof status.running === "boolean") return status.running;
  const state = String(status.status || status.state || "").toLowerCase();
  return state === "running" || state === "active";
}

function serviceStatusText(status: ServiceStatus | null) {
  if (!status) return "未知";
  if (isRunning(status)) return "运行中";
  if (status.installed === false) return "未安装";
  return "已停止";
}

function memoryText(status: ServiceStatus | null) {
  if (!status) return "-";
  if (status.memory_text) return status.memory_text;
  return formatBytes(status.memory_bytes ?? status.memory);
}

function cpuText(status: ServiceStatus | null) {
  if (!status) return "-";
  return formatPercent(status.cpu_percent ?? status.cpu);
}

export default function MihomoConfigPage() {
  const { toasts, showToast } = useToaster();
  const [content, setContent] = useState("");
  const [path, setPath] = useState(DEFAULT_PATH);
  const [files, setFiles] = useState<ConfigFile[]>([]);
  const [status, setStatus] = useState<ServiceStatus | null>(null);
  const [version, setVersion] = useState("");
  const [dirty, setDirty] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [acting, setActing] = useState(false);
  const [fullscreen, setFullscreen] = useState(false);
  const [modeInfo, setModeInfo] = useState<ConfigModeInfo>({});
  const [warnings, setWarnings] = useState<string[]>([]);
  const [suggestedName, setSuggestedName] = useState("user_config_0.yaml");
  const fileRef = useRef<HTMLInputElement>(null);

  const running = isRunning(status);

  const loadStatus = useCallback(async () => {
    try {
      const [servicePayload, versionPayload] = await Promise.all([
        api<any>("/api/v1/services/mihomo"),
        api<any>("/api/v1/mihomo/version"),
      ]);
      setStatus((servicePayload.data || servicePayload.service || servicePayload) as ServiceStatus);
      setVersion(String(versionPayload.version || versionPayload.data?.version || ""));
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err));
    }
  }, [showToast]);

  const loadFiles = useCallback(async () => {
    try {
      const payload = await api<any>("/api/v1/mihomo/user-configs");
      const data = payload.data || payload;
      setSuggestedName(String(data.suggested_name || payload.suggested_name || "user_config_0.yaml"));
      const fileRows = apiList<ConfigFile>(data, ["items", "files"]);
      if (fileRows.length > 0) {
        setFiles(fileRows);
        return;
      }
      const names = apiList<string>(data, ["configs"]);
      setFiles(names.map((name) => ({ name, path: configPathFor(name) })));
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err));
    }
  }, [showToast]);

  const loadMode = useCallback(async () => {
    try {
      const payload = await api<any>("/api/v1/mihomo/config/mode");
      setModeInfo((payload.data || payload) as ConfigModeInfo);
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err));
    }
  }, [showToast]);

  const loadConfig = useCallback(async (nextPath = path) => {
    setLoading(true);
    try {
      const payload = await api<any>(`/api/v1/mihomo/config?path=${encodeURIComponent(nextPath)}`);
      const raw = String(payload.content ?? "");
      setPath(String(payload.path || nextPath || DEFAULT_PATH));
      setContent(raw);
      setDirty(false);
      setWarnings([]);
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [path, showToast]);

  const reloadAll = useCallback(async () => {
    await Promise.all([loadConfig(path), loadFiles(), loadStatus(), loadMode()]);
  }, [loadConfig, loadFiles, loadMode, loadStatus, path]);

  useEffect(() => {
    void reloadAll();
  }, [reloadAll]);

  const promptConfigName = useCallback(() => {
    const currentName = isUserConfigPath(path) ? fileName(path) : "";
    const initial = currentName || suggestedName || "user_config_0.yaml";
    const raw = window.prompt("请输入用户配置名称", initial);
    if (raw === null) return "";
    const name = normalizeClientConfigName(raw);
    if (!name) {
      showToast("配置名称无效，请使用普通文件名并以 .yaml 或 .yml 结尾");
      return "";
    }
    const exists = files.some((file) => fileName(configPathFor(file)).toLowerCase() === name.toLowerCase());
    if (exists && !window.confirm(`用户配置 ${name} 已存在，是否覆盖？`)) {
      return "";
    }
    return name;
  }, [files, path, showToast, suggestedName]);

  const applyUserConfigPath = useCallback(async (targetPath: string) => {
    const payload = await api<any>("/api/v1/mihomo/user-configs/apply", {
      method: "POST",
      body: JSON.stringify({ path: targetPath, restart: true }),
    });
    const validation = payload.data?.validation as ConfigValidation | undefined;
    setWarnings(validation?.warnings || []);
    await Promise.all([loadFiles(), loadStatus(), loadMode()]);
    showToast((validation?.warnings || []).length > 0 ? "配置已应用并重启，但存在关键字段告警" : "配置已应用并重启 Mihomo");
  }, [loadFiles, loadMode, loadStatus, showToast]);

  const save = useCallback(async (applyAfterSave = false) => {
    const name = promptConfigName();
    if (!name) return;
    setSaving(true);
    try {
      const payload = await api<any>("/api/v1/mihomo/user-configs", {
        method: "PUT",
        body: JSON.stringify({ name, content, overwrite: true }),
      });
      if (payload.success === false) {
        showToast(payload.error || "配置保存失败");
        return;
      }
      const validation = payload.data?.validation as ConfigValidation | undefined;
      const savedPath = String(payload.data?.config?.path || `${USER_CONFIG_DIR}/${name}`);
      const nextWarnings = validation?.warnings || [];
      setWarnings(nextWarnings);
      setDirty(false);
      setPath(savedPath);
      showToast(nextWarnings.length > 0 ? "用户配置已保存，但存在关键字段告警" : "用户配置已保存");
      await loadFiles();
      if (applyAfterSave) {
        await applyUserConfigPath(savedPath);
      }
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  }, [applyUserConfigPath, content, loadFiles, promptConfigName, showToast]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
        event.preventDefault();
        void save();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [save]);

  const validate = useCallback(async () => {
    try {
      const payload = await api<any>("/api/v1/config/validate", {
        method: "POST",
        body: JSON.stringify({ path, content }),
      });
      if (payload.valid === false) {
        showToast(payload.error || "配置验证失败");
        return;
      }
      const mihomoPayload = await api<any>("/api/v1/mihomo/validate", {
        method: "POST",
        body: JSON.stringify({ content }),
      });
      const validation = (mihomoPayload.data || mihomoPayload) as ConfigValidation;
      setWarnings(validation.warnings || []);
      showToast((validation.warnings || []).length > 0 ? "语法通过，但存在关键字段告警" : "配置验证通过");
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err));
    }
  }, [content, path, showToast]);

  const copy = useCallback(async () => {
    await navigator.clipboard.writeText(content);
    showToast("配置已复制");
  }, [content, showToast]);

  const download = useCallback(() => {
    const blob = new Blob([content], { type: "text/yaml" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = fileName(path);
    a.click();
    URL.revokeObjectURL(url);
    showToast("配置已下载");
  }, [content, path, showToast]);

  const importCustomConfig = useCallback(async (file: File | null) => {
    if (!file) return;
    if (dirty && !window.confirm("当前修改未保存，确定载入上传文件？")) return;
    setSaving(true);
    try {
      const text = await file.text();
      const name = normalizeClientConfigName(file.name) || suggestedName || "user_config_0.yaml";
      setContent(text);
      setPath(`${USER_CONFIG_DIR}/${name}`);
      setDirty(true);
      setWarnings([]);
      showToast("已载入上传配置，保存时可命名为用户配置");
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
      if (fileRef.current) fileRef.current.value = "";
    }
  }, [dirty, showToast, suggestedName]);

  const newTemplate = useCallback(async () => {
    if (dirty && !window.confirm("当前修改未保存，确定用自定义模板替换编辑器内容？")) return;
    try {
      const payload = await api<any>("/api/v1/mihomo/config/custom-template");
      const raw = String(payload.data?.content || payload.content || "");
      const name = normalizeClientConfigName(String(payload.data?.suggested_name || suggestedName)) || suggestedName || "user_config_0.yaml";
      setPath(`${USER_CONFIG_DIR}/${name}`);
      setContent(raw);
      setDirty(true);
      setWarnings([]);
      showToast("已载入自定义配置模板，保存后会进入用户配置列表");
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err));
    }
  }, [dirty, showToast, suggestedName]);

  const restoreDefault = useCallback(async () => {
    if (!window.confirm("确定恢复 MSF 预设 Mihomo 配置？当前自定义 config.yaml 会进入历史备份。")) return;
    setActing(true);
    try {
      await api("/api/v1/mihomo/config/restore-default", { method: "POST" });
      showToast("已恢复预设配置并重启 Mihomo");
      await reloadAll();
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err));
    } finally {
      setActing(false);
    }
  }, [reloadAll, showToast]);

  const runServiceAction = useCallback(async (action: "start" | "stop" | "restart") => {
    setActing(true);
    try {
      const payload = await api<any>(`/api/v1/services/mihomo/${action}?wait=1&timeout_ms=3000`, { method: "POST" });
      if (payload.success === false) {
        showToast(payload.error || "服务操作失败");
      } else {
        showToast(action === "restart" ? "服务已重启" : action === "start" ? "服务已启动" : "服务已停止");
      }
      await loadStatus();
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err));
    } finally {
      setActing(false);
    }
  }, [loadStatus, showToast]);

  const switchFile = useCallback((file: ConfigFile) => {
    if (dirty && !window.confirm("当前修改未保存，确定切换文件？")) return;
    void loadConfig(configPathFor(file));
  }, [dirty, loadConfig]);

  const applyUserConfig = useCallback(async (file: ConfigFile) => {
    const targetPath = configPathFor(file);
    if (dirty && !window.confirm("当前编辑器有未保存修改，确定先应用列表中的配置？")) return;
    if (!window.confirm(`确定应用 ${file.name || fileName(targetPath)} 并重启 Mihomo？`)) return;
    setActing(true);
    try {
      await applyUserConfigPath(targetPath);
    } catch (err) {
      showToast(err instanceof Error ? err.message : "应用配置失败");
    } finally {
      setActing(false);
    }
  }, [applyUserConfigPath, dirty, showToast]);

  const deleteUserConfig = useCallback(async (file: ConfigFile) => {
    const targetPath = configPathFor(file);
    const name = file.name || fileName(targetPath);
    if (!window.confirm(`确定删除用户配置 ${name}？当前已经应用到运行配置的内容不会被删除。`)) return;
    setActing(true);
    try {
      await api(`/api/v1/mihomo/user-configs/${encodeURIComponent(name)}`, { method: "DELETE" });
      if (targetPath === path) {
        await loadConfig(DEFAULT_PATH);
      }
      await loadFiles();
      showToast("用户配置已删除");
    } catch (err) {
      showToast(err instanceof Error ? err.message : "删除配置失败");
    } finally {
      setActing(false);
    }
  }, [loadConfig, loadFiles, path, showToast]);

  const serviceInfo = useMemo(() => [
    ["版本", version || status?.version || "-"],
    ["CPU / 内存", `${cpuText(status)} / ${memoryText(status)}`],
    ["运行时间", status?.uptime_text || status?.uptime || "-"],
    ["PID", String(status?.pid || "-")],
  ], [status, version]);

  return (
    <AppShell>
      <div className="space-y-4 animate-fade-in">
        <ToastStack toasts={toasts} />

        <div className="rounded-xl border border-border/60 bg-card p-4 shadow-sm">
          <div className="flex flex-wrap items-center gap-3">
            <div className="rounded-xl border border-primary/20 bg-primary/10 p-2">
              <FileCode className="h-5 w-5 text-primary" />
            </div>
            <div className="flex items-center gap-2">
              <h2 className="font-semibold text-foreground">服务控制</h2>
              <span
                className={cn(
                  "inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium",
                  running ? "bg-green-500/10 text-green-600 dark:text-green-400" : "bg-muted text-muted-foreground"
                )}
              >
                <span className={cn("h-1.5 w-1.5 rounded-full", running ? "bg-green-500 animate-pulse" : "bg-muted-foreground")} />
                {serviceStatusText(status)}
              </span>
              <span
                className={cn(
                  "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
                  modeInfo.mode === "custom" ? "bg-amber-500/10 text-amber-600 dark:text-amber-400" : "bg-blue-500/10 text-blue-600 dark:text-blue-400"
                )}
              >
                {modeInfo.mode === "custom" ? "自定义配置" : "系统预设"}
              </span>
            </div>
            <div className="ml-auto flex items-center gap-1.5">
              <button
                onClick={() => void runServiceAction("restart")}
                disabled={acting}
                className="flex items-center gap-1.5 rounded-lg bg-primary/10 px-3 py-2 text-sm font-medium text-primary transition-colors hover:bg-primary/20 disabled:opacity-50"
              >
                <RotateCw className={cn("h-4 w-4", acting && "animate-spin")} />
                重启
              </button>
              <button
                onClick={() => void runServiceAction(running ? "stop" : "start")}
                disabled={acting}
                className="flex items-center gap-1.5 rounded-lg bg-destructive/10 px-3 py-2 text-sm font-medium text-destructive transition-colors hover:bg-destructive/20 disabled:opacity-50"
              >
                {running ? <Square className="h-4 w-4" /> : <Play className="h-4 w-4" />}
                {running ? "停止" : "启动"}
              </button>
            </div>
          </div>
          <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
            {serviceInfo.map(([k, v]) => (
              <div key={k}>
                <div className="text-xs text-muted-foreground">{k}</div>
                <div className="text-sm font-semibold text-foreground">{v}</div>
              </div>
            ))}
          </div>
          <div className="mt-3 rounded-lg bg-muted/40 px-3 py-2 font-mono text-xs text-muted-foreground">
            {path === DEFAULT_PATH ? "当前运行配置（内部 config.yaml）" : path}
          </div>
          {modeInfo.protected_warning && (
            <div className="mt-3 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs leading-5 text-amber-700 dark:text-amber-300">
              {modeInfo.protected_warning}
            </div>
          )}
        </div>

        <div className="overflow-hidden rounded-xl border border-border/60 bg-card shadow-sm">
          <div className="flex items-center gap-2 border-b border-border/50 px-4 py-3">
            <h3 className="font-semibold text-foreground">配置文件编辑器</h3>
            {dirty && (
              <span className="rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-600 dark:text-amber-400">
                未保存
              </span>
            )}
            <div className="ml-auto flex items-center gap-1.5">
              <button onClick={newTemplate} className="rounded-lg border border-border/60 p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground" aria-label="新建模板" title="新建自定义模板">
                <FileText className="h-4 w-4" />
              </button>
              <button onClick={restoreDefault} disabled={acting} className="rounded-lg border border-border/60 p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-50" aria-label="恢复预设" title="恢复 MSF 预设配置">
                <RotateCcw className="h-4 w-4" />
              </button>
              <button onClick={() => void reloadAll()} disabled={loading} className="rounded-lg border border-border/60 p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-50" aria-label="刷新配置" title="刷新配置">
                <RefreshCw className={cn("h-4 w-4", loading && "animate-spin")} />
              </button>
              <button onClick={copy} disabled={!content} className="rounded-lg border border-border/60 p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-50" aria-label="复制配置" title="复制配置">
                <Copy className="h-4 w-4" />
              </button>
              <button onClick={() => setFullscreen(true)} className="rounded-lg border border-border/60 p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground" aria-label="全屏编辑" title="全屏编辑">
                <Maximize2 className="h-4 w-4" />
              </button>
              <input ref={fileRef} type="file" accept=".yaml,.yml" className="hidden" onChange={(e) => void importCustomConfig(e.target.files?.[0] ?? null)} />
              <button onClick={() => fileRef.current?.click()} className="rounded-lg border border-border/60 p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground" aria-label="导入自定义配置" title="导入自定义配置">
                <Upload className="h-4 w-4" />
              </button>
              <button onClick={download} disabled={!content} className="rounded-lg border border-border/60 p-2 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:opacity-50" aria-label="下载" title="下载">
                <Download className="h-4 w-4" />
              </button>
            </div>
          </div>

          {loading ? (
            <div className="flex h-[460px] items-center justify-center bg-[#1e1e1e] text-sm text-[#d4d4d4]">
              正在加载配置...
            </div>
          ) : (
            <YamlEditor
              value={content}
              onChange={(value) => {
                setContent(value);
                setDirty(true);
              }}
            />
          )}

          {warnings.length > 0 && (
            <div className="border-t border-amber-500/20 bg-amber-500/10 px-4 py-3 text-xs leading-5 text-amber-700 dark:text-amber-300">
              <div className="mb-1 flex items-center gap-1.5 font-semibold">
                <TriangleAlert className="h-4 w-4" />
                关键字段告警
              </div>
              <ul className="list-inside list-disc space-y-0.5">
                {warnings.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          )}

          <div className="flex flex-wrap items-center gap-2 border-t border-border/50 px-4 py-3">
            <p className="font-mono text-xs text-muted-foreground">{path === DEFAULT_PATH ? "当前运行配置" : fileName(path)}</p>
            <div className="ml-auto flex items-center gap-2">
              <button onClick={validate} disabled={!content} className="flex items-center gap-1.5 rounded-lg border border-border/60 px-3 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted disabled:opacity-50">
                <CheckCircle2 className="h-4 w-4" />
                验证配置
              </button>
              <button disabled={!content || saving} onClick={() => void save()} className="flex items-center gap-1.5 rounded-lg border border-border/60 px-3 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted disabled:pointer-events-none disabled:opacity-50">
                <Save className="h-4 w-4" />
                {saving ? "保存中" : "保存配置"}
              </button>
              <button disabled={!content || saving} onClick={() => void save(true)} className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:pointer-events-none disabled:opacity-50">
                <RotateCw className="h-4 w-4" />
                保存并应用
              </button>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div className="rounded-xl border border-border/60 bg-card p-4 shadow-sm">
            <div className="mb-3 flex items-center gap-2">
              <Cpu className="h-5 w-5 text-primary" />
              <h3 className="font-semibold text-foreground">内核版本</h3>
            </div>
            <div className="font-mono text-sm text-foreground">{version || status?.version || "-"}</div>
            <div className="mt-1 text-xs text-muted-foreground">{serviceStatusText(status)}</div>
          </div>
          <div className="rounded-xl border border-border/60 bg-card p-4 shadow-sm">
            <div className="mb-3 flex items-center gap-2">
              <FileText className="h-5 w-5 text-primary" />
              <h3 className="font-semibold text-foreground">用户配置</h3>
              <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">{files.length}</span>
            </div>
            <div className="max-h-[220px] space-y-1.5 overflow-y-auto pr-1">
              {files.length === 0 && (
                <div className="rounded-lg border border-dashed border-border/70 px-3 py-6 text-center text-sm text-muted-foreground">
                  暂无用户配置。可通过新建模板、上传配置或保存当前编辑内容创建。
                </div>
              )}
              {files.map((file) => {
                const itemPath = configPathFor(file);
                const active = itemPath === path;
                return (
                  <div
                    key={itemPath}
                    className={cn(
                      "flex items-center gap-2 rounded-lg px-3 py-2 transition-colors",
                      active ? "bg-primary/10 text-primary" : "bg-muted/40 text-foreground hover:bg-muted"
                    )}
                  >
                    <button
                      onClick={() => switchFile(file)}
                      className="min-w-0 flex-1 text-left"
                    >
                      <div className="truncate font-mono text-sm">{file.name || fileName(itemPath)}</div>
                      <div className="mt-0.5 flex items-center gap-2 text-[11px] text-muted-foreground">
                        {file.active && <span className="text-green-500">已应用</span>}
                        {file.modified && <span>{file.modified}</span>}
                      </div>
                    </button>
                    <button
                      onClick={() => void applyUserConfig(file)}
                      disabled={acting}
                      className="rounded-md border border-border/60 px-2 py-1 text-xs text-foreground hover:bg-background disabled:opacity-50"
                    >
                      应用
                    </button>
                    <button
                      onClick={() => void deleteUserConfig(file)}
                      disabled={acting}
                      className="rounded-md p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive disabled:opacity-50"
                      aria-label="删除用户配置"
                      title="删除用户配置"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>

      {fullscreen && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center bg-black/70 p-4 backdrop-blur-sm animate-fade-in">
          <div className="flex h-[90vh] w-full max-w-6xl flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-2xl animate-scale-in">
            <div className="flex items-center gap-3 border-b border-border/50 px-4 py-3">
              <h3 className="font-semibold text-foreground">{path === DEFAULT_PATH ? "当前运行配置" : fileName(path)}</h3>
              {dirty && <span className="rounded-full bg-amber-500/10 px-2 py-0.5 text-xs font-medium text-amber-600 dark:text-amber-400">未保存</span>}
              <button onClick={() => setFullscreen(false)} className="ml-auto rounded-lg p-2 text-muted-foreground hover:bg-muted hover:text-foreground" aria-label="关闭">
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="min-h-0 flex-1 overflow-hidden">
              <YamlEditor
                value={content}
                maxHeight="100%"
                className="h-full"
                onChange={(value) => {
                  setContent(value);
                  setDirty(true);
                }}
              />
            </div>
            <div className="flex justify-end gap-2 border-t border-border/50 px-4 py-3">
              <button onClick={() => setFullscreen(false)} className="rounded-lg border border-border px-3 py-2 text-sm hover:bg-muted">关闭</button>
              <button disabled={!content || saving} onClick={() => void save()} className="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50">
                {saving ? "保存中" : "保存配置"}
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  );
}
