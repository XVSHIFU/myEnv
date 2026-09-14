import React, { useEffect, useRef, useState } from 'react';
import { Icon } from './Icon';
import type { Task } from './model';
import type { PackageApplyResult, PackageCatalog, PackageContext, PackageItem, PackagePlan, PackageRequest, PackageSearchResult, PackageSubmitPayload } from './package-model';
import { desiredLabel, packageContextKey, packageKind } from './package-model';
import './package-management.css';

type DraftItem = PackageItem & {desired:string;included:boolean};
type Props = {
    open:boolean;
    context:PackageContext;
    items:PackageItem[];
    mode?:'manage'|'store';
    busy:boolean;
    activity?:Task;
    onClose:()=>void;
    onSubmit:(action:string,payload:PackageSubmitPayload)=>Promise<Task|undefined>;
    onCancel?:()=>void;
};
const running=(task?:Task)=>!!task&&['running','confirm','canceling'].includes(task.state);
const operationNames:Record<string,string>={install:'安装',update:'更新',switch:'切换版本',remove:'卸载'};
const taskNames:Record<string,string>={'package-search':'查询官方包目录','package-catalog':'查询官方版本','package-plan':'核对变更计划','package-apply':'执行包变更'};
const outcomeNames:Record<string,string>={succeeded:'已完成',applied:'已应用',updated:'已更新',installed:'已安装',removed:'已卸载',changed:'已变更',not_applied:'未应用',unverified:'未核验',failed:'未完成',canceled:'已取消',cancelled:'已取消',skipped:'未执行',unchanged:'未变更',not_run:'未执行',unknown:'需要复查'};
function initialDesired(_item:PackageItem) { return 'latest'; }
function editableKind(kind?:string) {return ({dependencies:'dependency',devDependencies:'development',optionalDependencies:'optional'} as Record<string,string>)[kind||'']||kind;}

