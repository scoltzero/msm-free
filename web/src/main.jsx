import React, { useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";

const nav = [
  ["dashboard", "仪表盘", "home"],
  ["mosdns", "MosDNS", "dns"],
  ["proxy", "代理服务", "route"],
  ["mihomo", "Mihomo", "cube"],
  ["process", "进程管理", "process"],
  ["config", "配置管理", "file"],
  ["logs", "日志查看", "logs"],
  ["users", "用户管理", "users"],
  ["system", "系统诊断", "diagnostic"],
  ["settings", "系统设置", "settings"]
];

function api(path, options = {}) {
  const token = localStorage.getItem("msm_token");
  const headers = { ...(options.headers || {}) };
  if (!headers["Content-Type"] && options.body && !(options.body instanceof FormData)) {
    headers["Content-Type"] = "application/json";
  }
  if (token) headers.Authorization = `Bearer ${token}`;
  return fetch(path, { ...options, headers }).then(async (res) => {
    const type = res.headers.get("content-type") || "";
    const data = type.includes("application/json") ? await res.json() : await res.text();
    if (!res.ok) throw new Error(data?.message || data?.error || res.statusText);
    return data;
  });
}

function usePoll(loader, deps = [], interval = 3000) {
  const [data, setData] = useState(null);
  const [error, setError] = useState("");
  const run = () => loader().then(setData).catch((e) => setError(e.message));
  useEffect(() => {
    run();
    const id = setInterval(run, interval);
    return () => clearInterval(id);
  }, deps);
  return { data, error, reload: run };
}

function App() {
  const [boot, setBoot] = useState({ loading: true, initialized: false });
  const [user, setUser] = useState(null);
  const [route, setRoute] = useState("dashboard");

  useEffect(() => {
    api("/api/v1/setup/check")
      .then((res) => setBoot({ loading: false, initialized: !!res.is_initialized }))
      .catch(() => setBoot({ loading: false, initialized: false }));
  }, []);

  useEffect(() => {
    if (!localStorage.getItem("msm_token")) return;
    api("/api/v1/auth/me").then((res) => setUser(res.user)).catch(() => localStorage.removeItem("msm_token"));
  }, []);

  if (boot.loading) return <Splash />;
  if (!boot.initialized) return <SetupWizard onDone={() => setBoot({ loading: false, initialized: true })} />;
  if (!user) return <Login onLogin={setUser} />;

  const title = nav.find((item) => item[0] === route)?.[1];
  return (
    <div className="shell">
      <header className="globalHeader">
        <div className="brand">
          <div className="headerIcon">☰</div>
          <div className="brandMark">MS</div>
          <div>
            <strong>MSM</strong>
            <span>管理平台</span>
          </div>
        </div>
        <div className="headerTools">
          <button title="切换主题">◐</button>
          <button title="语言">中</button>
          <button className="userButton">
            <span className="avatar">R</span>
            <span><b>{user.username}</b><small>管理员</small></span>
          </button>
        </div>
      </header>
      <aside className="sidebar">
        <nav>
          {nav.map(([key, label, icon]) => (
            <button key={key} className={route === key ? "active" : ""} onClick={() => setRoute(key)}>
              <span className="navIndex"><NavGlyph name={icon} /></span>
              <span>{label}</span>
            </button>
          ))}
        </nav>
        <div className="sidebarFooter">
          <div>
            <strong>{user.username}</strong>
            <span>{user.role}</span>
          </div>
          <button
            className="ghost"
            onClick={() => {
              localStorage.removeItem("msm_token");
              setUser(null);
            }}
          >
            退出
          </button>
        </div>
      </aside>
      <main className="workspace">
        <div className="content">
          <header className="pageHero">
            <div className="pageIcon"><NavGlyph name={nav.find((item) => item[0] === route)?.[2] || "home"} /></div>
            <div>
              <h1>{title}</h1>
              <p>{route === "dashboard" ? "系统概览 · 实时监控" : "服务配置 · 状态管理"}</p>
            </div>
            <div className="pageActions">
              <StatusPills />
              <button onClick={() => window.location.reload()}>刷新</button>
            </div>
          </header>
          {route === "dashboard" && <Dashboard />}
          {route === "mosdns" && <MosDNSPage />}
          {route === "proxy" && <MihomoPage />}
          {route === "mihomo" && <MihomoPage />}
          {route === "process" && <ProcessPage />}
          {route === "config" && <ConfigPage />}
          {route === "logs" && <LogsPage />}
          {route === "users" && <UsersPage />}
          {route === "system" && <SystemPage />}
          {route === "settings" && <SettingsPage />}
          {route === "updates" && <UpdatesPage />}
        </div>
      </main>
    </div>
  );
}

function Splash() {
  return <div className="centerPanel">正在加载 msm-free</div>;
}

function Login({ onLogin }) {
  const [form, setForm] = useState({ username: "root", password: "" });
  const [error, setError] = useState("");
  const submit = async (e) => {
    e.preventDefault();
    setError("");
    try {
      const res = await api("/api/v1/auth/login", { method: "POST", body: JSON.stringify(form) });
      localStorage.setItem("msm_token", res.token);
      onLogin(res.user);
    } catch (err) {
      setError(err.message);
    }
  };
  return (
    <div className="loginShell">
      <section className="loginHero">
        <div className="heroBrand">
          <div className="brandMark large">MS</div>
          <div>
            <strong>MSM Free</strong>
            <span>0.1.0</span>
          </div>
        </div>
        <div className="heroCopy">
          <h1>MSM 管理平台</h1>
          <p>统一管理您的网络服务，提供 DNS 分流、代理管理、透明代理和配置审计能力。</p>
        </div>
        <div className="featureTiles">
          <div><b>DNS 服务</b><span>MosDNS 分流与审计</span></div>
          <div><b>代理管理</b><span>Mihomo 控制面板</span></div>
          <div><b>网络优化</b><span>nftables 透明代理</span></div>
        </div>
      </section>
      <section className="loginPanel">
        <form className="authBox" onSubmit={submit}>
          <p className="formEyebrow">欢迎使用 MSM</p>
          <h2>登录控制台</h2>
          <label>用户名<input placeholder="请输入用户名" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} /></label>
          <label>密码<input placeholder="请输入密码" type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} /></label>
          {error && <div className="error">{error}</div>}
          <button className="primary loginButton">登录</button>
          <div className="formHint">请使用初始化时创建的账号登录</div>
        </form>
      </section>
    </div>
  );
}

