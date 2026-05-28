import{j as e}from"./editor-vendor-Ca1lPq8W.js";import{r,d as D}from"./react-vendor-Bt7EHJHg.js";import{c as x,B as S,v as L,b as P,d as R}from"./index-CDz96t7w.js";import{C as E,a as M,c as I}from"./Card-uiOvXNOT.js";import{h as $}from"./crypto-DDRdfUrs.js";import{u as A}from"./utils-vendor-yfWiyOYv.js";import{x as H,r as _,N as U,U as V,a6 as W,a3 as q,a4 as z,a7 as B}from"./ui-vendor-DTdpf_eS.js";const Y=()=>{const{t}=A(),[n,j]=r.useState(""),[u,p]=r.useState(""),[d,w]=r.useState(!1),[i,c]=r.useState(!1),[f,m]=r.useState(""),[h,g]=r.useState(""),[v,o]=r.useState(!1),[N,k]=r.useState("unknown"),C=D();r.useEffect(()=>{(async()=>{try{const a=await L.getVersion();k(a.version||t("common.unknown"))}catch(a){console.error(t("login.loadVersionFailed"),a)}})()},[]);const O=async s=>{s.preventDefault(),c(!0),m(""),g("");try{const a=await $(u),l=await P.login({username:n,password:a});R(l.token,l.refresh_token,l.user),c(!1),p(""),C("/")}catch(a){c(!1);const l=a.message||"UNKNOWN_ERROR",b=`login.errors.${l}`,y=t(b);m(b===y?l==="UNKNOWN_ERROR"?t("login.loginFailed"):l:y),a.hint&&g(a.hint),(a?.status===403||/锁定|LOCKED|DISABLED/.test(l)||a.hint)&&o(!0)}};return e.jsxs("div",{className:"min-h-screen flex bg-gradient-to-br from-slate-50 via-blue-50/30 to-slate-100 dark:from-slate-950 dark:via-slate-900 dark:to-slate-950",children:[e.jsxs("div",{className:"hidden lg:flex lg:w-1/2 xl:w-3/5 flex-col justify-center items-center p-12 relative overflow-hidden",children:[e.jsxs("div",{className:"absolute inset-0 opacity-20",children:[e.jsx("div",{className:"absolute top-1/4 left-1/4 w-96 h-96 bg-primary/40 rounded-full blur-3xl animate-pulse-subtle"}),e.jsx("div",{className:"absolute bottom-1/4 right-1/4 w-80 h-80 bg-blue-400/30 rounded-full blur-3xl animate-pulse-subtle",style:{animationDelay:"1s"}})]}),e.jsxs("div",{className:"relative z-10 text-center space-y-8 max-w-lg",children:[e.jsx("div",{className:"flex justify-center",children:e.jsxs("div",{className:"relative group",children:[e.jsxs("div",{className:"absolute inset-0 -m-24",children:[e.jsx("div",{className:"absolute top-0 left-1/2 -translate-x-1/2",children:[...Array(5)].map((s,a)=>e.jsx("div",{className:x("absolute w-2.5 h-2.5 rounded-full bg-gradient-to-br from-primary to-purple-500 shadow-lg shadow-primary/50",a>=3&&"hidden lg:block"),style:{animation:"flowIn var(--flow-duration, 4s) ease-in-out infinite",animationDelay:`${a*.8}s`,opacity:0,willChange:"transform, opacity"}},`in-${a}`))}),e.jsx("div",{className:"absolute top-1/2 left-0 -translate-y-1/2",children:[...Array(5)].map((s,a)=>e.jsx("div",{className:x("absolute w-2.5 h-2.5 rounded-full bg-gradient-to-br from-green-400 to-emerald-500 shadow-lg shadow-green-500/50",a>=3&&"hidden lg:block"),style:{animation:"flowOutLeft var(--flow-duration, 4s) ease-in-out infinite",animationDelay:`${a*.8+.4}s`,opacity:0,willChange:"transform, opacity"}},`out-left-${a}`))}),e.jsx("div",{className:"absolute top-1/2 right-0 -translate-y-1/2",children:[...Array(5)].map((s,a)=>e.jsx("div",{className:x("absolute w-2.5 h-2.5 rounded-full bg-gradient-to-br from-blue-400 to-cyan-500 shadow-lg shadow-blue-500/50",a>=3&&"hidden lg:block"),style:{animation:"flowOutRight var(--flow-duration, 4s) ease-in-out infinite",animationDelay:`${a*.8+.4}s`,opacity:0,willChange:"transform, opacity"}},`out-right-${a}`))}),e.jsxs("svg",{className:"absolute inset-0 w-full h-full pointer-events-none opacity-40 lg:opacity-40",children:[e.jsxs("defs",{children:[e.jsxs("linearGradient",{id:"gradientIn",x1:"0%",y1:"0%",x2:"0%",y2:"100%",children:[e.jsx("stop",{offset:"0%",stopColor:"currentColor",stopOpacity:"0"}),e.jsx("stop",{offset:"50%",stopColor:"currentColor",stopOpacity:"1"}),e.jsx("stop",{offset:"100%",stopColor:"currentColor",stopOpacity:"0"})]}),e.jsxs("linearGradient",{id:"gradientOutLeft",x1:"100%",y1:"0%",x2:"0%",y2:"0%",children:[e.jsx("stop",{offset:"0%",stopColor:"currentColor",stopOpacity:"0"}),e.jsx("stop",{offset:"50%",stopColor:"currentColor",stopOpacity:"1"}),e.jsx("stop",{offset:"100%",stopColor:"currentColor",stopOpacity:"0"})]}),e.jsxs("linearGradient",{id:"gradientOutRight",x1:"0%",y1:"0%",x2:"100%",y2:"0%",children:[e.jsx("stop",{offset:"0%",stopColor:"currentColor",stopOpacity:"0"}),e.jsx("stop",{offset:"50%",stopColor:"currentColor",stopOpacity:"1"}),e.jsx("stop",{offset:"100%",stopColor:"currentColor",stopOpacity:"0"})]})]}),e.jsx("path",{d:"M 50% 0 L 50% 50%",stroke:"url(#gradientIn)",strokeWidth:"2",fill:"none",strokeDasharray:"8,4",className:"text-primary",children:e.jsx("animate",{attributeName:"stroke-dashoffset",from:"0",to:"12",dur:"1.5s",repeatCount:"indefinite"})}),e.jsx("path",{d:"M 50% 50% L 0 50%",stroke:"url(#gradientOutLeft)",strokeWidth:"2",fill:"none",strokeDasharray:"8,4",className:"text-green-500",children:e.jsx("animate",{attributeName:"stroke-dashoffset",from:"0",to:"12",dur:"1.5s",repeatCount:"indefinite"})}),e.jsx("path",{d:"M 50% 50% L 100% 50%",stroke:"url(#gradientOutRight)",strokeWidth:"2",fill:"none",strokeDasharray:"8,4",className:"text-blue-500",children:e.jsx("animate",{attributeName:"stroke-dashoffset",from:"0",to:"12",dur:"1.5s",repeatCount:"indefinite"})})]}),e.jsx("div",{className:"hidden lg:block absolute top-0 left-1/2 -translate-x-1/2 w-4 h-4 rounded-full bg-primary/30 blur-md animate-pulse",style:{animationDuration:"2s"}}),e.jsx("div",{className:"hidden lg:block absolute top-1/2 left-0 -translate-y-1/2 w-4 h-4 rounded-full bg-green-500/30 blur-md animate-pulse",style:{animationDuration:"2s",animationDelay:"0.5s"}}),e.jsx("div",{className:"hidden lg:block absolute top-1/2 right-0 -translate-y-1/2 w-4 h-4 rounded-full bg-blue-500/30 blur-md animate-pulse",style:{animationDuration:"2s",animationDelay:"1s"}})]}),e.jsx("div",{className:"absolute inset-0 -m-4",children:e.jsx("div",{className:"w-full h-full border-[3px] border-primary/30 rounded-full animate-spin",style:{animationDuration:"var(--spin-duration-1, 10s)",willChange:"transform"}})}),e.jsx("div",{className:"absolute inset-0 -m-6",children:e.jsx("div",{className:"w-full h-full border-[3px] border-blue-400/20 rounded-full animate-spin",style:{animationDuration:"var(--spin-duration-2, 15s)",animationDirection:"reverse",willChange:"transform"}})}),e.jsx("div",{className:"hidden lg:block absolute inset-0 -m-8",children:e.jsx("div",{className:"w-full h-full border-[2px] border-purple-400/15 rounded-full animate-spin",style:{animationDuration:"var(--spin-duration-3, 20s)",willChange:"transform"}})}),e.jsxs("div",{className:"relative",children:[e.jsx("div",{className:"absolute inset-0 bg-primary/20 blur-3xl rounded-full animate-pulse-subtle"}),e.jsx("div",{className:"hidden lg:block absolute inset-0 bg-blue-400/10 blur-2xl rounded-full animate-pulse-subtle",style:{animationDelay:"1s"}}),e.jsxs("div",{className:"relative bg-white dark:bg-slate-800 p-6 lg:p-8 rounded-3xl shadow-2xl group-hover:scale-110 transition-transform duration-500 animate-float-tilt",children:[e.jsx("div",{className:"pointer-events-none absolute inset-0 rounded-3xl bg-gradient-to-br from-primary/20 via-transparent to-blue-400/20 animate-rotate-slow"}),e.jsx("img",{src:"/logo/logo-square.svg",alt:"MSM",className:"h-16 w-16 lg:h-24 lg:w-24 relative z-10 drop-shadow-2xl"}),e.jsxs("div",{className:"absolute inset-0 flex items-center justify-center",children:[e.jsx("div",{className:"w-3 h-3 lg:w-4 lg:h-4 rounded-full bg-primary/40 animate-ping",style:{animationDuration:"2s"}}),e.jsx("div",{className:"hidden lg:block absolute w-3 h-3 rounded-full bg-primary/60 animate-ping",style:{animationDuration:"2s",animationDelay:"0.5s"}}),e.jsx("div",{className:"absolute w-2 h-2 rounded-full bg-primary animate-pulse"})]})]})]})]})}),e.jsx("style",{children:`
            /* 统一桌面端和移动端动画速度 */
            :root {
              --spin-duration-1: 10s;
              --spin-duration-2: 15s;
              --spin-duration-3: 20s;
              --flow-duration: 4s;
            }

            @keyframes flowIn {
              0% {
                transform: translate(-50%, -100px) scale(0.8);
                opacity: 0;
              }
              15% {
                opacity: 1;
                transform: translate(-50%, -50px) scale(1);
              }
              50% {
                transform: translate(-50%, 0) scale(1.1);
                opacity: 1;
              }
              55% {
                opacity: 0;
                transform: translate(-50%, 0) scale(0.9);
              }
              100% {
                transform: translate(-50%, 0) scale(0.8);
                opacity: 0;
              }
            }

            @keyframes flowOutLeft {
              0%, 50% {
                transform: translate(0, -50%) scale(0.8);
                opacity: 0;
              }
              55% {
                transform: translate(-10px, -50%) scale(1);
                opacity: 1;
              }
              100% {
                transform: translate(-100px, -50%) scale(0.8);
                opacity: 0;
              }
            }

            @keyframes flowOutRight {
              0%, 50% {
                transform: translate(0, -50%) scale(0.8);
                opacity: 0;
              }
              55% {
                transform: translate(10px, -50%) scale(1);
                opacity: 1;
              }
              100% {
                transform: translate(100px, -50%) scale(0.8);
                opacity: 0;
              }
            }

            /* 移动端性能优化 - 简化动画 */
            @media (max-width: 1024px) {
              @keyframes flowIn {
                0% {
                  transform: translate(-50%, -80px);
                  opacity: 0;
                }
                20% {
                  opacity: 1;
                }
                50% {
                  transform: translate(-50%, 0);
                  opacity: 1;
                }
                55% {
                  opacity: 0;
                }
                100% {
                  opacity: 0;
                }
              }

              @keyframes flowOutLeft {
                0%, 50% {
                  transform: translate(0, -50%);
                  opacity: 0;
                }
                55% {
                  opacity: 1;
                }
                100% {
                  transform: translate(-80px, -50%);
                  opacity: 0;
                }
              }

              @keyframes flowOutRight {
                0%, 50% {
                  transform: translate(0, -50%);
                  opacity: 0;
                }
                55% {
                  opacity: 1;
                }
                100% {
                  transform: translate(80px, -50%);
                  opacity: 0;
                }
              }
            }

            /* 尊重用户的动画偏好 */
            @media (prefers-reduced-motion: reduce) {
              * {
                animation-duration: 0.01ms !important;
                animation-iteration-count: 1 !important;
                transition-duration: 0.01ms !important;
              }
            }
          `}),e.jsxs("div",{className:"space-y-4 animate-fade-in",children:[e.jsx("h1",{className:"text-4xl xl:text-5xl font-bold text-slate-800 dark:text-slate-100",children:t("login.branding.title")}),e.jsx("p",{className:"text-sm text-slate-500 dark:text-slate-500 max-w-md mx-auto",children:t("login.branding.subtitle")})]}),e.jsxs("div",{className:"flex justify-center gap-8 pt-8 opacity-60",children:[e.jsxs("div",{className:"flex flex-col items-center gap-2 animate-slide-up animate-delay-100",children:[e.jsx("div",{className:"w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center",children:e.jsx(H,{className:"h-6 w-6 text-primary"})}),e.jsx("span",{className:"text-xs text-slate-600 dark:text-slate-400",children:t("login.branding.tags.dns")})]}),e.jsxs("div",{className:"flex flex-col items-center gap-2 animate-slide-up animate-delay-200",children:[e.jsx("div",{className:"w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center",children:e.jsx(_,{className:"h-6 w-6 text-primary"})}),e.jsx("span",{className:"text-xs text-slate-600 dark:text-slate-400",children:t("login.branding.tags.proxy")})]}),e.jsxs("div",{className:"flex flex-col items-center gap-2 animate-slide-up animate-delay-300",children:[e.jsx("div",{className:"w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center",children:e.jsx(U,{className:"h-6 w-6 text-primary"})}),e.jsx("span",{className:"text-xs text-slate-600 dark:text-slate-400",children:t("login.branding.tags.network")})]})]})]}),e.jsx("div",{className:"absolute bottom-8 text-center text-xs text-slate-500 dark:text-slate-600",children:e.jsxs("p",{children:["MSM ",N]})})]}),e.jsx("div",{className:"flex-1 flex items-center justify-center p-6 lg:p-12",children:e.jsxs(E,{className:"w-full max-w-md animate-scale-in shadow-2xl border-0 bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-3xl",children:[e.jsxs(M,{className:"pb-6 pt-8 px-8",children:[e.jsx("div",{className:"lg:hidden flex justify-center mb-6",children:e.jsxs("div",{className:"relative animate-float-tilt",children:[e.jsx("div",{className:"absolute inset-0 bg-primary/30 blur-2xl rounded-full animate-pulse-subtle"}),e.jsx("div",{className:"pointer-events-none absolute inset-0 rounded-2xl bg-gradient-to-br from-primary/15 via-transparent to-blue-400/15 animate-rotate-slow"}),e.jsx("img",{src:"/logo/logo-square.svg",alt:"MSM",className:"h-16 w-16 relative z-10 drop-shadow-2xl"})]})}),e.jsx("div",{className:"text-left",children:e.jsx("p",{className:"text-base font-medium text-slate-600 dark:text-slate-400",children:t("login.welcome")})})]}),e.jsxs(I,{className:"px-8 pb-8",children:[e.jsxs("form",{onSubmit:O,className:"space-y-5",children:[f&&e.jsx("div",{className:"p-3 rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800/50 text-red-600 dark:text-red-400 text-sm animate-fade-in",children:f}),e.jsxs("div",{className:"space-y-2",children:[e.jsx("label",{className:"text-sm font-medium text-slate-700 dark:text-slate-300",children:t("login.username")}),e.jsxs("div",{className:"relative group",children:[e.jsx("div",{className:"absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 group-focus-within:text-primary transition-colors",children:e.jsx(V,{className:"h-5 w-5"})}),e.jsx("input",{type:"text",value:n,onChange:s=>j(s.target.value),className:"w-full pl-11 pr-4 py-3 rounded-xl border-2 border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-slate-800 dark:text-slate-100 text-sm placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all",placeholder:t("login.usernamePlaceholder"),required:!0,disabled:i})]})]}),e.jsxs("div",{className:"space-y-2",children:[e.jsx("label",{className:"text-sm font-medium text-slate-700 dark:text-slate-300",children:t("login.password")}),e.jsxs("div",{className:"relative group",children:[e.jsx("div",{className:"absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 group-focus-within:text-primary transition-colors",children:e.jsx(W,{className:"h-5 w-5"})}),e.jsx("input",{type:d?"text":"password",value:u,onChange:s=>p(s.target.value),className:"w-full pl-11 pr-11 py-3 rounded-xl border-2 border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 text-slate-800 dark:text-slate-100 text-sm placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all",placeholder:t("login.passwordPlaceholder"),required:!0,disabled:i}),e.jsx("button",{type:"button",onClick:()=>w(s=>!s),className:"absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors","aria-label":t(d?"login.hidePassword":"login.showPassword"),children:d?e.jsx(q,{className:"h-5 w-5"}):e.jsx(z,{className:"h-5 w-5"})})]})]}),e.jsx(S,{type:"submit",className:"w-full h-12 bg-gradient-to-r from-primary via-blue-500 to-primary hover:shadow-xl text-white transition-all hover:scale-[1.02] disabled:hover:scale-100 rounded-xl font-medium text-base mt-6",disabled:i,children:i?e.jsxs("span",{className:"flex items-center gap-2",children:[e.jsx("div",{className:"w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin"}),e.jsx("span",{children:t("login.loggingIn")})]}):e.jsxs("span",{className:"flex items-center gap-2",children:[e.jsx(B,{className:"h-5 w-5"}),t("login.loginButton")]})})]}),e.jsxs("div",{className:"mt-6 text-center space-y-2",children:[e.jsx("p",{className:"text-xs text-slate-500 dark:text-slate-500",children:t("login.loginHint")}),e.jsx("button",{type:"button",className:"text-sm text-primary hover:text-primary/80 transition-colors font-medium",onClick:()=>o(!0),children:t("login.forgotPassword")})]})]})]})}),v&&e.jsx("div",{className:"fixed inset-0 z-50 bg-black/50 backdrop-blur-sm flex items-center justify-center p-4 animate-fade-in",onClick:()=>o(!1),children:e.jsxs("div",{className:"bg-white dark:bg-slate-900 rounded-2xl shadow-2xl w-[90%] max-w-lg p-6 border border-slate-200 dark:border-slate-800 animate-scale-in",onClick:s=>s.stopPropagation(),children:[e.jsxs("div",{className:"flex items-center justify-between mb-4",children:[e.jsx("h3",{className:"text-lg font-semibold text-slate-800 dark:text-slate-100",children:t("login.forgotPassword")}),e.jsx("button",{className:"text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors w-8 h-8 flex items-center justify-center rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800",onClick:()=>o(!1),children:"×"})]}),e.jsx("p",{className:"text-sm text-slate-600 dark:text-slate-400 mb-4",children:h||t("login.resetHelpIntro")}),!h&&e.jsxs(e.Fragment,{children:[e.jsx("pre",{className:"bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl p-4 text-xs overflow-auto font-mono text-slate-800 dark:text-slate-200",children:`./msm reset-password -u ${n||"admin"} -w MyTestPass123`}),e.jsx("p",{className:"text-xs text-slate-500 dark:text-slate-500 mt-4",children:t("login.resetHelpNote")})]}),e.jsx("div",{className:"mt-6 flex justify-end",children:e.jsx("button",{className:"px-4 py-2 rounded-xl border-2 border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors font-medium text-slate-700 dark:text-slate-300",onClick:()=>o(!1),children:t("common.close")})})]})})]})};export{Y as LoginPage};
