"use client";

export type DashboardComponentKey =
  | "device"
  | "hardware"
  | "rate"
  | "stats"
  | "resources"
  | "mosdns"
  | "singbox"
  | "mihomo";

export interface DashboardSettings {
  compact: boolean;
  visible: Record<DashboardComponentKey, boolean>;
}

export const DASHBOARD_SETTINGS_EVENT = "msf:dashboard-settings";
export const DASHBOARD_SETTINGS_STORAGE_KEY = "msf.dashboard.settings.v1";

export const dashboardComponentOptions: Array<{ key: DashboardComponentKey; label: string }> = [
  { key: "device", label: "设备信息" },
  { key: "hardware", label: "硬件信息" },
  { key: "rate", label: "实时速率" },
  { key: "stats", label: "统计信息" },
  { key: "resources", label: "资源趋势" },
  { key: "mosdns", label: "MosDNS" },
  { key: "singbox", label: "Sing-Box" },
  { key: "mihomo", label: "Mihomo" },
];

export const defaultDashboardSettings: DashboardSettings = {
  compact: false,
  visible: {
    device: true,
    hardware: true,
    rate: true,
    stats: true,
    resources: true,
    mosdns: true,
    singbox: true,
    mihomo: true,
  },
};

export function loadDashboardSettings(): DashboardSettings {
  if (typeof window === "undefined") return defaultDashboardSettings;
  try {
    const raw = window.localStorage.getItem(DASHBOARD_SETTINGS_STORAGE_KEY);
    if (!raw) return defaultDashboardSettings;
    const parsed = JSON.parse(raw) as Partial<DashboardSettings>;
    return {
      compact: Boolean(parsed.compact),
      visible: {
        ...defaultDashboardSettings.visible,
        ...(parsed.visible || {}),
      },
    };
  } catch {
    return defaultDashboardSettings;
  }
}

export function saveDashboardSettings(settings: DashboardSettings) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(DASHBOARD_SETTINGS_STORAGE_KEY, JSON.stringify(settings));
  window.dispatchEvent(new CustomEvent(DASHBOARD_SETTINGS_EVENT, { detail: settings }));
}