function SetupWizard({ onDone }) {
  const [step, setStep] = useState(0);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [sys, setSys] = useState(null);
  const [ifaces, setIfaces] = useState([]);
  const [form, setForm] = useState({
    username: "root",
    password: "",
    confirmPassword: "",
    email: "",
    webPort: "7777",
    selected_interface: "",
    amd64v3_enabled: false,
    mihomo_core_type: "meta",
    auto_set_dns: true,
    dns_on: "127.0.0.1",
    dns_off: "223.5.5.5",
    enableIPv6: true,
    fakeIPRangeV4: "28.0.0.0/8",
    fakeIPRangeV6: "f2b0::/18",
    linux_proxy_mode: "nft",
    nft_proxy_policy: "direct_default",
    proxyCore: "mihomo",
    mosdnsEnabled: true,
    subscription_urls: ""
  });

  useEffect(() => {
    Promise.all([api("/api/v1/setup/system-info"), api("/api/v1/setup/network-interfaces")]).then(([s, n]) => {
      setSys(s);
      const list = n.interfaces || [];
      setIfaces(list);
      const first = list.find((item) => item.is_up && !item.is_loopback);
      if (first) setForm((f) => ({ ...f, selected_interface: first.name }));
    });
  }, []);

  const steps = ["欢迎", "管理员", "系统", "组件", "启动", "完成"];
  const download = async (name) => {
    setBusy(true);
    setMessage(`正在安装 ${name}`);
    try {
      await api(`/api/v1/component-updates/${name}/update`, { method: "POST" });
      setMessage(`${name} 已处理`);
    } catch (e) {
      setMessage(e.message);
    } finally {
      setBusy(false);
    }
  };
  const initialize = async () => {
    if (form.password !== form.confirmPassword) {
      setMessage("两次密码不一致");
      return;
    }
    setBusy(true);
    try {
      await api("/api/v1/setup/initialize", { method: "POST", body: JSON.stringify(form) });
      setStep(4);
    } catch (e) {
      setMessage(e.message);
    } finally {
      setBusy(false);
    }
  };
  const startServices = async () => {
    setBusy(true);
    setMessage("");
    await api("/api/v1/services/mosdns/start", { method: "POST" }).catch((e) => setMessage(e.message));
    await api("/api/v1/services/mihomo/start", { method: "POST" }).catch((e) => setMessage(e.message));
    setBusy(false);
    setStep(5);
  };

  return (
    <div className="setupPage">
      <section className="setupFrame">
        <aside className="setupAside">
          <div className="brandMark large">MS</div>
          <h1>MSM Free 初始化向导</h1>
          <p>完成权限检查、管理员账号、网卡、DNS、代理核心、组件下载和服务启动。</p>
          <div className="setupMeta">
            <span>系统 {sys?.system?.os || "-"}/{sys?.system?.arch || "-"}</span>
            <span>CPU {sys?.cpu?.cores || "-"} 核</span>
            <span>AMD64 v3 {sys?.cpu?.amd64v3_status || "检测中"}</span>
          </div>
        </aside>
        <div className="setupMain">
        <div className="setupSteps">{steps.map((s, i) => <span key={s} className={i === step ? "active" : i < step ? "done" : ""}>{s}</span>)}</div>
        {step === 0 && <div className="setupBody"><h2>欢迎使用</h2><p>向导会生成 MosDNS、Mihomo 与 nftables 模板，并创建首个管理员账户。</p><button className="primary" onClick={() => setStep(1)}>开始初始化</button></div>}
        {step === 1 && <div className="setupBody grid2">
          <label>管理员用户名<input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} /></label>
          <label>邮箱<input value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} /></label>
          <label>密码<input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} /></label>
          <label>确认密码<input type="password" value={form.confirmPassword} onChange={(e) => setForm({ ...form, confirmPassword: e.target.value })} /></label>
          <div className="actions"><button onClick={() => setStep(0)}>返回</button><button className="primary" onClick={() => setStep(2)}>继续</button></div>
        </div>}
        {step === 2 && <div className="setupBody grid2">
          <label>Web 端口<input value={form.webPort} onChange={(e) => setForm({ ...form, webPort: e.target.value })} /></label>
          <label>物理网卡<select value={form.selected_interface} onChange={(e) => setForm({ ...form, selected_interface: e.target.value })}>{ifaces.map((i) => <option key={i.name}>{i.name}</option>)}</select></label>
          <label>DNS 开启值<input value={form.dns_on} onChange={(e) => setForm({ ...form, dns_on: e.target.value })} /></label>
          <label>DNS 关闭值<input value={form.dns_off} onChange={(e) => setForm({ ...form, dns_off: e.target.value })} /></label>
          <label>Fake-IP IPv4<input value={form.fakeIPRangeV4} onChange={(e) => setForm({ ...form, fakeIPRangeV4: e.target.value })} /></label>
          <label>代理策略<select value={form.nft_proxy_policy} onChange={(e) => setForm({ ...form, nft_proxy_policy: e.target.value })}><option value="direct_default">direct_default</option><option value="proxy_default">proxy_default</option></select></label>
          <label className="check"><input type="checkbox" checked={form.auto_set_dns} onChange={(e) => setForm({ ...form, auto_set_dns: e.target.checked })} />自动设置系统 DNS</label>
          <label className="check"><input type="checkbox" checked={form.enableIPv6} onChange={(e) => setForm({ ...form, enableIPv6: e.target.checked })} />启用 IPv6</label>
          <div className="actions"><button onClick={() => setStep(1)}>返回</button><button className="primary" onClick={() => setStep(3)}>继续</button></div>
        </div>}
        {step === 3 && <div className="setupBody">
          <label>订阅地址，支持 tag|url 或纯 URL<textarea rows="5" value={form.subscription_urls} onChange={(e) => setForm({ ...form, subscription_urls: e.target.value })} /></label>
          <div className="buttonRow"><button disabled={busy} onClick={() => download("mosdns")}>安装 MosDNS</button><button disabled={busy} onClick={() => download("mihomo")}>安装 Mihomo</button><button disabled={busy} onClick={() => download("zashboard")}>安装 zashboard</button></div>
          <div className="actions"><button onClick={() => setStep(2)}>返回</button><button className="primary" disabled={busy} onClick={initialize}>写入配置</button></div>
        </div>}
        {step === 4 && <div className="setupBody"><h2>启动服务</h2><p>配置已生成，可以启动 MosDNS 与 Mihomo。Linux 下 53 端口和 nftables 透明代理需要 root 权限。</p><button className="primary" disabled={busy} onClick={startServices}>启动服务</button></div>}
        {step === 5 && <div className="setupBody"><h2>初始化完成</h2><p>Mihomo、MosDNS、配置和用户体系已就绪。</p><button className="primary" onClick={onDone}>进入 WebUI</button></div>}
        {message && <div className="notice">{message}</div>}
        </div>
      </section>
    </div>
  );
}

