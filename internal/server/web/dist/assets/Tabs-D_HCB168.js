import{j as r}from"./editor-vendor-Ca1lPq8W.js";import{r as i}from"./react-vendor-Bt7EHJHg.js";import"./index-CDz96t7w.js";import{i as b}from"./utils-vendor-yfWiyOYv.js";const c=i.createContext(void 0),l=()=>{const e=i.useContext(c);if(!e)throw new Error(b.t("common.tabsContextRequired"));return e},p=({defaultValue:e,value:t,onValueChange:s,children:a,className:n=""})=>{const[o,u]=i.useState(e),d=t??o,m=s??u;return r.jsx(c.Provider,{value:{value:d,onValueChange:m},children:r.jsx("div",{className:n,children:a})})},h=({children:e,className:t=""})=>r.jsx("div",{role:"tablist",className:`inline-flex items-center gap-1 rounded-lg bg-muted p-1 ${t}`,children:e}),C=({value:e,children:t,className:s=""})=>{const{value:a,onValueChange:n}=l(),o=a===e;return r.jsx("button",{role:"tab","aria-selected":o,"data-state":o?"active":"inactive",onClick:()=>n(e),className:`
        inline-flex items-center justify-center whitespace-nowrap rounded-md
        px-3 py-1.5 text-sm font-medium
        transition-all duration-200
        focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2
        disabled:pointer-events-none disabled:opacity-50
        ${o?"bg-background text-foreground shadow-sm":"text-muted-foreground hover:text-foreground hover:bg-background/50"}
        ${s}
      `,children:t})},j=({value:e,children:t,className:s=""})=>{const{value:a}=l(),n=a===e;return n?r.jsx("div",{role:"tabpanel","data-state":n?"active":"inactive",className:`mt-2 animate-fade-in ${s}`,children:t}):null};export{p as T,h as a,C as b,j as c};
