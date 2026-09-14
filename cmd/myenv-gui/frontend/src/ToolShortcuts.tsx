import React from 'react';
import type { LocalInventory } from './LocalEnvironment';
import type { PackageContext, PackageItem } from './package-model';
import './tool-shortcuts.css';

export type ManagerEntry = { name:string; version?:string; context:PackageContext; item:PackageItem };
const order:Record<string,string[]>={python:['uv','pip'],node:['npm','pnpm']};

// Reuse inspected identities. A project pip must never fall back to machine pip.
export function inspectedManagers(result:LocalInventory|undefined,tool:string,project:boolean):ManagerEntry[] {
    const available=(result?.tools||[]).filter(t=>t.ecosystem===tool&&order[tool]?.includes(t.name)&&t.state==='available'&&t.path&&(!project||t.name!=='pip'));
    const rows:ManagerEntry[]=available.map(t=>{
        const version=t.version?.split(/\s+/)[0];
        return {name:t.name,version,context:{target:{ecosystem:tool,scope:'tool',root:'',manager:t.name,manager_path:t.path,interpreter:t.interpreter,tool:t.name},label:t.name==='pip'?'本机 pip · 当前 PATH 解释器':`本机 ${t.name}${t.name==='uv'?' · Astral 工具':''}`,path:t.interpreter||t.path},item:{name:t.name,version,kind:'tool',state:t.state}};
    });
    const python=result?.project?.python;
    const pip=python?.packages?.find((p:any)=>p.name.toLowerCase()==='pip'&&p.version);
    if(project&&tool==='python'&&python?.state==='available'&&python.interpreter&&pip) {
        rows.push({name:'pip',version:pip.version,context:{target:{ecosystem:'python',scope:'tool',root:'',tool:'pip',manager:'pip',manager_path:python.interpreter,interpreter:python.interpreter},label:'项目 pip · 当前项目解释器',path:python.interpreter},item:{name:'pip',version:pip.version,kind:'tool',state:'available'}});
    }
    const seen=new Set<string>();
    return rows.filter(row=>{const id=row.name+'|'+row.context.path; if(seen.has(id))return false;seen.add(id);return true;});
}

export function ToolShortcuts({tool,entries,busy,checked,placement,onChoose,onMore}:{tool:string;entries:ManagerEntry[];busy:boolean;checked:boolean;placement:'sidebar'|'content';onChoose:(name:string)=>void;onMore:()=>void}) {
    if(!order[tool])return null;
    const names=order[tool].filter(name=>entries.some(entry=>entry.name===name)).slice(0,3);
    return <div className={`tool-shortcuts tool-shortcuts-${placement}`} role="group" aria-label={`${tool==='python'?'Python':'Node.js'} 管理工具快捷入口`}>
        {placement==='content'&&<span className="tool-shortcuts-label">管理工具</span>}
        {names.map(name=>{const choices=entries.filter(entry=>entry.name===name);return <button key={name} className="tool-shortcut" disabled={busy} aria-label={`管理 ${name}`} title={choices.length>1?`${name} · 选择安装位置`:choices[0].context.label+'\n'+choices[0].context.path} onClick={()=>onChoose(name)}>{name}</button>;})}
        <button className="tool-shortcut shortcut-more" disabled={busy} onClick={onMore}>{names.length?'更多…':checked?'查看工具':'检查工具'}</button>
    </div>;
}