function StatusPills() {
  const { data } = usePoll(() => api("/api/v1/services"), [], 4000);
  const services = data?.data || [];
  return <div className="pills">{services.map((s) => <span key={s.name} className={s.running ? "ok" : "off"}>{s.display_name}: {s.status}</span>)}</div>;
}

function Dashboard() {
  const system = usePoll(() => api("/api/v1/monitor/system"), [], 3000).data?.data || {};
  const resources = usePoll(() => api("/api/v1/monitor/resources"), [], 3000).data?.data || {};
  const servicesPoll = usePoll(() => api("/api/v1/services"), [], 3000);
  const services = servicesPoll.data?.data || [];
  const mosdns = services.find((s) => s.name === "mosdns") || {};
  const mihomo = services.find((s) => s.name === "mihomo") || {};
  return <section className="stack">
    <div className="dashboardGrid">
      <InfoPanel title="设备信息" icon="◆" rows={[
        ["主机名", system.hostname || "-"],
        ["系统平台", system.os || "-"],
        ["本机地址", (system.local_ips || []).join(", ") || "-"],
        ["操作系统", system.os || "-"],
        ["架构", system.arch || "-"]
      ]} />
      <InfoPanel title="硬件信息" icon="◈" rows={[
        ["硬件信息", system.arch || "-"],
        ["核心数", `${resources.goroutines ?? 0} goroutines`],
        ["内存", formatBytes(resources.memory_total || 0)],
        ["已使用", formatBytes(resources.memory_used || 0)],
        ["内存使用率", `${Math.round(resources.memory_percent ?? 0)}%`]
      ]} />
      <RatePanel />
      <InfoPanel title="统计信息" icon="▣" rows={[
        ["系统运行时间", "-"],
        ["总上传流量", "0 B"],
        ["总下载流量", "0 B"],
        ["CPU 使用率", `${Math.round(resources.cpu_percent ?? 0)}%`],
        ["内存使用率", `${Math.round(resources.memory_percent ?? 0)}%`]
      ]} />
      <TrendPanel cpu={resources.cpu_percent ?? 0} mem={resources.memory_percent ?? 0} />
      <DashboardService service={mosdns} title="MosDNS" reload={servicesPoll.reload} />
      <EmptyService title="Sing-Box" />
      <DashboardService service={mihomo} title="Mihomo" reload={servicesPoll.reload} />
    </div>
  </section>;
}

