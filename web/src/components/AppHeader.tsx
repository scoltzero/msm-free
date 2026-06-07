"use client";

import { useEffect, useState } from "react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import {
  Check,
  CircleHelp,
  Languages,
  LogOut,
  Monitor,
  Moon,
  PanelLeftClose,
  PanelLeftOpen,
  Settings,
  Sun,
  User,
  Users,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuth } from "@/lib/auth";

type ThemeMode = "light" | "dark" | "system";

const themeOptions: { id: ThemeMode; label: string; Icon: typeof Sun }[] = [
  { id: "light", label: "明亮", Icon: Sun },
  { id: "dark", label: "暗黑", Icon: Moon },
  { id: "system", label: "跟随系统", Icon: Monitor },
];

const languageOptions = ["简体中文", "English"];

function getInitialTheme(): ThemeMode {
  if (typeof window === "undefined") return "system";
  const stored = window.localStorage.getItem("msf-theme");
  if (stored === "light" || stored === "dark" || stored === "system") return stored;
  return "system";
}

function prefersDarkMode() {
  return typeof window !== "undefined" && window.matchMedia("(prefers-color-scheme: dark)").matches;
}

function applyTheme(mode: ThemeMode) {
  if (typeof window === "undefined") return;
  const shouldUseDark = mode === "dark" || (mode === "system" && prefersDarkMode());
  document.documentElement.classList.toggle("dark", shouldUseDark);
  document.documentElement.classList.toggle("light", !shouldUseDark);
  window.localStorage.setItem("msf-theme", mode);
}

