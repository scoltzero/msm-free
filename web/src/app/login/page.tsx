import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Eye, EyeOff, Lock, LogIn, Network, Server, Shield, User } from "lucide-react";
import { LoginLogoShowcase } from "@/components/login/LoginLogoShowcase";
import { useAuth } from "@/lib/auth";
import { api, apiData } from "@/lib/api";

const features = [
  { icon: Server, label: "DNS 服务" },
  { icon: Shield, label: "代理管理" },
  { icon: Network, label: "网络优化" },
];

export default function LoginPage() {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const { login } = useAuth();
  const [showPassword, setShowPassword] = useState(false);
  const [username, setUsername] = useState("root");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [releaseVersion, setReleaseVersion] = useState("未知");

  useEffect(() => {
    let cancelled = false;
    api<any>("/api/v1/version", { skipAuth: true })
      .then((payload) => {
        const version = apiData<{ version?: string }>(payload)?.version;
        if (!cancelled && version) {
          setReleaseVersion(`v ${version}`);
        }
      })
      .catch(() => {
        /* leave as 未知 */
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError("");
    setBusy(true);
    try {
      await login(username, password);
      const redirect = params.get("redirect") || "/";
      navigate(redirect.startsWith("/") && !redirect.startsWith("//") ? redirect : "/", { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-h-screen flex bg-gradient-to-br from-slate-50 via-blue-50/30 to-slate-100 dark:from-slate-950 dark:via-blue-950/30 dark:to-slate-900">
      <div className="hidden lg:flex lg:w-1/2 xl:w-3/5 flex-col justify-center items-center p-12 relative overflow-hidden">
        <div className="absolute inset-0 opacity-20">
          <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-primary/40 rounded-full blur-3xl animate-pulse-subtle" />
          <div className="absolute bottom-1/4 right-1/4 w-80 h-80 bg-blue-400/30 rounded-full blur-3xl animate-pulse-subtle" />
        </div>
        <div className="relative z-10 text-center space-y-4 max-w-lg">
          <div className="flex flex-col items-center gap-6">
            <LoginLogoShowcase />
          </div>
          <div className="space-y-4 animate-fade-in">
            <h1 className="text-4xl xl:text-5xl font-bold text-slate-800 dark:text-slate-100">
              MSF 管理平台
            </h1>
            <p className="text-sm text-slate-500 dark:text-slate-500 max-w-md mx-auto">
              统一管理您的网络服务，提供 DNS 分流、代理管理等功能
            </p>
          </div>
          <div className="flex justify-center gap-8 pt-3 opacity-70">
            {features.map((feature) => {
              const Icon = feature.icon;
              return (
                <div key={feature.label} className="flex flex-col items-center gap-2">
                  <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center">
                    <Icon className="h-6 w-6 text-primary" />
                  </div>
                  <span className="text-xs text-slate-600 dark:text-slate-400">{feature.label}</span>
                </div>
              );
            })}
          </div>
        </div>
        <div
          data-login-version
          className="absolute bottom-8 left-1/2 -translate-x-1/2 text-xs text-slate-500 dark:text-slate-600"
        >
          {releaseVersion}
        </div>
      </div>

      <div className="flex-1 flex items-center justify-center p-6 lg:p-12">
        <div className="text-card-foreground w-full max-w-md animate-scale-in border-0 bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-3xl shadow-none">
          <div className="p-8 pb-4">
            <div className="mb-6 flex justify-center lg:hidden">
              <LoginLogoShowcase compact />
            </div>
            <p className="text-base font-medium text-slate-600 dark:text-slate-400">欢迎使用 MSF</p>
          </div>
          <div className="px-8 pb-8">
            <form className="space-y-5" onSubmit={submit}>
              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700 dark:text-slate-300">用户名</label>
                <div className="relative group">
                  <User className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-slate-400 group-focus-within:text-primary" />
                  <input
                    type="text"
                    required
                    value={username}
                    onChange={(event) => setUsername(event.target.value)}
                    placeholder="请输入用户名"
                    className="w-full pl-11 pr-4 py-3 rounded-xl border-2 border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-slate-800 dark:text-slate-100 text-sm placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-slate-700 dark:text-slate-300">密码</label>
                <div className="relative group">
                  <Lock className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-slate-400 group-focus-within:text-primary" />
                  <input
                    type={showPassword ? "text" : "password"}
                    required
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    placeholder="请输入密码"
                    className="w-full pl-11 pr-11 py-3 rounded-xl border-2 border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-slate-800 dark:text-slate-100 text-sm placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all"
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword((value) => !value)}
                    aria-label="显示密码"
                    className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
                  >
                    {showPassword ? <EyeOff className="h-5 w-5" /> : <Eye className="h-5 w-5" />}
                  </button>
                </div>
              </div>

              {error && <div className="rounded-xl border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-600">{error}</div>}

              <button
                type="submit"
                disabled={busy}
                className="inline-flex items-center justify-center gap-2 w-full h-12 bg-gradient-to-r from-primary via-blue-500 to-primary hover:shadow-xl text-white transition-all hover:scale-[1.02] disabled:hover:scale-100 rounded-xl font-medium text-base mt-6 disabled:opacity-60"
              >
                <LogIn className="h-5 w-5" />
                {busy ? "登录中..." : "登录"}
              </button>
            </form>

            <div className="mt-6 text-center">
              <p className="text-xs text-slate-500 dark:text-slate-500">请使用初始化时创建的账号登录</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
