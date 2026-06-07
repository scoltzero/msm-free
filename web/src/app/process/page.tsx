"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Activity, RefreshCw, Square, Play, RotateCw, Clock } from "lucide-react";
import { AppShell } from "@/components/AppShell";
import { useToaster, ToastStack } from "@/components/Toaster";
import { api, apiList } from "@/lib/api";
import { cn } from "@/lib/utils";

interface ServiceItem {
  name: string;
  display_name?: string;
  displayName?: string;
  running?: boolean;
  status?: string;
  pid?: number;
  uptime_seconds?: number;
  uptime?: string;
  cpu_percent?: number;
  memory_bytes?: number;
  installed?: boolean;
  supported?: boolean;
}

const SUPPORTED_SERVICES = new Set(["msf", "mosdns", "mihomo"]);

function formatUptime(value?: number | string) {
  if (typeof value === "string" && value.trim()) return value;
  const seconds = Number(value || 0);
  if (!Number.isFinite(seconds) || seconds <= 0) return "-";
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (days > 0) return `${days}天 ${hours}小时`;
  if (hours > 0) return `${hours}小时 ${minutes}分钟`;
  return `${minutes}分钟`;
}

function logHref(name: string) {
  if (name === "mihomo") return "/mihomo/logs";
  if (name === "mosdns") return "/mosdns/logs";
  return `/logs/${name}`;
}

function isRunning(service: ServiceItem) {
  const status = String(service.status || "").toLowerCase();
  return service.running === true || ["running", "active", "ok"].includes(status);
}