function InfoPanel({ title, icon, rows }) {
  return <article className="dashCard">
    <div className="dashCardTitle"><span>{icon}</span><h3>{title}</h3></div>
    <div className="infoRows">
      {rows.map(([k, v]) => <div key={k}><span>{k}</span><strong>{v}</strong></div>)}
    </div>
  </article>;
}

function RatePanel() {
  return <article className="dashCard chartCard">
    <div className="dashCardTitle"><span>↕</span><h3>实时速率</h3></div>
    <div className="rateNumbers">
      <div><span>连接数</span><strong>0</strong></div>
      <div><span>下载</span><strong>0 B/s</strong></div>
      <div><span>上传</span><strong>0 B/s</strong></div>
    </div>
    <MiniChart />
    <div className="chartControls"><button>连接数</button><button>上传速度</button><button>下载速度</button><button>全部</button></div>
  </article>;
}

function TrendPanel({ cpu, mem }) {
  return <article className="dashCard chartCard">
    <div className="dashCardTitle"><span>▥</span><h3>资源使用趋势</h3></div>
    <div className="rateNumbers">
      <div><span>CPU</span><strong>{Math.round(cpu)}%</strong></div>
      <div><span>内存</span><strong>{Math.round(mem)}%</strong></div>
    </div>
    <MiniChart />
    <div className="chartControls"><button>CPU</button><button>内存</button><button>全部</button><button>75%</button></div>
  </article>;
}

