import {
  LayoutDashboard,
  Server,
  Box,
  Network,
  Activity,
  FileText,
  ScrollText,
  Users,
  Stethoscope,
  Settings,
  ChartColumn,
  Shield,
  Search,
  Cog,
  Cable,
  FileCog,
  FileCode,
  Zap,
} from "lucide-react";
import type { InfoRow, NavItem, ServiceStatus } from "@/types";

export const navItems: NavItem[] = [
  { label: "仪表盘", href: "/", icon: LayoutDashboard },
  {
    label: "MosDNS",
    href: "/mosdns",
    icon: Server,
    children: [
      { label: "概述", href: "/mosdns/overview", icon: ChartColumn },
      { label: "规则管理", href: "/mosdns/rules", icon: Shield },
      { label: "客户端设置", href: "/mosdns/clients", icon: Users },
      { label: "DNS 日志", href: "/mosdns/query-log", icon: Search },
      { label: "系统功能", href: "/mosdns/system", icon: Cog },
      { label: "配置管理", href: "/mosdns/service-config", icon: Settings },
      { label: "实时日志", href: "/mosdns/logs", icon: FileText },
    ],
  },
  { label: "代理服务", href: "/proxy", icon: Box },
  {
    label: "Mihomo",
    href: "/mihomo",
    icon: Network,
    children: [
      { label: "概览", href: "/mihomo/overview", icon: ChartColumn },
      { label: "代理节点", href: "/mihomo/proxies", icon: Network },
      { label: "规则管理", href: "/mihomo/rules", icon: Shield },
      { label: "连接管理", href: "/mihomo/connections", icon: Cable },
      { label: "配置管理", href: "/mihomo/config", icon: FileCog },
      { label: "日志查看", href: "/mihomo/logs", icon: FileCode },
    ],
  },
  { label: "进程管理", href: "/process", icon: Activity },
  { label: "配置管理", href: "/config", icon: FileText },
  { label: "日志查看", href: "/logs", icon: ScrollText },
  { label: "用户管理", href: "/users", icon: Users },
  { label: "系统诊断", href: "/system", icon: Stethoscope },
  { label: "系统设置", href: "/settings", icon: Settings },
];

/** Top-level entries shown in the mobile bottom nav bar (groups appear as a single button). */
export const mobileNavItems: NavItem[] = navItems.map((item) => ({
  label: item.label,
  href: item.href,
  icon: item.icon,
}));

export const deviceInfo: InfoRow[] = [
  { label: "主机名", value: "msf" },
  { label: "系统平台", value: "debian" },
  { label: "运行时间", value: "1 天 8 小时" },
  { label: "操作系统", value: "linux" },
  { label: "架构", value: "amd64" },
];

export const hardwareInfo: InfoRow[] = [
  { label: "硬件信息", value: "12th Gen Intel(R) Core(TM) i9-12900HK" },
  { label: "核心数", value: "4 核心" },
  { label: "内存", value: "1.92 GB" },
  { label: "硬盘容量", value: "18.09 GB" },
];

export const diskUsagePercent = 49.3;

export const statsInfo: InfoRow[] = [
  { label: "系统运行时间", value: "1 天 8 小时" },
  { label: "总上传流量", value: "1.99 GB" },
  { label: "总下载流量", value: "1.72 GB" },
  { label: "CPU 使用率", value: "0.2%" },
  { label: "内存使用率", value: "46.3%" },
];

export const trendStats = { cpu: "0.2%", memory: "46.3%" };

export const rateStats = {
  connections: "0",
  download: "10.1 KB/s",
  upload: "10.2 KB/s",
};

export const services: ServiceStatus[] = [
  {
    name: "MosDNS",
    icon: Server,
    configured: true,
    running: true,
    cpu: "1.0%",
    memory: "136.0 MB",
    uptime: "1 天 8 小时",
  },
  { name: "Sing-Box", icon: Box, configured: false },
  {
    name: "Mihomo",
    icon: Zap,
    configured: true,
    running: true,
    cpu: "0.0%",
    memory: "66.9 MB",
    uptime: "1 天 8 小时",
  },
];