export default function ProcessPage() {
  const router = useRouter();
  const { toasts, showToast } = useToaster();
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<Record<string, boolean>>({});

  const loadServices = useCallback(async () => {
    setLoading(true);
    try {
      const payload = await api("/api/v1/services");
      const list = apiList<ServiceItem>(payload, ["data", "services", "items"])
        .filter((service) => SUPPORTED_SERVICES.has(service.name))
        .filter((service) => service.supported !== false);
      setServices(list);
    } catch (error) {
      showToast(error instanceof Error ? error.message : "服务列表加载失败");
    } finally {
      setLoading(false);
    }
  }, [showToast]);

  useEffect(() => {
    loadServices();
  }, [loadServices]);

  const runningCount = services.filter(isRunning).length;
  const stoppedCount = services.length - runningCount;
  const stats: [string, number, string?][] = useMemo(
    () => [
      ["总进程数", services.length],
      ["运行中", runningCount, "text-green-600 dark:text-green-400"],
      ["已停止", stoppedCount, stoppedCount > 0 ? "text-red-500" : undefined],
    ],
    [runningCount, services.length, stoppedCount]
  );

  const runAction = async (service: ServiceItem, action: "start" | "stop" | "restart") => {
    setBusy((current) => ({ ...current, [service.name]: true }));
    try {
      await api(`/api/v1/services/${encodeURIComponent(service.name)}/${action}?wait=1&timeout=5`, { method: "POST" });
      showToast(`${service.display_name || service.displayName || service.name} ${action === "start" ? "已启动" : action === "stop" ? "已停止" : "已重启"}`);
      await loadServices();
    } catch (error) {
      showToast(error instanceof Error ? error.message : "服务操作失败");
    } finally {
      setBusy((current) => ({ ...current, [service.name]: false }));
    }
  };

  return (
    <AppShell>
      <div className="space-y-6 animate-fade-in">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-[10px] bg-gradient-to-br from-primary/10 to-secondary/10">
            <Activity className="h-6 w-6 text-primary" />
          </div>
          <div className="flex-1">
            <h1 className="text-2xl font-bold text-foreground">进程管理</h1>
            <p className="text-sm text-muted-foreground">管理所有服务进程的启动、停止和监控</p>
          </div>
          <button
            onClick={loadServices}
            disabled={loading}
            className="inline-flex items-center gap-1.5 px-3 py-2 rounded-[8px] border border-border bg-background text-sm font-medium text-foreground hover:bg-muted transition-colors disabled:opacity-50"
          >
            <RefreshCw className={cn("h-4 w-4", loading && "animate-spin")} />
            刷新
          </button>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {stats.map(([label, value, color]) => (
            <div key={label} className="bg-card p-5 rounded-[10px] border border-border/50 shadow-sm">
              <div className="text-sm text-muted-foreground">{label}</div>
              <div className={cn("text-3xl font-bold mt-1", color || "text-foreground")}>{value}</div>
            </div>
          ))}
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {services.map((service) => {
            const serviceRunning = isRunning(service);
            const serviceBusy = busy[service.name] === true;
            const displayName = service.display_name || service.displayName || service.name;
            return (
              <div
                key={service.name}
                className="rounded-[12px] border bg-card text-card-foreground !border-border/20 shadow-none hover:shadow-sm transition-all duration-200 p-5"
              >
                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-[10px] bg-gradient-to-br from-primary/10 to-secondary/10">
                    <Activity className="h-5 w-5 text-primary" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <h3 className="font-bold text-foreground leading-tight">{displayName}</h3>
                    <p className="text-sm text-muted-foreground">{service.name}</p>
                  </div>
                  <span
                    className={cn(
                      "inline-flex items-center gap-1.5 text-xs px-2 py-0.5 rounded-full font-medium",
                      serviceRunning
                        ? "bg-green-500/10 text-green-600 dark:text-green-400"
                        : "bg-muted text-muted-foreground"
                    )}
                  >
                    <span className={cn("h-1.5 w-1.5 rounded-full", serviceRunning ? "bg-green-500 animate-pulse" : "bg-muted-foreground")} />
                    {serviceRunning ? "运行中" : "已停止"}
                  </span>
                </div>

                <div className="grid grid-cols-2 gap-3 mt-4">
                  <div className="bg-muted/30 rounded-[8px] p-3">
                    <div className="text-xs text-muted-foreground">进程 ID</div>
                    <div className="text-sm font-semibold text-foreground mt-0.5 tabular-nums">{serviceRunning && service.pid ? service.pid : "-"}</div>
                  </div>
                  <div className="bg-muted/30 rounded-[8px] p-3">
                    <div className="text-xs text-muted-foreground flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      运行时长
                    </div>
                    <div className="text-sm font-semibold text-foreground mt-0.5">{serviceRunning ? formatUptime(service.uptime_seconds || service.uptime) : "-"}</div>
                  </div>
                </div>

                <div className="flex items-center gap-2 mt-4">
                  <button
                    onClick={() => runAction(service, serviceRunning ? "stop" : "start")}
                    disabled={serviceBusy || service.installed === false}
                    className={cn(
                      "flex-1 inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-[8px] text-sm font-medium transition-colors disabled:opacity-50",
                      serviceRunning
                        ? "bg-destructive text-white hover:bg-destructive/90"
                        : "bg-green-600 text-white hover:bg-green-600/90"
                    )}
                  >
                    {serviceRunning ? <Square className="h-4 w-4" /> : <Play className="h-4 w-4" />}
                    {serviceRunning ? "停止" : "启动"}
                  </button>
                  <button
                    onClick={() => runAction(service, "restart")}
                    disabled={serviceBusy || service.installed === false}
                    className="flex-1 inline-flex items-center justify-center gap-1.5 px-3 py-2 rounded-[8px] border border-border bg-background text-sm font-medium text-foreground hover:bg-muted transition-colors disabled:opacity-50"
                  >
                    <RotateCw className={cn("h-4 w-4", serviceBusy && "animate-spin")} />
                    重启
                  </button>
                  <button
                    onClick={() => router.push(logHref(service.name))}
                    className="inline-flex items-center justify-center px-3 py-2 rounded-[8px] border border-border bg-background text-sm font-medium text-foreground hover:bg-muted transition-colors"
                  >
                    查看日志
                  </button>
                </div>
              </div>
            );
          })}
          {!loading && services.length === 0 && (
            <div className="rounded-[12px] border border-dashed border-border p-8 text-center text-sm text-muted-foreground lg:col-span-2">
              后端没有返回可管理服务
            </div>
          )}
        </div>
      </div>
      <ToastStack toasts={toasts} />
    </AppShell>
  );
}