function MiniChart() {
  return <div className="miniChart">
    {Array.from({ length: 24 }).map((_, i) => <i key={i} style={{ height: `${18 + ((i * 17) % 52)}%` }} />)}
  </div>;
}

function DashboardService({ service, title, reload }) {
  const status = service?.running ? "运行中" : service?.installed ? "已停止" : "未安装";
  const action = (name) => service?.name && api(`/api/v1/services/${service.name}/${name}`, { method: "POST" }).then(() => reload?.());
  return <article className="dashCard">
    <div className="dashCardTitle"><span>●</span><h3>{title}</h3></div>
    <div className="infoRows">
      <div><span>状态</span><strong className={service?.running ? "greenText" : ""}>{status}</strong></div>
      <div><span>CPU</span><strong>0.0%</strong></div>
      <div><span>内存</span><strong>-</strong></div>
      <div><span>运行时间</span><strong>-</strong></div>
    </div>
    <div className="cardActions"><button onClick={() => action("stop")}>停止</button><button onClick={() => action("restart")}>重启</button></div>
  </article>;
}

function EmptyService({ title }) {
  return <article className="dashCard emptyService">
    <div className="dashCardTitle"><span>○</span><h3>{title}</h3></div>
    <div className="emptyBody"><strong>服务未配置</strong><span>该服务尚未在系统中配置</span></div>
    <p>请前往设置页面配置 {title}</p>
  </article>;
}

