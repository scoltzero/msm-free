import { Box, FileCode, ScrollText, TriangleAlert } from "lucide-react";
import { AppShell } from "@/components/AppShell";
import { useApiPath } from "@/lib/use-api";

export function SingBoxPage({ tab }: { tab: "overview" | "config" | "logs" }) {
  const status = useApiPath<any>("/api/v1/singbox/status", [], 5000);
  const data = status.data?.data || {};

  return (
    <AppShell>
      <div className="space-y-5 animate-fade-in">
        <div className="flex items-center gap-3">
          <div className="rounded-xl bg-muted p-2 text-muted-foreground">
            <Box className="h-6 w-6" />
          </div>
          <div>
            <h1 className="text-2xl font-bold">Sing-Box</h1>
            <p className="text-sm text-muted-foreground">当前版本暂未启用 Sing-Box 管理能力</p>
          </div>
        </div>

        <section className="rounded-xl border bg-card p-5">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="space-y-2">
              <div className="inline-flex items-center gap-2 rounded-full bg-yellow-500/10 px-3 py-1 text-sm font-medium text-yellow-700 dark:text-yellow-300">
                <TriangleAlert className="h-4 w-4" />
                {data.status || "disabled"}
              </div>
              <h2 className="text-lg font-semibold">{data.display_name || "Sing-Box"} 未配置</h2>
              <p className="max-w-2xl text-sm text-muted-foreground">
                {data.message || "sing-box is not implemented in msf x86 first version"}
              </p>
            </div>
          </div>
        </section>

        {tab === "config" && (
          <section className="rounded-xl border bg-card p-5">
            <div className="mb-4 flex items-center gap-2 font-semibold">
              <FileCode className="h-5 w-5 text-primary" />
              配置管理
            </div>
            <pre className="min-h-40 rounded-lg border bg-muted/40 p-4 text-sm text-muted-foreground">暂无 Sing-Box 配置</pre>
          </section>
        )}

        {tab === "logs" && (
          <section className="rounded-xl border bg-card p-5">
            <div className="mb-4 flex items-center gap-2 font-semibold">
              <ScrollText className="h-5 w-5 text-primary" />
              日志查看
            </div>
            <div className="rounded-lg border bg-muted/40 p-8 text-center text-sm text-muted-foreground">暂无日志</div>
          </section>
        )}
      </div>
    </AppShell>
  );
}
