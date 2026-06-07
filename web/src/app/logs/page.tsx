"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams } from "react-router-dom";
import { Download, Pause, Play, RefreshCw, Search, Trash2 } from "lucide-react";
import { AppShell } from "@/components/AppShell";
import { useToaster, ToastStack } from "@/components/Toaster";
import { api, apiList } from "@/lib/api";
import { cn } from "@/lib/utils";

type Service = "msf" | "mosdns" | "singbox" | "mihomo";
type Level = "ERROR" | "WARN" | "INFO" | "DEBUG";

interface LogEntry {
  time?: string;
  timestamp?: string;
  level?: string;
  message?: string;
  display?: string;
  raw?: string;
  source?: string;
}

const services: Service[] = ["msf", "mosdns", "singbox", "mihomo"];
const levels: Array<"all" | Level> = ["all", "ERROR", "WARN", "INFO", "DEBUG"];
const badgeClass: Record<Level, string> = {
  ERROR: "bg-red-600 dark:bg-red-500 text-white",
  WARN: "bg-yellow-500 dark:bg-yellow-600 text-white",
  INFO: "bg-green-600 dark:bg-green-500 text-white",
  DEBUG: "bg-gray-500 dark:bg-gray-600 text-white",
};
const hardBorder = "border-neutral-950/70 dark:border-zinc-400/90";
const toolIconBtn =
  "inline-flex h-8 w-8 items-center justify-center rounded-lg border bg-background text-foreground transition-colors hover:bg-accent disabled:opacity-50";

function normalizeService(value?: string): Service {
  if (value === "mosdns" || value === "singbox" || value === "mihomo") return value;
  return "msf";
}

function normalizeLevel(value?: string): Level {
  const upper = String(value || "").toUpperCase();
  if (upper.includes("ERR") || upper.includes("FATAL") || upper.includes("PANIC")) return "ERROR";
  if (upper.includes("WARN") || upper.includes("WRN")) return "WARN";
  if (upper.includes("DEBUG") || upper.includes("TRACE")) return "DEBUG";
  return "INFO";
}

function cleanTime(value?: string) {
  const text = String(value || "").trim();
  if (!text) return "";
  const normalized = text
    .replace("T", " ")
    .replace(/^(\d{4})\/(\d{2})\/(\d{2})/, "$1-$2-$3")
    .replace(/Z$/, "")
    .replace(/([+-]\d{2}:?\d{2})$/, "")
    .trim();
  return normalized.replace(/\.(\d{3})\d+$/, ".$1");
}