function NavGlyph({ name }) {
  const common = { width: 18, height: 18, viewBox: "0 0 24 24", fill: "none", stroke: "currentColor", strokeWidth: 2, strokeLinecap: "round", strokeLinejoin: "round", "aria-hidden": true };
  const paths = {
    home: <><path d="M3 10.8 12 3l9 7.8" /><path d="M5 10v10h14V10" /><path d="M9 20v-6h6v6" /></>,
    dns: <><rect x="4" y="4" width="16" height="6" rx="2" /><rect x="4" y="14" width="16" height="6" rx="2" /><path d="M8 7h.01M8 17h.01M12 10v4" /></>,
    route: <><path d="M6 19a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z" /><path d="M18 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z" /><path d="M8.5 14.5 15.5 9.5" /></>,
    cube: <><path d="m12 3 8 4.5v9L12 21l-8-4.5v-9L12 3Z" /><path d="M12 12 4.4 7.7M12 12l7.6-4.3M12 12v9" /></>,
    process: <><rect x="4" y="5" width="16" height="14" rx="2" /><path d="M8 9h8M8 13h5M8 17h3" /></>,
    file: <><path d="M6 3h8l4 4v14H6z" /><path d="M14 3v5h5M9 13h6M9 17h6" /></>,
    logs: <><path d="M5 4h14v16H5z" /><path d="M8 8h8M8 12h8M8 16h5" /></>,
    users: <><path d="M16 21v-2a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4v2" /><circle cx="9.5" cy="7" r="4" /><path d="M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" /></>,
    diagnostic: <><path d="M12 3 4 6v6c0 5 3.4 8 8 9 4.6-1 8-4 8-9V6l-8-3Z" /><path d="m9 12 2 2 4-5" /></>,
    settings: <><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.88l.04.04a2 2 0 1 1-2.83 2.83l-.04-.04A1.7 1.7 0 0 0 15 19.4a1.7 1.7 0 0 0-1 .6 1.7 1.7 0 0 0-.4 1.1V21a2 2 0 1 1-4 0v-.06a1.7 1.7 0 0 0-.4-1.1 1.7 1.7 0 0 0-1-.6 1.7 1.7 0 0 0-1.88.34l-.04.04a2 2 0 1 1-2.83-2.83l.04-.04A1.7 1.7 0 0 0 4.6 15a1.7 1.7 0 0 0-.6-1 1.7 1.7 0 0 0-1.1-.4H3a2 2 0 1 1 0-4h.06a1.7 1.7 0 0 0 1.1-.4 1.7 1.7 0 0 0 .6-1 1.7 1.7 0 0 0-.34-1.88l-.04-.04a2 2 0 1 1 2.83-2.83l.04.04A1.7 1.7 0 0 0 9 4.6a1.7 1.7 0 0 0 1-.6 1.7 1.7 0 0 0 .4-1.1V3a2 2 0 1 1 4 0v.06a1.7 1.7 0 0 0 .4 1.1 1.7 1.7 0 0 0 1 .6 1.7 1.7 0 0 0 1.88-.34l.04-.04a2 2 0 1 1 2.83 2.83l-.04.04A1.7 1.7 0 0 0 19.4 9c.2.36.4.7.6 1 .3.24.7.4 1.1.4H21a2 2 0 1 1 0 4h-.06a1.7 1.7 0 0 0-1.1.4c-.3.3-.44.56-.44.2Z" /></>
  };
  return <svg {...common}>{paths[name] || paths.home}</svg>;
}

function Metric({ title, value, caption, compact }) {
  return <article className={compact ? "card metric compact" : "card metric"}><span>{title}</span><strong>{value}</strong>{caption && <p>{caption}</p>}</article>;
}

function ServiceCard({ service, reload }) {
  const action = (name) => api(`/api/v1/services/${service.name}/${name}`, { method: "POST" }).then(() => reload?.());
  return <article className="card serviceCard">
    <div className="serviceHead">
      <div>
        <span>{service.display_name}</span>
        <strong>{service.running ? "运行中" : "已停止"}</strong>
      </div>
      <StatusBadge ok={service.running}>{service.status}</StatusBadge>
    </div>
    <p>{service.installed ? service.binary_path : "未安装组件"}</p>
    <div className="buttonRow"><button onClick={() => action("start")}>启动</button><button onClick={() => action("stop")}>停止</button><button onClick={() => action("restart")}>重启</button></div>
  </article>;
}

function ProcessPage() {
  const poll = usePoll(() => api("/api/v1/services"), [], 3000);
  return <section className="grid2">{(poll.data?.data || []).map((s) => <ServiceCard key={s.name} service={s} reload={poll.reload} />)}</section>;
}

function MosDNSPage() {
  const overview = usePoll(() => api("/api/v1/mosdns/overview"), [], 3000);
  const clients = usePoll(() => api("/api/v1/mosdns/clients"), [], 5000);
  const [switches, setSwitches] = useState({});
  useEffect(() => { api("/api/v1/mosdns/switches").then((r) => setSwitches(r.data || {})); }, []);
  const saveSwitches = () => api("/api/v1/mosdns/switches", { method: "PUT", body: JSON.stringify(switches) });
  return <section className="stack">
    <div className="overviewStrip"><Metric title="服务状态" value={overview.data?.data?.running ? "运行中" : "已停止"} /><Metric title="客户端" value={overview.data?.data?.clients ?? 0} /><Metric title="DNS 监听" value=":53" /><Metric title="API" value=":9099" /></div>
    <article className="panel"><PanelHeader title="开关管理" desc="控制 MosDNS 分流、审计、规则特性" /><div className="switchGrid">{Object.keys(switches).map((k) => <label key={k} className="check switchItem"><input type="checkbox" checked={!!switches[k]} onChange={(e) => setSwitches({ ...switches, [k]: e.target.checked })} />{k}</label>)}</div><button className="primary" onClick={saveSwitches}>保存开关</button></article>
    <article className="panel"><PanelHeader title="客户端" desc="局域网扫描与代理设备列表" action={<button onClick={() => api("/api/v1/mosdns/clients/scan", { method: "POST" }).then(clients.reload)}>扫描局域网</button>} /><DataTable rows={clients.data?.data || []} columns={["ip", "mac", "hostname", "interface", "is_online"]} /></article>
    <LogBox service="mosdns" />
  </section>;
}