export function PackageManager({open,context,items,mode='manage',busy,activity,onClose,onSubmit,onCancel}:Props) {
    const panel=useRef<HTMLElement>(null),heading=useRef<HTMLHeadingElement>(null),restoreFocus=useRef<HTMLElement|null>(null),closeAction=useRef(onClose);
    closeAction.current=onClose;
    const initialized=useRef(''),lastHandled=useRef(''),autoCatalog=useRef(''),inFlight=useRef(false);
    const [drafts,setDrafts]=useState<DraftItem[]>([]),[operation,setOperation]=useState('update'),[view,setView]=useState<'edit'|'store'|'review'|'result'>('edit');
    const [query,setQuery]=useState(''),[search,setSearch]=useState<PackageSearchResult>(),[catalogs,setCatalogs]=useState<Record<string,PackageCatalog>>({});
    const [catalogName,setCatalogName]=useState(''),[showPreview,setShowPreview]=useState(false),[versionFilter,setVersionFilter]=useState('');
    const [watched,setWatched]=useState<Task>(),[jobName,setJobName]=useState(''),[error,setError]=useState('');
    const [planTask,setPlanTask]=useState<Task>(),[planDraft,setPlanDraft]=useState(''),[outcome,setOutcome]=useState<PackageApplyResult>(),[resultState,setResultState]=useState(''),[starting,setStarting]=useState(false);
    const scopeKey=packageContextKey(context),sessionKey=scopeKey+'|'+mode+'|'+items.map(p=>p.name).join('|');
    const task=activity?.id===watched?.id?activity:watched,ownBusy=running(task),selected=drafts.filter(p=>p.included),catalog=catalogs[catalogName];
    const request:PackageRequest={target:context.target,operation,items:selected.map(p=>({name:p.name,desired:operation==='remove'?'':p.desired.trim(),kind:p.kind}))};
    const fingerprint=JSON.stringify(request),plan=planTask?.result as PackagePlan|undefined;
    const planValid=!!plan?.id&&planDraft===fingerprint&&planTask?.state==='succeeded';
    const official=context.target.ecosystem==='node'?'npm':'PyPI';
    const pending=busy||ownBusy||starting;

    useEffect(()=>{
        if(!open||initialized.current===sessionKey)return;
        initialized.current=sessionKey;autoCatalog.current='';lastHandled.current='';
        setDrafts(items.map(p=>({...p,kind:editableKind(p.kind),desired:initialDesired(p),included:true})));
        setOperation(items.some(p=>p.version)?'update':'install');setView(mode==='store'?'store':'edit');
        setQuery('');setSearch(undefined);setCatalogs({});setCatalogName(items.length===1?items[0].name:'');setVersionFilter('');setShowPreview(false);
        setWatched(undefined);setPlanTask(undefined);setPlanDraft('');setOutcome(undefined);setError('');
    },[open,sessionKey]);

    useEffect(()=>{
        if(!open)return;
        restoreFocus.current=document.activeElement as HTMLElement;
        heading.current?.focus();
        function key(event:KeyboardEvent) {
            if(event.key==='Escape'){event.preventDefault();event.stopPropagation();closeAction.current();return;}
            if(event.key!=='Tab'||!panel.current)return;
            const controls=Array.from(panel.current.querySelectorAll<HTMLElement>('button:not(:disabled),input:not(:disabled),select:not(:disabled),textarea:not(:disabled),a[href],[tabindex="0"]')).filter(node=>node.getClientRects().length>0);
            const first=controls[0],last=controls[controls.length-1];
            if(!first){event.preventDefault();heading.current?.focus();return;}
            if(event.shiftKey&&(document.activeElement===first||!controls.includes(document.activeElement as HTMLElement))){event.preventDefault();last.focus();}
            else if(!event.shiftKey&&(document.activeElement===last||!panel.current.contains(document.activeElement))){event.preventDefault();first.focus();}
        }
        document.addEventListener('keydown',key,true);
        return()=>{document.removeEventListener('keydown',key,true);const previous=restoreFocus.current;requestAnimationFrame(()=>{if(previous?.isConnected&&previous.getClientRects().length&&!previous.closest('[inert]'))previous.focus();});};
    },[open]);

    async function submit(action:string,payload:PackageSubmitPayload) {
        if(inFlight.current||pending)return;
        inFlight.current=true;setStarting(true);setError('');setJobName(payload.packageName||'');
        try {const created=await onSubmit(action,payload);if(created)setWatched(created);}
        catch(reason){setError(reason instanceof Error?reason.message:String(reason));}
        finally{inFlight.current=false;setStarting(false);}
    }
    function lookup(name:string) {setCatalogName(name);setVersionFilter('');void submit('package-catalog',{package:request,packageName:name});}
    useEffect(()=>{
        if(!open||mode==='store'||items.length!==1||pending||initialized.current!==sessionKey||autoCatalog.current===sessionKey)return;
        autoCatalog.current=sessionKey;lookup(items[0].name);
    },[open,sessionKey,pending]);

    useEffect(()=>{
        if(!task||running(task))return;
        const stamp=task.id+':'+task.state;if(lastHandled.current===stamp)return;lastHandled.current=stamp;
        if(task.error){setError(task.error.message+(task.error.next_action?' '+task.error.next_action:''));}
        else if(task.state==='canceled')setError('任务已取消。已填写的目标保留，可以调整后重试。');
        if(task.request.action==='package-search'&&task.state==='succeeded')setSearch(task.result as PackageSearchResult);
        if(task.request.action==='package-catalog'&&task.state==='succeeded'){
            const result=task.result as PackageCatalog;setCatalogs(previous=>({...previous,[result.name||jobName]:result,[jobName||result.name]:result}));
        }
        if(task.request.action==='package-plan'&&task.state==='succeeded') {setPlanTask(task);setView('review');}
        if(task.request.action==='package-apply') {
            if(task.result?.outcomes)setOutcome(task.result as PackageApplyResult);
            setResultState(task.state);setView('result');setPlanTask(undefined);
        }
    },[task,jobName]);

    function edit(name:string,patch:Partial<DraftItem>) {setDrafts(previous=>previous.map(p=>p.name===name?{...p,...patch}:p));setPlanTask(undefined);setOutcome(undefined);}
    function changeOperation(value:string) {
        setOperation(value);
        if(value!=='remove')setDrafts(previous=>previous.map(item=>({...item,desired:value==='switch'?item.version||'':'latest'})));
        setPlanTask(undefined);setOutcome(undefined);setError('');
    }
    function add(name:string) {
        setDrafts(previous=>previous.some(p=>p.name===name)?previous.map(p=>p.name===name?{...p,included:true}:p):[...previous,{name,desired:'latest',kind:context.target.scope==='project'?'dependency':undefined,included:true}]);
        setPlanTask(undefined);setView('edit');setCatalogName(name);if(!catalogs[name])lookup(name);
    }
    function preview() {setPlanDraft(fingerprint);setPlanTask(undefined);void submit('package-plan',{package:request});}
    function apply() {if(planValid&&planTask)void submit('package-apply',{package:(planTask.request as Task['request']&{package?:PackageRequest}).package||request,planTaskId:planTask.id,expectedDigest:plan?.state_digest});}
    const invalid=selected.length===0||selected.length>100||operation!=='remove'&&selected.some(p=>!p.desired.trim());

    if(!open)return null;
    return <div className="package-manager-layer">
        <div className="package-manager-scrim" aria-hidden="true" onClick={onClose}/>
        <aside className="package-manager" ref={panel} role="dialog" aria-modal="true" aria-labelledby="package-manager-title">
            <header className="package-manager-header"><div><h2 id="package-manager-title" tabIndex={-1} ref={heading}>{view==='store'?'官方包目录':context.target.scope==='tool'?`${items[0]?.name||'工具'} 管理`:drafts.length>1?'批量管理包':'管理包'}</h2><p>{context.label}{selected.length>1?` · 已选 ${selected.length} 项`:''}</p></div><button className="icon-button" aria-label="关闭包管理" onClick={onClose}><Icon name="close"/></button></header>
            <div className="package-manager-scope"><Icon name="folder"/><span><strong>{context.target.manager||official}</strong><code title={context.path}>{context.path}</code></span></div>
            <nav className="package-manager-nav" aria-label="包管理步骤"><button aria-current={view==='edit'?'step':undefined} onClick={()=>setView('edit')}>目标与操作{selected.length>0&&<span>{selected.length}</span>}</button>{context.target.scope!=='tool'&&<button aria-current={view==='store'?'page':undefined} onClick={()=>setView('store')}>{official} 官方目录</button>}<button disabled={!planValid} aria-current={view==='review'?'step':undefined} onClick={()=>setView('review')}>变更预览</button>{outcome&&<button aria-current={view==='result'?'page':undefined} onClick={()=>setView('result')}>执行结果</button>}</nav>
            <div className="package-manager-content">
                {error&&<div className="package-notice is-error" role="alert"><Icon name="alert"/><div><strong>{error.includes('OFFICIAL_NOT_FOUND')?'官方未找到':task?.request.action==='package-plan'?'暂时无法生成计划':'操作未完成'}</strong><p>{error.replace('OFFICIAL_NOT_FOUND: ','')}</p>{task?.request.action==='package-catalog'&&<button disabled={pending} className="text-link" onClick={()=>lookup(catalogName)}>重试官方查询</button>}</div></div>}
                {view==='store'&&<section className="package-store"><p className="package-help">{context.target.ecosystem==='node'?'搜索 npm 官方仓库中的包，查看版本后加入当前环境的变更。':'输入完整包名查询 PyPI 官方仓库。PyPI 不提供此处可用的公开关键词搜索接口。'}</p><form className="package-store-search" onSubmit={event=>{event.preventDefault();if(query.trim())void submit('package-search',{package:request,packageQuery:query.trim()});}}><label className="search-field"><span className="sr-only">{context.target.ecosystem==='node'?'搜索 npm 官方包':'按完整包名查询 PyPI'}</span><Icon name="search"/><input value={query} disabled={pending} placeholder={context.target.ecosystem==='node'?'包名或关键词，例如 typescript':'完整包名，例如 requests'} onChange={event=>setQuery(event.target.value)}/></label><button disabled={pending||!query.trim()}>{context.target.ecosystem==='node'?'搜索':'查询'}</button></form>
                    {!search&&!ownBusy&&<div className="package-store-empty"><Icon name="search"/><h3>从 {official} 官方仓库查找</h3><p>安装位置固定为上方环境。加入列表后还会核对实际版本并展示变更计划。</p></div>}
                    {search&&<><p className="package-help">{search.mode==='exact'?'按完整包名查询':'搜索结果'} · {search.packages.length} 项{search.truncated?' · 当前结果有数量上限':''}</p>{!search.packages.length?<div className="package-notice"><Icon name="info"/><div><strong>官方未找到</strong><p>{search.query} 没有返回官方记录。检查完整名称；本机已有包仍可从列表中管理或卸载。</p></div></div>:search.packages.map(p=><article className="package-search-result" key={p.name}><div><strong>{p.name}</strong><span>{p.latest||p.version||''}</span><p>{p.summary||p.description||'官方未提供简介'}</p></div><button disabled={pending} onClick={()=>add(p.name)}>{drafts.some(d=>d.name===p.name&&d.included)?'查看目标':'加入变更'}<Icon name="arrow"/></button></article>)}</>}
                </section>}
                {view==='edit'&&<>
                    {!drafts.length?<div className="package-store-empty"><Icon name="layers"/><h3>先选择要管理的包</h3><p>可以从官方目录添加，或关闭抽屉后在包列表中选择。</p><button onClick={()=>setView('store')}>打开 {official} 官方目录</button></div>:<>
                        <div className="package-operation"><label htmlFor="package-operation">本次操作</label><select id="package-operation" value={operation} disabled={pending} onChange={event=>changeOperation(event.target.value)}><option value="update">更新到期望版本</option><option value="switch">切换到指定版本</option><option value="install">安装</option><option value="remove">卸载</option></select></div>
                        <p className="package-help">{operation==='remove'?'将移除所选位置中的包。预览会列出具体目标及命令。':context.target.scope==='project'&&context.target.ecosystem==='node'?'期望写入当前项目声明，预览时解析出本次安装的具体版本。':'“最新版”在本次预览时解析为官方稳定版本，不会自动在后台更新。'}</p>
                        {drafts.length>1&&<div className="package-batch-heading"><strong>{selected.length} / {drafts.length} 项参与变更</strong>{operation!=='remove'&&<button disabled={pending} className="text-link" onClick={()=>{setDrafts(previous=>previous.map(p=>p.included?{...p,desired:'latest'}:p));setPlanTask(undefined);}}>已选全部设为最新版</button>}</div>}
                        {selected.length>100&&<p className="package-notice is-error">本次最多管理 100 个包，请取消部分选择后分批执行。</p>}
                        <div className="package-draft-list">{drafts.map(item=><article className={'package-draft-item'+(!item.included?' is-excluded':'')} key={item.name}><div className="package-draft-name">{drafts.length>1&&<input type="checkbox" checked={item.included} disabled={pending} aria-label={`本次管理 ${item.name}`} onChange={event=>edit(item.name,{included:event.target.checked})}/>}<div><strong>{item.name}</strong><p>已安装 {item.version||'未核验'}{item.requested?` · 原期望 ${desiredLabel(item.requested)}`:''}</p></div>{drafts.length>1&&<button disabled={pending} className="text-link" onClick={()=>lookup(item.name)} aria-expanded={catalogName===item.name}>官方版本<Icon name="chevron"/></button>}</div>
                            {operation!=='remove'&&<div className="package-desired-fields"><label>期望<select disabled={pending||!item.included} value={item.desired==='latest'?'latest':'exact'} onChange={event=>edit(item.name,{desired:event.target.value==='latest'?'latest':item.version||''})}><option value="latest">最新版</option><option value="exact">指定版本</option></select></label>{item.desired!=='latest'&&<label className="package-exact-field">版本号<input disabled={pending||!item.included} value={item.desired} placeholder="输入具体版本号" onChange={event=>edit(item.name,{desired:event.target.value})}/></label>}{context.target.scope==='project'&&context.target.ecosystem==='node'&&<label>依赖类型<select disabled={pending||!item.included} value={item.kind||'dependency'} onChange={event=>edit(item.name,{kind:event.target.value})}><option value="dependency">依赖</option><option value="development">开发依赖</option><option value="optional">可选依赖</option></select></label>}</div>}
                            {catalogName===item.name&&<CatalogSection catalog={catalog} name={item.name} official={official} busy={pending} showPreview={showPreview} setShowPreview={setShowPreview} filter={versionFilter} setFilter={setVersionFilter} selected={item.desired} onRefresh={()=>lookup(item.name)} onSelect={version=>edit(item.name,{desired:version})} removable={operation==='remove'}/>}
                        </article>)}</div>
                    </>}
                </>}
                {view==='review'&&plan&&<section className="package-review"><div className="package-review-heading"><Icon name={operation==='remove'?'alert':'check'}/><div><h3>确认这次{operationNames[operation]}</h3><p>以下版本已在预览时确定，执行使用这份计划。</p></div></div><div className="package-plan-changes">{plan.changes.map(change=><article key={change.name}><strong>{change.name}</strong><div><span>{change.before||'未安装'}</span><Icon name="arrow"/><strong>{operation==='remove'?'移除':change.version}</strong></div><small>{packageKind(change.kind)}{change.desired?` · 期望 ${desiredLabel(change.desired)}`:''}</small></article>)}</div>{plan.warnings?.map((warning,index)=><p className="package-plan-warning" key={index}>{warning}</p>)}<details className="technical package-plan-commands"><summary>查看执行位置与命令</summary>{plan.commands.map((command,index)=><div key={index}><p>{command.directory}</p><pre>{[command.executable,...command.args].map(value=>JSON.stringify(value)).join(' ')}</pre></div>)}</details></section>}
                {view==='result'&&<section className="package-results"><h3>{resultState==='succeeded'?'本次操作已完成':resultState==='canceled'?'操作已取消':'本次操作未全部完成'}</h3><p className="package-help">每项结果按执行和重新检查记录显示。已完成的外部包操作不会自动回滚。</p>{outcome?.outcomes?.map((item,index)=><article className="package-outcome-row" key={item.name+index}><div><strong>{item.name}</strong><p>{item.version||''}</p>{item.problem&&<p className="package-outcome-problem">{item.problem}</p>}</div><span className={'tag '+(['succeeded','applied','updated','installed','removed'].includes(item.state)?'green':['failed','unknown','unverified','not_applied'].includes(item.state)?'red':'')}>{outcomeNames[item.state]||item.state}</span></article>)}{!outcome&&<p className="package-help">没有返回逐项执行结果，请刷新原列表核对实际环境。</p>}{outcome?.log&&<details className="technical"><summary>原始执行日志{outcome.log_truncated?'（已截断）':''}</summary><pre>{outcome.log}</pre></details>}</section>}
            </div>
            <footer className="package-manager-footer"><div className="package-manager-feedback" role="status">{ownBusy?<><span className="spinner"/><span>{task?.state==='canceling'?'正在取消并检查结果':taskNames[task?.request.action||'']||'正在处理'}{jobName&&task?.request.action==='package-catalog'?` · ${jobName}`:''}</span></>:<span>{view==='review'?'确认后修改上方环境':view==='result'?'操作记录保留在任务历史':selected.length?`${selected.length} 项 · 尚未执行变更`:'选择包后预览变更'}</span>}</div><div className="package-manager-actions">{ownBusy&&onCancel?<button disabled={task?.state==='canceling'} onClick={onCancel}>取消任务</button>:view==='review'?<><button onClick={()=>setView('edit')}>返回调整</button><button className={'primary'+(operation==='remove'?' package-confirm-remove':'')} disabled={pending||!planValid} onClick={apply}>确认{operationNames[operation]}<Icon name="arrow"/></button></>:view==='result'?<><button disabled={pending} onClick={()=>{setView('edit');setError('');}}>返回管理</button><button className="primary" onClick={onClose}>完成</button></>:<><button onClick={onClose}>关闭</button><button className="primary" disabled={pending||invalid} onClick={preview}>预览{operationNames[operation]}<Icon name="arrow"/></button></>}</div></footer>
        </aside>
    </div>;
}