function parseLineParts(value?: string) {
  const text = String(value || "").trim();
  if (!text) return {};

  const logfmt = text.match(/(?:^|\s)time=("[^"]+"|\S+)\s+level=([a-zA-Z]+)\s+msg=("[\s\S]*"|\S[\s\S]*)$/);
  if (logfmt) {
    const rawMessage = logfmt[3].trim();
    const message = rawMessage.startsWith('"') && rawMessage.endsWith('"') ? rawMessage.slice(1, -1) : rawMessage;
    return { time: cleanTime(logfmt[1].replace(/^"|"$/g, "")), level: normalizeLevel(logfmt[2]), message };
  }

  const timeLevel = text.match(
    /^(\d{4}[-/]\d{2}[-/]\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\s+(?:\[?([a-zA-Z]+)\]?)\s+(.+)$/
  );
  if (timeLevel) {
    if (/^(fatal|error|erro|err|warn|warning|wrn|info|debug|trace)$/i.test(timeLevel[2])) {
      return { time: cleanTime(timeLevel[1]), level: normalizeLevel(timeLevel[2]), message: timeLevel[3] };
    }
    return { time: cleanTime(timeLevel[1]), message: `${timeLevel[2]} ${timeLevel[3]}` };
  }

  const timeOnly = text.match(
    /^(\d{4}[-/]\d{2}[-/]\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\s+(.+)$/
  );
  if (timeOnly) {
    return { time: cleanTime(timeOnly[1]), message: timeOnly[2] };
  }

  const bracketLevel = text.match(/^\[?([a-zA-Z]+)\]?\s+(.+)$/);
  if (bracketLevel && /^(fatal|error|erro|err|warn|warning|wrn|info|debug|trace)$/i.test(bracketLevel[1])) {
    return { level: normalizeLevel(bracketLevel[1]), message: bracketLevel[2] };
  }

  return {};
}

function formatEntry(entry: LogEntry) {
  const preferred = entry.display || entry.message || entry.raw || "";
  const candidates = [entry.raw, entry.display, entry.message].filter(Boolean) as string[];
  const parsed = candidates.map(parseLineParts).find((item) => item.time || item.level || item.message) || {};
  const rawTime = cleanTime(entry.time || entry.timestamp);
  const parsedTime = parsed.time || "";
  const time = parsedTime || rawTime;
  const level = normalizeLevel(parsed.level || entry.level);
  const message = parsed.message || preferred;
  return { time, level, message };
}

function entryMessage(entry: LogEntry) {
  return formatEntry(entry).message;
}

function logIdentity(entry: LogEntry) {
  const formatted = formatEntry(entry);
  return `${entry.source || ""}|${entry.raw || ""}|${formatted.time}|${formatted.level}|${formatted.message}`;
}

export default function LogsPage({ initialService }: { initialService?: Service } = {}) {
  const params = useParams();
  const { toasts, showToast } = useToaster();
  const [service, setService] = useState<Service>(() => initialService || normalizeService(params.service));
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [stats, setStats] = useState({ total: 0, error: 0, warn: 0 });
  const [query, setQuery] = useState("");
  const [level, setLevel] = useState<"all" | Level>("all");
  const [paused, setPaused] = useState(false);
  const [loading, setLoading] = useState(false);
  const serviceLocked = !!initialService;

  useEffect(() => {
    setService(initialService || normalizeService(params.service));
  }, [params.service, initialService]);

  const load = useCallback(async () => {
    if (paused) return;
    setLoading(true);
    try {
      const search = new URLSearchParams({ lines: "1000" });
      if (level !== "all") search.set("level", level.toLowerCase());
      if (query.trim()) search.set("q", query.trim());
      const payload = await api<any>(`/api/v1/logs/${service}?${search}`);
      const items = apiList<LogEntry>(payload, ["logs", "items", "data"]);
      setLogs(items);
      setStats({
        total: Number(payload.stats?.total ?? items.length),
        error: Number(payload.stats?.error ?? 0),
        warn: Number(payload.stats?.warn ?? 0),
      });
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [level, paused, query, service, showToast]);

  useEffect(() => {
    void load();
    const id = window.setInterval(() => void load(), 8000);
    return () => window.clearInterval(id);
  }, [load]);

  useEffect(() => {
    if (paused) return;
    const token = window.localStorage.getItem("msf_token") || "";
    const search = new URLSearchParams({ lines: "80" });
    if (token) search.set("token", token);
    if (level !== "all") search.set("level", level.toLowerCase());
    const source = new EventSource(`/api/v1/events/logs/${service}?${search}`);
    const onLogs = (event: MessageEvent) => {
      try {
        const payload = JSON.parse(event.data);
        const incoming = apiList<LogEntry>(payload, ["logs", "items", "data"]);
        if (incoming.length === 0) return;
        setLogs((current) => {
          const seen = new Set(current.map(logIdentity));
          const merged = [...current];
          for (const item of incoming) {
            const key = logIdentity(item);
            if (!seen.has(key)) {
              seen.add(key);
              merged.push(item);
            }
          }
          return merged.slice(-1000);
        });
      } catch {
        // Ignore malformed stream frames and keep the polling fallback alive.
      }
    };
    source.addEventListener("logs", onLogs as EventListener);
    return () => source.close();
  }, [level, paused, service]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return logs.filter((entry) => !q || entryMessage(entry).toLowerCase().includes(q));
  }, [logs, query]);

  const clearLogs = async () => {
    try {
      await api(`/api/v1/logs/${service}`, { method: "DELETE" });
      showToast("日志已清空");
      await load();
    } catch (err) {
      showToast(err instanceof Error ? err.message : String(err));
    }
  };

  const download = () => {
    window.location.href = `/api/v1/logs/${service}/download?token=${encodeURIComponent(window.localStorage.getItem("msf_token") || "")}`;
  };

  return (
    <AppShell>
      <div className="space-y-4 animate-fade-in">
        <ToastStack toasts={toasts} />
        <div className="flex flex-col justify-between gap-3 md:flex-row md:items-center">
          <h1 className="text-2xl font-bold tracking-tight text-foreground md:text-3xl">日志查看</h1>
          {serviceLocked ? (
            <span className="inline-flex items-center rounded-full border border-transparent bg-secondary px-3 py-1 text-xs font-semibold text-secondary-foreground">
              {service.toUpperCase()}
            </span>
          ) : (
            <select
              value={service}
              onChange={(event) => {
                setService(event.target.value as Service);
                setLevel("all");
                setQuery("");
              }}
              className={cn("w-full rounded-md border bg-background px-3 py-2 text-sm md:w-auto", hardBorder)}
            >
              {services.map((name) => (
                <option key={name} value={name}>{name.toUpperCase()}</option>
              ))}
            </select>
          )}
        </div>

        <section className="space-y-3 rounded-[12px] border border-border/20 bg-card p-4 text-card-foreground">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                onKeyDown={(event) => event.key === "Enter" && void load()}
                placeholder="搜索日志..."
                className={cn("w-full rounded-lg border bg-background py-2 pl-10 pr-4 text-sm text-foreground transition-all placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/30", hardBorder)}
              />
            </div>
            <select
              value={level}
              onChange={(event) => setLevel(event.target.value as "all" | Level)}
              className={cn("rounded-lg border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/30", hardBorder)}
            >
              {levels.map((item) => (
                <option key={item} value={item}>{item === "all" ? "全部" : item}</option>
              ))}
            </select>
            <div className="flex items-center gap-2">
              <button onClick={() => setPaused((value) => !value)} className={cn(toolIconBtn, paused && "border-green-500/40 bg-green-500/10 text-green-600 dark:text-green-400")} title={paused ? "继续" : "暂停"} aria-label={paused ? "继续" : "暂停"}>
                {paused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
              </button>
              <button onClick={() => void load()} disabled={loading} className={cn(toolIconBtn, hardBorder)} title="刷新" aria-label="刷新">
                <RefreshCw className={cn("h-4 w-4", loading && "animate-spin")} />
              </button>
              <button onClick={download} className={cn(toolIconBtn, hardBorder)} title="下载" aria-label="下载">
                <Download className="h-4 w-4" />
              </button>
              <button onClick={() => void clearLogs()} className={cn(toolIconBtn, "text-destructive hover:bg-destructive/10", hardBorder)} title="清空" aria-label="清空">
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
            <div className="flex items-center gap-4 whitespace-nowrap text-xs">
              <span className="text-muted-foreground">总日志数: <span className="font-semibold text-foreground">{stats.total}</span></span>
              {stats.error > 0 && <span className="text-destructive">ERROR: {stats.error}</span>}
              {stats.warn > 0 && <span className="text-yellow-600">WARN: {stats.warn}</span>}
            </div>
          </div>

          <div className={cn("h-[calc(100vh-280px)] min-h-[400px] overflow-auto rounded-md border bg-muted/50 p-3 font-mono text-xs leading-relaxed", hardBorder)}>
            {loading && filtered.length === 0 ? (
              <div className="py-12 text-center text-sm text-muted-foreground">正在加载日志...</div>
            ) : filtered.length === 0 ? (
              <div className="py-12 text-center text-sm text-muted-foreground">暂无日志</div>
            ) : (
              <div className="space-y-0.5">
                {filtered.map((entry, index) => {
                  const formatted = formatEntry(entry);
                  return (
                    <div key={`${formatted.time || entry.timestamp || index}-${index}`} className="flex min-w-max gap-2 rounded px-2 py-0.5 transition-colors hover:bg-accent/50">
                      <span className="w-[154px] shrink-0 whitespace-nowrap text-[10px] text-muted-foreground">{formatted.time}</span>
                      <span className={cn("inline-flex h-4 shrink-0 items-center rounded-full px-1 text-[10px] font-semibold leading-none", badgeClass[formatted.level])}>
                        {formatted.level}
                      </span>
                      <span className="break-all text-foreground">{formatted.message}</span>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </section>
      </div>
    </AppShell>
  );
}