function MihomoPage() {
  const overview = usePoll(() => api("/api/v1/mihomo/overview"), [], 3000);
  const proxies = usePoll(() => api("/api/v1/mihomo/proxies"), [], 5000);
  return <section className="stack">
    <div className="overviewStrip"><Metric title="服务状态" value={overview.data?.data?.service?.status || "-"} /><Metric title="版本" value={String(overview.data?.data?.version || "-").slice(0, 48)} /><Metric title="控制端口" value="9090" /><Metric title="核心" value="Mihomo" /></div>
    <article className="panel"><PanelHeader title="端口映射" desc="HTTP / SOCKS / Mixed / Redir / TProxy / Controller" /><DataTable rows={[overview.data?.data?.ports || {}]} columns={["http", "socks", "mixed", "redir", "tproxy", "controller"]} /></article>
    <article className="panel"><PanelHeader title="代理组" desc="来自 Mihomo external-controller 的实时数据" /><pre>{JSON.stringify(proxies.data?.data || proxies.data || {}, null, 2)}</pre></article>
    <ConfigEditor defaultPath="configs/mihomo/config.yaml" />
    <LogBox service="mihomo" />
  </section>;
}

function ConfigPage() {
  return <section className="stack"><ConfigEditor defaultPath="configs/mihomo/config.yaml" /></section>;
}

function ConfigEditor({ defaultPath }) {
  const [path, setPath] = useState(defaultPath);
  const [content, setContent] = useState("");
  const [message, setMessage] = useState("");
  const load = () => api(`/api/v1/config/file?path=${encodeURIComponent(path)}`).then((r) => setContent(r.content));
  useEffect(() => { load().catch((e) => setMessage(e.message)); }, [path]);
  const save = () => api("/api/v1/config/file", { method: "PUT", body: JSON.stringify({ path, content }) }).then(() => setMessage("已保存"));
  return <article className="panel editor"><PanelHeader title="配置编辑" desc="支持 Mihomo、MosDNS、network 模板文件" /><div className="row"><input value={path} onChange={(e) => setPath(e.target.value)} /><button onClick={load}>读取</button><button className="primary" onClick={save}>保存</button></div><textarea value={content} onChange={(e) => setContent(e.target.value)} spellCheck="false" />{message && <div className="notice inline">{message}</div>}</article>;
}

function LogsPage() {
  const [service, setService] = useState("mihomo");
  return <section className="stack"><div className="row"><select value={service} onChange={(e) => setService(e.target.value)}><option>mihomo</option><option>mosdns</option></select></div><LogBox service={service} /></section>;
}

function LogBox({ service, compact }) {
  const { data } = usePoll(() => api(`/api/v1/logs/${service}?lines=300`), [service], 3000);
  return <article className="panel"><PanelHeader title={`${service} 日志`} desc="自动刷新最近 300 行" /><pre className={compact ? "log compactLog" : "log"}>{data?.content || ""}</pre></article>;
}

function UsersPage() {
  const poll = usePoll(() => api("/api/v1/users"), [], 4000);
  const [form, setForm] = useState({ username: "", password: "", role: "operator" });
  const create = () => api("/api/v1/users", { method: "POST", body: JSON.stringify(form) }).then(() => { setForm({ username: "", password: "", role: "operator" }); poll.reload(); });
  return <section className="stack">
    <article className="panel"><PanelHeader title="新增用户" desc="管理员、操作员、访客三类权限" /><div className="row"><input placeholder="用户名" value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} /><input type="password" placeholder="密码" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} /><select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value })}><option value="operator">operator</option><option value="viewer">viewer</option><option value="admin">admin</option></select><button className="primary" onClick={create}>创建</button></div></article>
    <article className="panel"><PanelHeader title="用户列表" desc="登录审计和启用状态" /><DataTable rows={poll.data?.data || []} columns={["id", "username", "role", "is_active", "last_login"]} /></article>
    <TokensPanel />
  </section>;
}

