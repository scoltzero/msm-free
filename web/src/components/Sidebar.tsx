"use client";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronUp, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";
import { navItems } from "@/lib/dashboard-data";
import type { NavItem } from "@/types";

/**
 * A single nav row. The element is identical in both states — only the layout
 * classes change (centered icon + no padding when collapsed) and the label span
 * is omitted. This mirrors the original: the content swaps instantly while the
 * sidebar width animates, so the label is revealed/hidden by the widening clip
 * rather than by mounting a different component.
 */
function NavRow({
  item,
  indent,
  active,
  collapsed,
  flex1,
}: {
  item: NavItem;
  indent?: boolean;
  active: boolean;
  collapsed: boolean;
  flex1?: boolean;
}) {
  const Icon = item.icon;
  return (
    <Link
      href={item.href}
      title={collapsed ? item.label : undefined}
      aria-label={collapsed ? item.label : undefined}
      className={cn(
        "flex items-center gap-3 py-2.5 rounded-[10px] transition-all group/item",
        flex1 && "flex-1",
        collapsed ? "justify-center px-0" : "px-3 group-hover:pl-8",
        indent && !collapsed && "ml-4",
        active
          ? "bg-primary/10 text-primary font-medium shadow-sm"
          : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
      )}
    >
      <Icon className={cn("h-5 w-5 flex-shrink-0", active && "text-primary")} />
      {!collapsed && <span className="text-sm whitespace-nowrap">{item.label}</span>}
    </Link>
  );
}

function itemMatchesPath(item: NavItem, pathname: string) {
  if (item.href === "/") return pathname === "/";
  if (pathname === item.href || pathname.startsWith(`${item.href}/`)) return true;
  return item.children?.some((child) => pathname === child.href || pathname.startsWith(`${child.href}/`)) ?? false;
}

function NavGroup({
  item,
  pathname,
  collapsed,
}: {
  item: NavItem;
  pathname: string;
  collapsed: boolean;
}) {
  const hasActiveChild = item.children?.some((child) => itemMatchesPath(child, pathname));
  const [open, setOpen] = useState(true);
  const parentActive = pathname === item.href;
  const active = parentActive || Boolean(hasActiveChild);

  // Collapsed: the whole group folds down to a single icon row (no chevron,
  // no children) — exactly like the original compact rail.
  if (collapsed) {
    return (
      <div className="group relative">
        <NavRow item={item} active={active} collapsed />
      </div>
    );
  }

  return (
    <div>
      <div className="group relative flex items-center">
        <NavRow item={item} active={active} collapsed={false} flex1 />
        <button
          onClick={() => setOpen((v) => !v)}
          className="p-1 rounded hover:bg-accent/70 transition-colors flex items-center justify-center flex-shrink-0"
          aria-label={open ? "收起" : "展开"}
        >
          {open ? (
            <ChevronUp className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          )}
        </button>
      </div>
      {open && item.children && (
        <div className="mt-1 space-y-1">
          {item.children.map((child) => (
            <div key={child.href} className="group relative">
              <NavRow item={child} indent active={child.href === pathname} collapsed={false} />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export function Sidebar({ collapsed = false }: { collapsed?: boolean }) {
  const pathname = usePathname();
  const navRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    const nav = navRef.current;
    if (!nav) return;
    const saved = Number(window.sessionStorage.getItem("msf-sidebar-scroll") || 0);
    window.requestAnimationFrame(() => {
      nav.scrollTop = saved;
    });
  }, [pathname]);

  return (
    <aside
      className={cn(
        "hidden md:block fixed left-0 top-14 md:top-16 z-40 h-[calc(100vh-3.5rem)] md:h-[calc(100vh-4rem)] border-r border-border bg-sidebar transition-all duration-300",
        collapsed ? "w-20" : "w-56"
      )}
    >
      <div className="flex flex-col h-full">
        <nav
          ref={navRef}
          onScroll={(event) => {
            window.sessionStorage.setItem("msf-sidebar-scroll", String(event.currentTarget.scrollTop));
          }}
          className="flex-1 overflow-y-auto overflow-x-hidden scrollbar-thin px-3 py-4 space-y-1"
        >
          {navItems.map((item) => {
            const active = itemMatchesPath(item, pathname);
            return item.children ? (
              <NavGroup key={item.href} item={item} pathname={pathname} collapsed={collapsed} />
            ) : (
              <div key={item.href} className="group relative">
                <NavRow item={item} active={active} collapsed={collapsed} />
              </div>
            );
          })}
        </nav>
      </div>
    </aside>
  );
}
