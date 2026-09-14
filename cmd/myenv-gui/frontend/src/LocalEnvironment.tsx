import React, { useEffect, useMemo, useState } from 'react';
import { Icon } from './Icon';
import { desiredLabel, packageContextKey, packageKind } from './package-model';
import type { PackageContext, PackageItem } from './package-model';
import './package-management.css';

export type Installation = {id:string; tool:string; path:string; owner:string; manager:string; state:string; version_output?:string; problem?:string; discovery_source:string; actions:string[]};
export type LocalCommand = {tool:string; command:string; path?:string; version_output?:string; state:string; problem?:string; installation_id?:string};
export type LocalInventory = {installations:Installation[]; commands:LocalCommand[]; context?:{platform:string; directory?:string; explanation:string}; tools?:any[]; project?:any; package_groups?:any[]; warnings?:string[]; coverage?:string};
export type InventorySnapshot = {result:LocalInventory; checkedAt:number};
export const languageNames:Record<string,string> = {python:'Python',node:'Node.js',java:'Java',go:'Go',rust:'Rust'};
export function versionText(value?:string) { return value?.split(/\r?\n/)[0].replace(/^Python\s+|^go version go|^rustc\s+|^(?:openjdk|java) (?:version )?"?/,'').replace(/".*$|\s+\(.*$|\s+(?:windows|linux|darwin)\/.*$/,'') || ''; }
const stateNames:Record<string,string> = {available:'已检测',missing:'未找到',not_found:'未找到',unverified:'待核验',broken:'无法运行',alias:'系统别名',shim:'版本管理器入口',declared:'仅声明',installed:'已安装',unavailable:'不可用',not_installed:'未安装'};
const statusText = (state:string) => stateNames[state] || state;
export function commandLabel(command?:LocalCommand) { return versionText(command?.version_output) || (command?.path ? statusText(command.state) : '未在 PATH 找到'); }

export function LocalEnvironment({tool,snapshot,busy,onRefresh,onManage,onToolchain,onPackages}: {
    tool:string;snapshot?:InventorySnapshot;busy:boolean;onRefresh:()=>void;onManage:(row:Installation)=>void;onToolchain:()=>void;onPackages:()=>void;
}) {
    const command = snapshot?.result.commands?.find(c=>c.tool===tool), installs = snapshot?.result.installations?.filter(i=>i.tool===tool)||[];
    return <section className="local-environment" aria-label="本机环境检查结果">
        <div className="section-heading"><h3>当前 PATH 命令</h3><button disabled={busy} onClick={onRefresh}><Icon name="refresh"/>{busy?'检查中':'检查本机环境'}</button></div>
        {!snapshot ? <div className="local-empty"><Icon name="search"/><div><h3>先看看电脑上已有的 {languageNames[tool]}</h3><p>检查命令位置、已安装版本和管理工具。检查后会保留记录，便于下次打开时查看。</p><button className="primary" disabled={busy} onClick={onRefresh}>{busy?'正在检查…':'开始检查'}</button></div></div>
        : <><div className="resolved-command"><div className="resolved-heading"><strong>{versionText(command?.version_output)|| (command?.path?'需要核验':'未在 PATH 中找到')}</strong><span className={'tag '+(command?.state==='available'?'green':'amber')}>{command?statusText(command.state):'未找到'}</span></div><code>{command?.path||`当前应用的 PATH 中没有 ${tool==='rust'?'rustc':tool}`}</code>{command?.problem&&<p>{command.problem}</p>}</div>
        <p className="inspection-caption">检查于 {new Date(snapshot.checkedAt).toLocaleString()} · {snapshot.result.context?.platform || '本机'}<br/>按应用启动时的 PATH 解析；终端配置、虚拟环境或后来修改的 PATH 可能不同。记录不会自动刷新。</p>
        <div className="section-heading local-section"><h3>发现的安装 <span className="muted">{installs.length}</span></h3>{['node','python'].includes(tool)&&<button className="text-link" onClick={onPackages}>查看包与工具<Icon name="arrow"/></button>}</div>
        {!installs.length&&<p className="inspection-caption">当前检查范围内没有找到其他安装。自定义位置可在终端核对，或使用 myEnv 准备所需版本。</p>}
        <div className="installation-list">{installs.map(i=><article className="installation-row" key={i.id}><div className="installation-title"><strong>{versionText(i.version_output)||languageNames[i.tool]}</strong><span className="tag">{i.id===command?.installation_id?'PATH 首选':i.owner==='myenv'?'myEnv 受管':i.manager==='unknown'?'外部安装':i.manager}</span><span className="muted">{statusText(i.state)}</span></div><code>{i.path}</code><div className="installation-actions"><span>{i.owner==='myenv'?'独立的 myEnv 默认工具链':i.manager==='unknown'?'来源未确认；检查记录不代表已接管':`由 ${i.manager} 管理`}</span>{i.owner==='external'&&<button className="text-link" disabled={busy} onClick={()=>onManage(i)}>管理此安装<Icon name="arrow"/></button>}</div></article>)}</div></>}
        <div className="managed-entry"><div><h3>myEnv 默认工具链</h3><p>跨项目使用的一套受管版本。与终端 PATH 中已有的安装分别管理。</p></div><button disabled={busy} onClick={onToolchain}>选择受管版本<Icon name="arrow"/></button></div>
        {!!snapshot?.result.warnings?.length&&<details className="technical"><summary>检查范围与未完成项目（{snapshot.result.warnings.length}）</summary><p className="inspection-caption">{snapshot.result.coverage}</p><pre>{snapshot.result.warnings.join('\n')}</pre></details>}
    </section>;
}

export function PackageEnvironment({tool,snapshot,busy,onRefresh,project,onOpenManage}: {
    tool:string;snapshot?:InventorySnapshot;busy:boolean;onRefresh:()=>void;project:boolean;
    onOpenManage?:(context:PackageContext,items:PackageItem[],mode?:'manage'|'store')=>void;
}) {
    const name=languageNames[tool], result=snapshot?.result;
    return <section className="package-environment"><div className="section-heading"><h3>{name} 包与工具</h3><button disabled={busy} onClick={onRefresh}><Icon name="refresh"/>{busy?'检查中':'刷新检查'}</button></div>
        <p className="inspection-caption">{tool==='node'?'npm、pnpm 管理 Node.js 包；项目依赖与全局工具按安装位置分别显示。':'pip 的包属于具体 Python 解释器；uv 可独立安装，用于管理 Python、项目依赖和工具。'}</p>
        {!result?<div className="local-empty"><Icon name="search"/><div><h3>还没有这次环境的检查记录</h3><p>{project?'只检查当前选中的项目目录及本机工具。':'检查本机工具和明确的包安装位置。'}</p><button className="primary" disabled={busy} onClick={onRefresh}>开始检查</button></div></div>:<>
            {project&&<PackageGroups result={result} tool={tool} project busy={busy} onOpenManage={onOpenManage}/>}
            <div className="section-heading local-section"><h3>本机管理工具</h3><small>按实际安装位置管理</small></div>
            {(result.tools||[]).filter(t=>t.ecosystem===tool).map(t=><article className="installation-row package-tool-row" key={t.name+String(t.path)}><div className="package-tool-copy"><div className="installation-title"><strong>{t.name}</strong><span>{t.version || statusText(t.state)}</span></div>{t.path&&<code>{t.path}</code>}{t.interpreter&&<p className="inspection-caption">所属解释器：{t.interpreter}</p>}{t.problem&&<p className="inspection-caption">{t.problem}</p>}</div><button className="text-link package-manage-button" disabled={busy||!onOpenManage} onClick={()=>onOpenManage?.({target:{ecosystem:tool,scope:'tool',root:'',interpreter:t.interpreter,manager:t.name,manager_path:t.path,tool:t.name},label:`本机 ${t.name}`,path:t.path||t.interpreter||'尚未发现安装位置'},[{name:t.name,version:t.name==='uv'?t.version?.split(/\s+/)[0]:t.version,kind:'tool',state:t.state}])}>管理<Icon name="arrow"/></button></article>)}
            {!project&&<PackageGroups result={result} tool={tool} project={false} busy={busy} onOpenManage={onOpenManage}/>}
            <p className="inspection-caption">列表来自最近一次检查。管理时先核对官方版本和实际位置，再预览并确认变更。</p>
        </>}
    </section>;
}

function PackageGroups({result,tool,project,busy,onOpenManage}:{result:LocalInventory;tool:string;project:boolean;busy:boolean;onOpenManage?:(context:PackageContext,items:PackageItem[],mode?:'manage'|'store')=>void}) {
    const groups=project?[]:(result.package_groups||[]).filter(g=>g.ecosystem===tool);
    const detail=result.project?.[tool];
    if(project&&detail) groups.push({...detail,scope:'project',label:'当前项目',path:result.project.directory});
    return <>{groups.map((group,index)=>{
        const manager=group.manager==='npm/pnpm'?'npm':group.manager==='pip/uv'?'pip':group.manager?.split('@')[0];
        const context:PackageContext={target:{ecosystem:tool,scope:group.scope,root:group.root||'',directory:project?result.project?.directory:undefined,interpreter:group.interpreter||(tool==='node'?result.commands.find(c=>c.tool==='node')?.path:undefined),manager,manager_path:result.tools?.find(t=>t.name===(manager|| (tool==='node'?'npm':'pip')))?.path},label:group.label||({global_package_root:'全局工具安装位置',global:'全局工具',interpreter:'当前 PATH 解释器中的包',project:'当前项目'} as Record<string,string>)[group.scope]||'包安装位置',path:group.interpreter||group.root||group.path||group.manifest||''};
        return <PackageGroupList key={packageContextKey(context)+String(index)} context={context} group={group} busy={busy} onOpenManage={onOpenManage}/>;
    })}{!groups.length&&<p className="inspection-caption">{project?'当前项目未发现可读取的包记录。':'尚未发现可读取的包安装位置。项目依赖请切换到“项目环境”查看。'}</p>}</>;
}

function PackageGroupList({context,group,busy,onOpenManage}:{context:PackageContext;group:any;busy:boolean;onOpenManage?:(context:PackageContext,items:PackageItem[],mode?:'manage'|'store')=>void}) {
    const [filter,setFilter]=useState(''),[expanded,setExpanded]=useState(false),[selecting,setSelecting]=useState(false),[selected,setSelected]=useState<Set<string>>(new Set());
    const packages:PackageItem[]=group.packages||[];
    const filtered=useMemo(()=>packages.filter(p=>p.name.toLocaleLowerCase().includes(filter.trim().toLocaleLowerCase())),[packages,filter]);
    const visible=expanded?filtered:filtered.slice(0,10), chosen=packages.filter(p=>selected.has(p.name)), allFiltered=filtered.length>0&&filtered.every(p=>selected.has(p.name));
    useEffect(()=>setSelected(previous=>new Set([...previous].filter(name=>packages.some(p=>p.name===name)))),[group.packages]);
    function toggle(name:string) { setSelected(previous=>{const next=new Set(previous);next.has(name)?next.delete(name):next.add(name);return next;}); }
    function toggleAll() { setSelected(previous=>{const next=new Set(previous);filtered.forEach(p=>allFiltered?next.delete(p.name):next.add(p.name));return next;}); }
    return <section className="package-group">
        <div className="section-heading local-section package-group-heading"><h3>{context.label}<span className="package-count">{packages.length}{group.truncated?'+':''}</span></h3><div className="package-group-tools"><label className="search-field package-filter"><span className="sr-only">搜索{context.label}中的包</span><Icon name="search"/><input type="search" placeholder="搜索包" value={filter} onChange={e=>{setFilter(e.target.value);setExpanded(false);}}/></label><button className="quiet" disabled={busy||!packages.length} aria-pressed={selecting} onClick={()=>{setSelecting(!selecting);if(selecting)setSelected(new Set());}}>{selecting?'取消选择':'选择'}</button><button disabled={busy||!onOpenManage} onClick={()=>onOpenManage?.(context,[],'store')}>官方包目录<Icon name="arrow"/></button></div></div>
        <p className="inspection-caption package-group-path"><span>{context.target.manager|| (context.target.ecosystem==='python'?'Python':'Node.js')}</span><code>{context.path}</code></p>
        {group.coverage&&<p className="inspection-caption">{group.coverage.includes('direct requirement')?'当前记录包含 requirements.txt 的直接依赖，不递归读取引用文件。':'包记录属于上方位置；其他解释器、安装前缀与项目分别管理。'}</p>}
        {group.truncated&&<p className="inspection-caption">检查结果已达到数量上限；“全选”只包含本次已检查的记录。</p>}
        {group.problem&&<p className="inspection-caption">{group.problem}</p>}
        {selecting&&<div className="package-selection-bar"><label><input type="checkbox" checked={allFiltered} ref={node=>{if(node)node.indeterminate=!allFiltered&&filtered.some(p=>selected.has(p.name));}} onChange={toggleAll} disabled={!filtered.length||busy}/>全选{filter?'搜索结果':'本组'}（{filtered.length}）</label><span>已选 {chosen.length} 项{chosen.some(p=>!filtered.includes(p))?' · 含筛选外的包':''}</span><button className="primary" disabled={busy||!chosen.length||!onOpenManage} onClick={()=>onOpenManage?.(context,chosen)}>管理已选<Icon name="arrow"/></button></div>}
        {!packages.length?<p className="inspection-caption">{group.state==='missing'?'未发现依赖声明或已安装包，可从官方包目录查找。':'当前范围未发现包，可从官方包目录查找。'}</p>:!filtered.length?<p className="package-filter-empty">没有匹配“{filter}”的包。<button className="text-link" onClick={()=>setFilter('')}>清空搜索</button></p>:<div className={'package-list package-management-list'+(selecting?' is-selecting':'')}>
            <div className="package-row package-labels">{selecting&&<span/>}<span>包名</span><span>已安装 / 期望</span><span>类型</span><span className="package-action-label">操作</span></div>
            {visible.map(p=><div className={'package-row'+(selected.has(p.name)?' is-selected':'')} key={p.name}>{selecting&&<input type="checkbox" checked={selected.has(p.name)} disabled={busy} aria-label={`选择 ${p.name}`} onChange={()=>toggle(p.name)}/>}<strong>{p.name}</strong><span className="package-version-cell"><span>{p.version||'未安装 / 未核验'}</span><small>期望 {desiredLabel(p.requested)}</small></span><span>{packageKind(p.kind)}</span><button className="text-link package-manage-button" disabled={busy||!onOpenManage} aria-label={`管理 ${p.name}`} onClick={()=>onOpenManage?.(context,[p])}>管理<Icon name="arrow"/></button></div>)}
        </div>}
        {filtered.length>10&&<button className="text-link package-expand" aria-expanded={expanded} onClick={()=>setExpanded(!expanded)}>{expanded?'收起至 10 个':`展开其余 ${filtered.length-10} 个包`}<Icon name="chevron"/></button>}
    </section>;
}