function CatalogSection({catalog,name,official,busy,showPreview,setShowPreview,filter,setFilter,selected,onRefresh,onSelect,removable}:{catalog?:PackageCatalog;name:string;official:string;busy:boolean;showPreview:boolean;setShowPreview:(value:boolean)=>void;filter:string;setFilter:(value:string)=>void;selected:string;onRefresh:()=>void;onSelect:(version:string)=>void;removable:boolean}) {
    const versions=(catalog?.versions||[]).filter(version=>(showPreview||!version.preview)&&(!filter||version.version.toLowerCase().includes(filter.toLowerCase())));
    return <section className="package-catalog"><div className="package-catalog-heading"><strong>{official} 官方版本</strong><button className="text-link" disabled={busy} onClick={onRefresh}><Icon name="refresh"/>刷新</button></div>
        {!catalog?<p className="package-help">{busy?'正在查询官方记录…':'官方记录尚未查询，可点击刷新。'}</p>:catalog.state==='not_found'?<div className="package-notice"><Icon name="info"/><div><strong>官方未找到 {name}</strong><p>可能来自私有仓库、文件或已下架。保留本机记录，仍可选择卸载。</p></div></div>:<>
            {catalog.summary&&<p className="package-catalog-summary">{catalog.summary}</p>}<p className="package-help">{catalog.latest?`官方最新版 ${catalog.latest}`:'官方未提供稳定版标签'}{catalog.checked_at?` · 查询于 ${new Date(catalog.checked_at).toLocaleTimeString()}`:''}</p>
            {!removable&&<><div className="package-version-tools"><label className="search-field"><span className="sr-only">筛选 {name} 官方版本</span><Icon name="search"/><input type="search" placeholder="筛选版本" value={filter} onChange={event=>setFilter(event.target.value)}/></label><label className="package-preview-option"><input type="checkbox" checked={showPreview} onChange={event=>setShowPreview(event.target.checked)}/>预发布</label></div><div className="package-official-versions">{versions.slice(0,8).map(version=><button key={version.version} type="button" disabled={busy||version.yanked||version.available===false} aria-pressed={selected===version.version} onClick={()=>onSelect(version.version)}><span>{version.version}</span><small>{version.yanked?'已撤回':version.available===false?'无可安装制品':version.preview?'预发布':version.version===catalog.latest?'最新版':version.deprecated?'已弃用':''}</small>{selected===version.version&&<Icon name="check"/>}</button>)}</div>{!versions.length&&<p className="package-help">当前筛选没有可显示的版本。</p>}{versions.length>8&&<p className="package-help">显示前 8 个匹配版本，输入版本号缩小范围，或在上方直接指定。</p>}{catalog.versions_truncated&&<p className="package-help">官方版本记录较多，列表有数量上限；指定的完整版本会在预览时单独核验。</p>}</>}
        </>}
    </section>;
}