export function AppHeader({ onToggleSidebar, sidebarCollapsed = false }: { onToggleSidebar?: () => void; sidebarCollapsed?: boolean }) {
  const router = useRouter();
  const { user, logout } = useAuth();
  const [theme, setTheme] = useState<ThemeMode>(() => getInitialTheme());
  const [themeOpen, setThemeOpen] = useState(false);
  const [langOpen, setLangOpen] = useState(false);
  const [userOpen, setUserOpen] = useState(false);
  const [lang, setLang] = useState("简体中文");

  const isDark = theme === "dark" || (theme === "system" && prefersDarkMode());
  const ThemeIcon = theme === "system" ? Monitor : isDark ? Moon : Sun;
  const username = user?.username || "root";
  const displayName = user?.display_name || user?.username || "root";
  const role = user?.role === "admin" ? "管理员" : user?.role || "用户";
  const initial = displayName.slice(0, 1).toUpperCase();

  useEffect(() => {
    applyTheme(theme);
    if (theme !== "system" || typeof window === "undefined") return;

    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const handleChange = () => applyTheme("system");
    media.addEventListener("change", handleChange);
    return () => media.removeEventListener("change", handleChange);
  }, [theme]);

  useEffect(() => {
    const close = () => {
      setThemeOpen(false);
      setLangOpen(false);
      setUserOpen(false);
    };
    if (themeOpen || langOpen || userOpen) {
      window.addEventListener("click", close);
      return () => window.removeEventListener("click", close);
    }
  }, [themeOpen, langOpen, userOpen]);

  const selectTheme = (mode: ThemeMode) => {
    setTheme(mode);
    applyTheme(mode);
    setThemeOpen(false);
  };

  return (
    <header className="fixed top-0 z-50 w-full border-b border-border/50 glass-effect-strong shadow-apple">
      <div className="flex h-14 items-center px-3 md:h-16 md:px-4">
        <button
          onClick={onToggleSidebar}
          className="mr-2 hidden rounded-[10px] p-2 transition-all hover:bg-accent/50 active:scale-95 md:mr-4 md:block"
          title={sidebarCollapsed ? "展开侧边栏" : "折叠侧边栏"}
          aria-label={sidebarCollapsed ? "展开侧边栏" : "折叠侧边栏"}
        >
          {sidebarCollapsed ? <PanelLeftOpen className="h-5 w-5" /> : <PanelLeftClose className="h-5 w-5" />}
        </button>

        <div className="flex items-center gap-2 md:gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] bg-gradient-to-br from-primary/10 to-secondary/10 md:h-10 md:w-10">
            <Image alt="MSF" src="/logo/logo-square.png" width={32} height={32} className="h-7 w-7 object-contain md:h-8 md:w-8" />
          </div>
          <span className="bg-gradient-to-r from-primary to-secondary bg-clip-text text-lg font-bold text-transparent md:text-xl">
            MSF
          </span>
        </div>

        <div className="ml-auto flex items-center gap-1 md:gap-2">
          <div className="relative">
            <button
              onClick={(event) => {
                event.stopPropagation();
                setThemeOpen((open) => !open);
                setLangOpen(false);
                setUserOpen(false);
              }}
              className="rounded-[10px] p-2 text-gray-700 transition-all hover:bg-accent/50 hover:text-primary active:scale-95 dark:text-gray-300"
              title="切换主题"
              aria-label="切换主题"
            >
              <ThemeIcon className="h-5 w-5" />
            </button>
            {themeOpen && (
              <div
                onClick={(event) => event.stopPropagation()}
                className="absolute right-0 z-50 mt-2 w-44 animate-slide-up rounded-xl border border-border bg-popover p-1.5 shadow-lg"
              >
                {themeOptions.map(({ id, label, Icon }) => (
                  <button
                    key={id}
                    onClick={() => selectTheme(id)}
                    className="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm text-foreground transition-colors hover:bg-accent"
                  >
                    <span className="flex items-center gap-2">
                      <Icon className="h-4 w-4 text-muted-foreground" />
                      {label}
                    </span>
                    {theme === id && <Check className="h-4 w-4 text-primary" />}
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="relative">
            <button
              onClick={(event) => {
                event.stopPropagation();
                setLangOpen((open) => !open);
                setThemeOpen(false);
                setUserOpen(false);
              }}
              className="rounded-[10px] p-2 text-gray-700 transition-all hover:bg-accent/50 hover:text-secondary active:scale-95 dark:text-gray-300"
              title="语言"
              aria-label="语言"
            >
              <Languages className="h-5 w-5" />
            </button>
            {langOpen && (
              <div
                onClick={(event) => event.stopPropagation()}
                className="absolute right-0 z-50 mt-2 w-40 animate-slide-up rounded-xl border border-border bg-popover p-1.5 shadow-lg"
              >
                {languageOptions.map((item) => (
                  <button
                    key={item}
                    onClick={() => {
                      setLang(item);
                      setLangOpen(false);
                    }}
                    className="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm text-foreground transition-colors hover:bg-accent"
                  >
                    {item}
                    {lang === item && <Check className="h-4 w-4 text-primary" />}
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="relative flex-shrink-0">
            <button
              onClick={(event) => {
                event.stopPropagation();
                setUserOpen((open) => !open);
                setThemeOpen(false);
                setLangOpen(false);
              }}
              className="flex items-center gap-2 rounded-xl px-1 py-1 transition-colors hover:bg-accent/40 md:gap-3"
              aria-label="打开用户菜单"
            >
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gradient-to-br from-primary to-secondary text-sm font-semibold text-primary-foreground shadow-apple md:h-9 md:w-9">
                {initial}
              </div>
              <div className="hidden items-center gap-2 lg:flex">
                <div className="text-left">
                  <div className="text-sm font-medium text-foreground">{displayName}</div>
                  <div className="text-xs text-muted-foreground">{role}</div>
                </div>
              </div>
            </button>
            {userOpen && (
              <div
                onClick={(event) => event.stopPropagation()}
                className="absolute right-0 z-50 mt-3 w-72 animate-slide-up rounded-2xl border border-border/70 bg-popover/95 p-2 shadow-apple-xl backdrop-blur-xl"
              >
                <div className="mb-2 flex items-center gap-3 rounded-xl bg-muted/40 px-3 py-3">
                  <div className="flex h-11 w-11 items-center justify-center rounded-full bg-gradient-to-br from-primary to-secondary text-base font-semibold text-primary-foreground shadow-apple">
                    {initial}
                  </div>
                  <div className="min-w-0">
                    <div className="truncate text-sm font-semibold text-foreground">{username}</div>
                    <div className="text-xs text-muted-foreground">{role}</div>
                  </div>
                </div>

                {[
                  { label: "个人信息", Icon: User, onClick: () => router.push("/settings?tab=profile") },
                  { label: "系统设定", Icon: Settings, onClick: () => router.push("/settings") },
                  { label: "用户管理", Icon: Users, onClick: () => router.push("/users") },
                  { label: "帮助文档", Icon: CircleHelp, onClick: () => router.push("/system") },
                ].map(({ label, Icon, onClick }) => (
                  <button
                    key={label}
                    onClick={onClick}
                    className="flex w-full items-center gap-2 rounded-xl px-3 py-2.5 text-left text-sm text-foreground transition-colors hover:bg-accent"
                  >
                    <Icon className="h-4 w-4 text-muted-foreground" />
                    {label}
                  </button>
                ))}

                <div className="my-1 border-t border-border/60" />
                <button
                  onClick={() => {
                    logout();
                    router.push("/login");
                  }}
                  className={cn(
                    "flex w-full items-center gap-2 rounded-xl px-3 py-2.5 text-left text-sm text-destructive transition-colors hover:bg-destructive/10"
                  )}
                >
                  <LogOut className="h-4 w-4" />
                  退出登录
                </button>
              </div>
            )}
          </div>
        </div>
      </div>
    </header>
  );
}