function TokensPanel() {
  const poll = usePoll(() => api("/api/v1/api-tokens"), [], 5000);
  const [name, setName] = useState("automation");
  const [token, setToken] = useState("");
  const create = () => api("/api/v1/api-tokens", { method: "POST", body: JSON.stringify({ name }) }).then((r) => { setToken(r.token); poll.reload(); });
  return <article className="panel"><PanelHeader title="API Token" desc="用于脚本、自动化和后续 Unraid 插件调用" /><div className="row"><input value={name} onChange={(e) => setName(e.target.value)} /><button onClick={create}>生成</button></div>{token && <pre>{token}</pre>}<DataTable rows={poll.data?.data || []} columns={["id", "name", "created_at", "last_used_at", "revoked"]} /></article>;
}

function SettingsPage() {
  const [settings, setSettings] = useState({});
  useEffect(() => { api("/api/v1/settings").then((r) => setSettings(r.data || {})); }, []);
  const save = () => api("/api/v1/settings", { method: "PUT", body: JSON.stringify(settings) });
  return <section className="stack"><article className="panel editor"><PanelHeader title="系统设置" desc="全局键值配置，兼容 MSM 设置接口" /><textarea value={JSON.stringify(settings, null, 2)} onChange={(e) => { try { setSettings(JSON.parse(e.target.value)); } catch {} }} /><button className="primary" onClick={save}>保存</button></article></section>;
}

function SystemPage() {
  const diag = usePoll(() => api("/api/v1/system/diagnostics"), [], 5000);
  const net = usePoll(() => api("/api/v1/netlink/nftables/status"), [], 5000);
  return <section className="stack">
    <article className="panel"><PanelHeader title="系统诊断" desc="权限、核心组件、nftables 配置检查" /><DataTable rows={diag.data?.checks || []} columns={["name", "ok", "message"]} /></article>
    <article className="panel"><PanelHeader title="nftables 状态" desc="透明代理表和策略路由状态" /><pre>{JSON.stringify(net.data?.data || {}, null, 2)}</pre></article>
  </section>;
}

function UpdatesPage() {
  const poll = usePoll(() => api("/api/v1/component-updates"), [], 5000);
  const update = (c) => api(`/api/v1/component-updates/${c}/update`, { method: "POST" }).then(poll.reload);
  return <section className="stack"><article className="panel"><PanelHeader title="组件更新" desc="MosDNS、Mihomo、zashboard 下载与状态机" /><DataTable rows={poll.data?.data || []} columns={["component", "current_version", "latest_version", "has_update", "status", "progress"]} action={(row) => <button onClick={() => update(row.component)}>安装/更新</button>} /></article></section>;
}

function DataTable({ rows, columns, action }) {
  return <div className="tableWrap"><table><thead><tr>{columns.map((c) => <th key={c}>{c}</th>)}{action && <th>操作</th>}</tr></thead><tbody>{rows.map((row, i) => <tr key={row.id || i}>{columns.map((c) => <td key={c}>{String(row?.[c] ?? "")}</td>)}{action && <td>{action(row)}</td>}</tr>)}</tbody></table></div>;
}

function PanelHeader({ title, desc, action }) {
  return <div className="panelHeader"><div><h2>{title}</h2>{desc && <p>{desc}</p>}</div>{action}</div>;
}

function StatusBadge({ ok, children }) {
  return <span className={ok ? "badge ok" : "badge off"}>{children}</span>;
}

function formatBytes(v) {
  if (!v) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let n = v;
  let i = 0;
  while (n > 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

createRoot(document.getElementById("root")).render(<App />);
