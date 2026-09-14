import React, { useEffect, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import './style.css';
import { TaskResult, CommandEditor, describeTaskError } from './Result';
import type { Request, Lock, View, Task } from './model';
import { Icon } from './Icon';
import { BrandIcon } from './BrandIcon';
import { TaskDock, TaskDrawer } from './TaskFeedback';
import { LocalEnvironment, PackageEnvironment, versionText, commandLabel } from './LocalEnvironment';
import type { InventorySnapshot, Installation } from './LocalEnvironment';
import { PackageManager } from './PackageManager';
import { ToolShortcuts, inspectedManagers } from './ToolShortcuts';
import type { ManagerEntry } from './ToolShortcuts';
import type { PackageContext, PackageItem, PackageSubmitPayload } from './package-model';
type Dir = {
    path: string;
    key: string;
};
type Draft = {
    request: Request;
    commands: Record<string, string>;
    tool: string;
};
declare global {
    interface Window {
        go: {
            main: {
                App: {
                    Start(r: Request): Promise<number>;
                    SetAppearance(theme: string, title: string): Promise<void>;
                    Current(): Promise<Task>;
                    History(): Promise<Task[]>;
                    Cancel(id: number): Promise<void>;
                    Confirm(id: number, b: boolean): Promise<void>;
                    OpenProject(): Promise<string>;
                    OpenManager(): Promise<string>;
                    NormalizeDirectory(p: string): Promise<Dir>;
                    RevealPath(p: string): Promise<void>;
                    CopyPath(p: string): Promise<void>;
                    Version(): Promise<string>;
                    Run(d: string, g: boolean, args: string[]): Promise<string>;
                };
            };
        };
        runtime: {
            EventsOn(n: string, cb: (t: Task) => void): () => void;
        };
    }
}
const api = window.go.main.App, tools = ['python', 'node', 'java', 'go', 'rust'];
const names: Record<string, string> = { python: 'Python', node: 'Node.js', java: 'Java', go: 'Go', rust: 'Rust' };
const commands: Record<string, string> = { python: '["python","--version"]', node: '["node","--version"]', java: '["java","-version"]', go: '["go","version"]', rust: '["rustc","--version"]' };
const defaults: Request = { action: 'workbench', directory: '.', global: false, tool: 'python', selection: '', provider: 'astral', preview: false, locked: false, deep: false, apply: false, major: 0, channel: '', date: '', nodeMirror: '', uvMirror: '', pythonMirror: '', certificate: '', id: '', manager: '', externalAction: 'repair', expectedDigest: '' };
const active = (t?: Task) => !!t && ['running', 'confirm', 'canceling'].includes(t.state), labels: Record<string, string> = { idle: '暂无任务', running: '进行中', confirm: '等待本次构建许可', canceling: '取消已请求，正在收尾', canceled: '已取消', failed: '失败', succeeded: '已完成' };
function read<T>(k: string, d: T): T {
    try {
        return JSON.parse(localStorage.getItem(k) || 'null') ?? d;
    }
    catch {
        return d;
    }
}
function save(k: string, v: unknown) {
    try {
        localStorage.setItem(k, JSON.stringify(v));
        return true;
    }
    catch { return false; }
}
function persistInspection(record:InventorySnapshot,key:string) {
    let stored=save('myenv.inventory.'+key,record);
    if(key!=='@user') { const {project,...machine}=record.result; stored=save('myenv.inventory.@user',{...record,result:machine})&&stored; }
    return stored;
}
function provider(w?: View) { return w?.desired.python?.startsWith('python.org/') ? 'python.org' : 'astral'; }
function source(t: string, p: string, l?: Lock) {
    if (t === 'python')
        return (l ? l.backend.startsWith('uv-') : p !== 'python.org') ? 'CPython · Astral 构建 / uv 后端' : 'CPython · python.org 官方包 / myEnv ZIP 安装';
    return ({ java: 'Eclipse Temurin / HotSpot JDK', node: 'Node.js 官方归档', go: 'Go 官方归档', rust: 'Rust 官方工具链归档' } as Record<string, string>)[t];
}
function Button({ children, onClick, disabled = false, primary = false }: {
    children: React.ReactNode;
    onClick: () => void;
    disabled?: boolean;
    primary?: boolean;
}) { return <button className={'btn' + (primary ? ' primary' : '')} disabled={disabled} onClick={onClick}>{children}</button>; }
function Result({ task }: {
    task?: Task;
}) { return <TaskResult task={task}/>; }
function environment(w?: View) {
    if (!w) return {label:'读取环境', tone:'neutral', next:'请选择项目或myEnv 默认工具链'};
    if (w.empty) return {label:'空项目',tone:'neutral',next:'选择目标版本，确认后创建本目录配置'};
    return ({ready:{label:'已应用',tone:'ready',next:'可以运行；修改版本后先保存并预览'},drifted:{label:'已保存，待同步',tone:'warning',next:'配置或依赖有变化，预览整个环境后应用'},incomplete:{label:'需要修复',tone:'error',next:'已有环境不完整，请先诊断或预览修复'},not_ready:{label:'等待首次安装',tone:'neutral',next:'配置已保存，预览后确认安装'}} as Record<string,{label:string,tone:string,next:string}>)[w.status?.environment||''] || {label:'需要检查',tone:'warning',next:'打开维护，检查当前环境'};
}
function toolStatus(w: View|undefined,t:string) {
    if (!w) return '读取中';
    if (!w.desired[t]) return w.applied[t] ? '待移除' : '未配置';
    if (!w.applied[t]) return '待安装';
    return w.status?.environment==='ready' ? '已应用' : w.status?.environment==='incomplete'?'需要修复':'环境待同步';
}
const externalActions:Record<string,string>={repair:'修复',upgrade:'升级',remove:'删除'};
function catalogScope(request: Request, coverage: string) {
    if (request.tool==='python') return request.provider==='astral'?'当前平台可安装 CPython · 固定 uv 目录':coverage.includes('Windows x64')?'Windows x64 完整运行时 · python.org':'python.org 发布页目录 · 不能直接安装';
    if (request.tool==='java') return '当前平台 · Eclipse Temurin / HotSpot · 各主版本近期发布';
    if (request.tool==='rust') return '当前平台官方工具链 · 历史版本按渠道和日期查询';
    return '当前平台官方归档 · 预发布版本需明确选择';
}
function catalogMatches(row:any,search:string,tool:string) {
    const text=search.trim().toLowerCase(); if(!text)return true;
    const fields=[row.version,row.runtime_version,row.display_version,row.provider].join(' ').toLowerCase();
    return text.split(/\s+/).every(token=>{
        if(tool==='java') {
            if(token==='java'||token==='jdk') return true;
            const match=token.match(/^(?:java|jdk)?-?(1\.8|[0-9]{1,2}|8u)$/);
            if(match) return row.major===(match[1]==='1.8'||match[1]==='8u'?8:Number(match[1]));
        }
        return fields.includes(token);
    });
}
function App() {
    const [managerChoices, ManagerChoices] = useState<ManagerEntry[]>([]);
    const quickFocus=useRef<HTMLElement|null>(null);
    const [packagePanel, PackagePanel] = useState<{context:PackageContext;items:PackageItem[];mode:'manage'|'store'}>();
    const packagePanelRef=useRef(packagePanel), packageDirty=useRef(false);
    packagePanelRef.current=packagePanel;
    const [taskOpen, TaskOpen] = useState(false), [submitting, Submitting] = useState(''), [runTask, RunTask] = useState<Task>(), [runHistory, RunHistory] = useState<Task[]>([]), [limit, Limit] = useState(80);
    const submission = useRef(false), pageRef = useRef('versions'), frozenSave = useRef<Request | undefined>(undefined), localTaskID = useRef(0);
    const [page, Page] = useState('versions'), [saving, Saving] = useState(false), [initChain, InitChain] = useState<{
        request: Request;
        id: number;
    }>();
    const [snapshot, Snapshot] = useState<InventorySnapshot>(), [onboardPending, OnboardPending] = useState(false), [onboardDirectory, OnboardDirectory] = useState('');
    const inspectionPending = useRef(false), inspectionAfter = useRef<{id:number;key:string}|undefined>(undefined), queryOptions = useRef<HTMLDetailsElement>(null), backPages = useRef<string[]>([]);
    const [installations, setInstallations] = useState<any[]>([]), [inventoryLoaded, InventoryLoaded] = useState(false), [externalPlan, EP] = useState<Task>();
    const [version, V] = useState(''), [theme, T] = useState(() => read('myenv.theme', 'system')), [dir, D] = useState<Dir>({ path: '.', key: '.' }), [global, G] = useState(false), [draft, F] = useState<Draft>({ request: defaults, commands: { ...commands }, tool: 'python' }), [view, W] = useState<View>(), [task, K] = useState<Task>(), [history, H] = useState<Task[]>([]), [modal, M] = useState(''), [message, S] = useState(''), [filter, Q] = useState(''), [recent, R] = useState<Dir[]>(read('myenv.recent', [])), [switching, SW] = useState(false), [catalog, C] = useState<{
        request: Request;
        rows: any[];
        coverage: string;
        checkedAt:number;
    }>(), [preview, P] = useState<{
        id: number;
        task: Task;
        request: Request;
        result: any;
        digest: string;
    }>(), [shown, SH] = useState<Task>();
    const drafts = useRef<Record<string, Draft>>(read('myenv.drafts', {})), draftRef = useRef(draft), key = global ? '@user' : dir.key, currentKey = useRef(key), lastID = useRef(0), latestTask = useRef<Task | undefined>(undefined), epoch = useRef(0), dialog = useRef<HTMLDialogElement>(null), saveDialog = useRef<HTMLDialogElement>(null);
    draftRef.current = draft;
    currentKey.current = key;
    const r = draft.request, tool = draft.tool, busy = active(task) || active(runTask) || switching || !!submitting;
    const contextName = global ? (['machine','packages'].includes(page) ? '本机环境' : 'myEnv 默认工具链') : dir.path.split(/[\\/]/).filter(Boolean).pop() || dir.path;
    pageRef.current = page;
    function closeOptions() { if (queryOptions.current) queryOptions.current.open=false; }
    function navigate(next: string) { closeOptions(); if (pageRef.current!==next) backPages.current.push(pageRef.current); SH(undefined); TaskOpen(false); Page(next); }
    function goBack() { closeOptions(); SH(undefined); TaskOpen(false); Page(backPages.current.pop() || (global?'machine':'versions')); }
    function skipOnboarding() { save('myenv.onboarding.v1',true); OnboardPending(false); inspectionPending.current=false; M(''); }
    function rememberInspection() { if(snapshot&&!persistInspection(snapshot,key)){S('本地记录空间不足，检查结果仍可在本次窗口查看。请稍后重试保存。');return;} skipOnboarding(); S('检查记录已保存；以后可以在本机环境或维护中重新检查'); }
    async function beginInspection() {
        const target=onboardDirectory; M(''); inspectionPending.current=true; OnboardPending(true);
        await selectProject(target||dir.path,!target,true);
    }
    function inspectHere() { start('inventory'); }
    function manageInstallation(row:Installation) { change({id:row.id,...(r.id===row.id?{}:{manager:'',externalAction:'repair'})}); navigate('external'); }
    function openPackages(context:PackageContext,items:PackageItem[],mode:'manage'|'store'='manage') { SH(undefined); TaskOpen(false); PackagePanel({context,items,mode}); }
    function closePackages() {
        PackagePanel(undefined);
        const previous=quickFocus.current;quickFocus.current=null;
        if(previous)requestAnimationFrame(()=>{
            const label=previous.getAttribute('aria-label');
            const target=previous.getClientRects().length?previous:Array.from(document.querySelectorAll<HTMLButtonElement>('.tool-shortcuts button')).find(button=>button.getAttribute('aria-label')===label&&button.getClientRects().length);
            if(target?.isConnected&&!target.closest('[inert]'))target.focus();
        });
        if(packageDirty.current&&!active(latestTask.current)) {packageDirty.current=false; void start('inventory');}
    }
    function dismissModal() { if(modal==='tool-locations')quickFocus.current=null;modal==='onboard'?skipOnboarding():M(''); }
    function openChosenManager(entry:ManagerEntry) { dialog.current?.close();M('');requestAnimationFrame(()=>openPackages(entry.context,[entry.item])); }
    const quickManagers=inspectedManagers(snapshot?.result,tool,!global);
    function chooseManager(name:string) {
        if(busy)return;
        quickFocus.current=document.activeElement as HTMLElement;
        const choices=quickManagers.filter(entry=>entry.name===name);
        if(choices.length===1)openPackages(choices[0].context,[choices[0].item]);
        else if(choices.length>1){ManagerChoices(choices);M('tool-locations');}
    }
    function showTools() {navigate('packages');if(!snapshot)void start('inventory');}
    function managerLinks(placement:'sidebar'|'content') {return <ToolShortcuts tool={tool} entries={quickManagers} busy={busy} checked={!!snapshot} placement={placement} onChoose={chooseManager} onMore={showTools}/>;}
    async function submitPackages(action:string,payload:PackageSubmitPayload) { const id=await start(action,payload); if(!id)return; const current=await api.Current(); accept(current); return current.id===id?current:undefined; }
    useEffect(()=>{
        const outside=(e:PointerEvent)=>{ if(queryOptions.current && !queryOptions.current.contains(e.target as Node)) closeOptions(); };
        const keyboard=(e:KeyboardEvent)=>{ if(e.key==='Escape'&&queryOptions.current?.open) {e.preventDefault();closeOptions();queryOptions.current.querySelector('summary')?.focus();} else if(e.altKey&&e.key==='ArrowLeft'&&!modal&&!saving&&!taskOpen&&!packagePanel) {e.preventDefault();goBack();} };
        document.addEventListener('pointerdown',outside); window.addEventListener('keydown',keyboard);
        return ()=>{document.removeEventListener('pointerdown',outside);window.removeEventListener('keydown',keyboard);};
    },[modal,saving,taskOpen,global,packagePanel]);
    function closeTask() { TaskOpen(false); requestAnimationFrame(() => document.querySelector<HTMLButtonElement>('.detail-toggle')?.focus()); }
    const change = (v: Partial<Request>) => F(d => ({ ...d, request: { ...d.request, ...v } }));
    useEffect(() => {
        const media = matchMedia('(prefers-color-scheme: dark)');
        const apply = () => {
            const actual = theme === 'system' ? (media.matches ? 'dark' : 'light') : theme;
            document.documentElement.dataset.theme = actual;
            document.documentElement.style.colorScheme = actual;
            api.SetAppearance(theme, contextName + ' — myEnv').catch(e => S('窗口外观更新失败：' + String(e)));
        };
        apply(); save('myenv.theme', theme); media.addEventListener('change', apply);
        return () => media.removeEventListener('change', apply);
    }, [theme, contextName]);
    useEffect(() => {
        if (!message) return;
        const timer = setTimeout(() => S(''), 6000);
        return () => clearTimeout(timer);
    }, [message]);
    useEffect(() => {
        const dismiss = (event: KeyboardEvent) => { if (event.key === 'Escape' && taskOpen && !modal && !saving) closeTask(); };
        window.addEventListener('keydown', dismiss);
        return () => window.removeEventListener('keydown', dismiss);
    }, [taskOpen, modal, saving]);
    useEffect(() => { drafts.current[key] = draft; save('myenv.drafts', drafts.current); }, [key, draft]);
    useEffect(() => {
        const e = dialog.current;
        if (modal && !e?.open)
            e?.showModal();
        if (!modal && e?.open)
            e.close();
    }, [modal]);
    useEffect(() => { if (saving)
        saveDialog.current?.showModal();
    else
        saveDialog.current?.close(); }, [saving]);
    function accept(t: Task) {
        if (t.id < lastID.current || t.id === latestTask.current?.id && ((!active(latestTask.current) && active(t)) || (t.updatedAt || 0) < (latestTask.current.updatedAt || 0)))
            return;
        const newResult = t.id !== latestTask.current?.id || t.state !== latestTask.current?.state;
        lastID.current = t.id;
        latestTask.current = t;
        const d = draftRef.current.request;
        if (t.request.global !== d.global || (!d.global && t.request.directory !== d.directory))
            return;
        P(p => p && p.id !== t.id ? undefined : p);
        K(t);
        if (newResult && !active(t) && t.request.action==='package-apply') {
            packageDirty.current=true;
            if(!packagePanelRef.current) { packageDirty.current=false; queueMicrotask(()=>start('inventory')); }
        }
        if (newResult && t.state === 'succeeded' && t.request.action === 'inventory') { const record={result:t.result,checkedAt:t.finishedAt||Date.now()}; Snapshot(record); if(!inspectionPending.current) persistInspection(record,currentKey.current); setInstallations(t.result?.installations || []); InventoryLoaded(true); }
        if (newResult && !active(t) && inspectionAfter.current?.id===t.id && inspectionAfter.current.key===currentKey.current) {
            inspectionAfter.current=undefined;
            if(t.state==='succeeded') queueMicrotask(()=>start('inventory'));
        }
        if (newResult && t.state === 'confirm') TaskOpen(true);
        if (newResult && t.state==='succeeded' && t.request.action==='external' && t.request.apply) InventoryLoaded(false);
        if (newResult && t.state==='succeeded' && t.request.action==='external' && !t.request.apply) { EP(t); if (pageRef.current==='planning') { TaskOpen(false); Page('external-plan'); } }
        if (t.view)
            W(t.view);
        if (t.state === 'succeeded' && t.request.action === 'versions')
            C({ request: t.request, rows: t.result?.releases || [], coverage: t.result?.coverage || '', checkedAt:t.finishedAt||Date.now() });
        if (t.state === 'succeeded' && (t.request.action === 'desired' || t.request.action === 'sync' && t.request.preview || t.request.action === 'clean' && !t.request.apply))
            { P({ id:t.id, task:t, request:t.request, result:t.result, digest:t.view?.digest || '' }); if(newResult && pageRef.current==='planning') { TaskOpen(false); Page('preview'); } }
        if (newResult && !active(t))
            api.History().then(H).catch(() => { });
    }
    useEffect(() => {
        let alive = true;
        const off = window.runtime.EventsOn('task', t => {
            if (alive)
                accept(t);
        });
        api.Version().then(V);
        api.Current().then(t => {
            if (alive && t.id)
                accept(t);
        });
        selectProject('.', false).then(()=>{if(!read('myenv.onboarding.v1',false)) M('onboard');});
        return () => { alive = false; off(); };
    }, []);
    useEffect(() => {
        if (initChain && task?.id === initChain.id && task.request.action === 'init' && !active(task)) {
            const frozen = initChain.request;
            InitChain(undefined);
            if (task.state === 'succeeded')
                start('desired', { ...frozen, action: 'desired' });
        }
    }, [task, initChain]);
    async function saveDesired() {
        Saving(false);
        P(undefined);
        const frozen = frozenSave.current || {...r,selection:r.selection.includes('@')?r.selection:tool+'@'+r.selection};
        frozenSave.current = undefined;
        if (view?.empty && !global) {
            const id = await start('init', frozen);
            if (id)
                InitChain({ request: frozen, id });
        }
        else
            await start('desired', frozen);
    }
    async function confirmPreview() {
        if (!preview || busy || page !== "preview" || shown || preview.id !== task?.id)
            return;
        const p = preview, action = p.request.action === 'desired' ? 'sync' : p.request.action;
        await start(action, { ...p.request, action, preview: false, apply: action === 'clean', expectedDigest: p.digest });
    }
    async function selectProject(path: string, user = false, inspect = false) {
        if (active(latestTask.current) || active(runTask) || submission.current)
            return;
        SW(true);
        const e = ++epoch.current;
        try {
            const next = await api.NormalizeDirectory(path);
            if (e !== epoch.current)
                return;
            drafts.current[currentKey.current] = draftRef.current;
            const k = user ? '@user' : next.key, old = drafts.current[k] || { request: defaults, commands: { ...commands }, tool: 'python' }, d = { ...old, request: { ...old.request, directory: next.path, global: user } };
            draftRef.current = d;
            currentKey.current = k;
            D(next);
            G(user);
            F(d);
            W(undefined);
            TaskOpen(false); RunTask(undefined); Limit(80);
            const stored=read<InventorySnapshot|undefined>('myenv.inventory.'+k,undefined); Snapshot(stored); InventoryLoaded(!!stored); setInstallations(stored?.result.installations||[]); backPages.current=[];
            EP(undefined);
            C(undefined);
            P(undefined);
            S('');
            M('');
            Page(user?'machine':inspect&&['node','python'].includes(d.tool)?'packages':'versions');
            SH(undefined);
            K(undefined);
            Saving(false);

            if (!user) {
                const list = [next, ...recent.filter(x => x.key !== next.key)].slice(0, 12);
                R(list);
                save('myenv.recent', list);
            }
            const id=await api.Start({ ...d.request, action: 'workbench' });
            inspectionAfter.current=inspect?{id,key:k}:undefined;
            accept(await api.Current());
        }
        catch (e) {
            S(String(e));
        }
        finally {
            SW(false);
        }
    }
    async function start(action: string, extra: Partial<Request> = {}) {
        if (submission.current || active(latestTask.current) || active(runTask)) return;
        submission.current = true; Submitting(action);
        closeOptions(); S(''); SH(undefined); P(undefined); EP(undefined); RunTask(undefined);
        if (action === 'versions') { Page('versions'); TaskOpen(false); }
        else if (action === 'desired' || action === 'init' || action === 'sync' && extra.preview || action === 'clean' && !extra.apply || action === 'external' && !extra.apply) { Page('planning'); TaskOpen(false); }
        else if (action === 'sync') { Page('versions'); TaskOpen(false); }
        else if (action !== 'inventory' && !action.startsWith('package-')) TaskOpen(true);
        try {
            const id = await api.Start({ ...draftRef.current.request, apply: false, preview: false, rebuild: false, expectedDigest: '', ...extra, action });
            accept(await api.Current());
            return id;
        } catch (e) { S(String(e)); if(action.startsWith('package-')) throw e; }
        finally { submission.current = false; Submitting(''); }
    }
    function choose(t: string) { closeOptions(); SH(undefined); TaskOpen(false); Limit(80); F(d => ({ ...d, tool: t, request: { ...d.request, tool: t, selection: '', provider: t === 'python' ? provider(view) : '', major: 0, channel: '', date: '' } })); Q(''); C(undefined); }
    function openExternal() {
        SH(undefined);
        if (externalPlan && externalPlan.id===task?.id) { Page('external-plan'); return; }
        navigate('external');
        if (!inventoryLoaded) start('inventory');
    }
    async function applyExternal() {
        if (!externalPlan || page!=='external-plan' || shown || busy || externalPlan.id!==task?.id) return;
        const p=externalPlan;
        await start('external', {...p.request,apply:true,planTaskId:p.id});
    }
    async function run() {
        if (submission.current || busy) return;
        const request = { ...r, action: 'run' }, startedAt = Date.now(), id = --localTaskID.current;
        let args: string[];
        try {
            args = JSON.parse(draft.commands[tool] || commands[tool]);
            if (!Array.isArray(args) || !args.length || !args[0].trim() || args.some(x => typeof x !== 'string')) throw Error('请输入程序和有效的字符串参数数组');
        } catch (e) { S(String(e)); return; }
        const running: Task = { id, request, state: 'running', phase: '正在启动外部终端', startedAt, updatedAt: startedAt, result: { args } };
        submission.current = true; RunTask(running); TaskOpen(false); S('');
        let finished: Task;
        try {
            const result = await api.Run(request.directory, request.global, args);
            finished = { ...running, state: 'succeeded', finishedAt: Date.now(), updatedAt: Date.now(), result: { args, handoff: true }, summary: {title:'已交给终端',lines:[result,'请在终端查看实际输出与退出码。']} };
        } catch (e) {
            finished = { ...running, state: 'failed', finishedAt: Date.now(), updatedAt: Date.now(), error: {code:'RUN_START_FAILED',message:String(e),next_action:'检查当前生效环境与运行命令后重试。'} };
        } finally { submission.current = false; }
        RunTask(finished); RunHistory(items => [...items, finished].slice(-16));
    }
    const desired = view?.desired[tool], applied = view?.applied[tool], path = view?.path || '', same = catalog && catalog.request.tool === tool && catalog.request.provider === r.provider && catalog.request.major === r.major && catalog.request.channel === r.channel && catalog.request.date === r.date && catalog.request.preview === r.preview, rows = same ? catalog.rows.filter(v => catalogMatches(v,filter,tool)) : [], state = toolStatus(view,tool), candidate = shown ? history.find(t => t.id === shown.id) || shown : task, visible = candidate && candidate.request.global === global && (global || candidate.request.directory === dir.path) ? candidate : undefined;
    const title = ({ 'tool-locations':`选择 ${managerChoices[0]?.name||'工具'} 安装位置`, onboard:'检查已有开发环境', projects: '项目目录', versions: '选择期望版本', source: '来源详情', sync: '同步预览', clean: '清理预览', doctor: '环境诊断', settings: '任务设置', history: '任务与错误', task: '任务详情', external: '本机与外部环境', rollback: '恢复上一次受管环境', remove: '移除默认工具确认', manage: '默认工具维护' } as Record<string, string>)[modal];
    function currentPage() { SH(undefined); Page(preview && preview.id===task?.id ? 'preview' : externalPlan && externalPlan.id===task?.id ? 'external-plan' : 'result'); }
    const stageNames: Record<string,string> = {waiting:'等待环境锁',resolving:'解析目标版本',backend:'准备安装后端',sdk:'准备工具链',node:'准备 Node.js',python:'准备 Python',dependencies:'安装项目依赖',verifying:'检查新环境',publishing:'应用新环境'};
    const actionNames: Record<string,string> = {versions:'查询版本',workbench:'读取环境',init:'创建配置',desired:'保存并预览',sync:'同步环境',doctor:'诊断',clean:'清理',rollback:'回退',inventory:'发现本机环境',external:'外部管理',remove:'移除默认工具'};
    const taskText = !task || task.request.action==='workbench' ? '选择语言与版本开始' : active(task) ? (task.state==='running' ? stageNames[task.phase] || actionNames[task.request.action] || '正在处理' : labels[task.state]) : task.request.action==='versions' && task.state==='succeeded' ? `已找到 ${task.result?.releases?.length || 0} 个版本` : task.summary?.title || labels[task.state];
    const previewShown = page==='preview' && preview && !shown && preview.id===task?.id;
    const env = environment(view);
    const runReady = task?.state==='succeeded' && task.request.action==='sync' && !task.request.preview;

    const queryHere = task?.request.action === 'versions' && task.request.tool === tool && task.request.provider === r.provider && task.request.major === r.major && task.request.channel === r.channel && task.request.date === r.date && task.request.preview === r.preview;
    const queryProblem = queryHere && task?.state==='failed' ? describeTaskError(task) : undefined;
    const querying = queryHere && active(task) || submitting === 'versions';
    const displayTask = active(task) ? task : runTask || task;
    const scopedHistory = [...history, ...runHistory].filter(t => t.request.global === global && (global || t.request.directory === dir.path)).sort((a,b) => (a.startedAt || 0) - (b.startedAt || 0)).slice(-32);
    const commandArgs = (() => { try { const value = JSON.parse(draft.commands[tool] || commands[tool]); return Array.isArray(value) && value.every(v=>typeof v==='string') ? value as string[] : []; } catch { return []; } })();
    const goRun = () => { if (!applied) { const next = tools.find(t => view?.applied[t]); if (next) choose(next); } navigate('run'); };
    const pendingPlan = preview && preview.id === task?.id || externalPlan && externalPlan.id === task?.id;
    const openPlan = () => { TaskOpen(false); SH(undefined); Page(externalPlan && externalPlan.id === task?.id ? 'external-plan' : 'preview'); };
    const saveSelection = () => {
        if (busy || !r.selection.trim()) return;
        frozenSave.current = {...r, selection: r.selection.includes('@') ? r.selection.trim() : tool + '@' + r.selection.trim()};
        if (view?.empty && !global) Saving(true); else saveDesired();
    };
    const returnToTaskInput = () => {
        const action = task?.request.action;
        navigate(action?.startsWith('package-') ? 'packages' : action === 'inventory' ? (global?'machine':['node','python'].includes(tool)?'packages':'versions') : action === 'external' ? 'external' : action === 'remove' && global ? 'manage' : ['doctor','status','clean','rollback','remove'].includes(action || '') ? 'maintenance' : 'versions');
    };
    const nextTaskAction = task?.state==='succeeded' && task.request.action==='external' && task.request.apply
        ? {label:'刷新本机列表',run:()=>{TaskOpen(false);openExternal();}}
        : task?.state==='succeeded' && ['doctor','rollback','remove'].includes(task.request.action)
            ? {label:'查看版本与环境',run:()=>navigate('versions')}
            : task?.state==='succeeded' && task.request.action==='clean' && task.request.apply
                ? {label:'返回维护',run:()=>navigate('maintenance')} : undefined;
    return <div className="app-window workbench-shell">
      <header className="toolbar environment-bar" inert={!!packagePanel}>
        <div className="scope-toggle scopes" aria-label="作用范围"><button aria-pressed={!global} disabled={busy||onboardPending} onClick={()=>selectProject(dir.path,false)}>项目环境</button><button aria-pressed={global} disabled={busy||onboardPending} onClick={()=>selectProject(dir.path,true)}>本机环境</button></div>
        <div className="project-control"><Icon name="folder"/><button className="project-picker quiet" disabled={busy||onboardPending||global} title={global?'本机环境':dir.path} onClick={()=>M('projects')}>{global?'本机环境':contextName}{!global&&<Icon name="chevron"/>}</button><span className="project-path" title={dir.path}>{global?'已安装版本与工具链':dir.path}</span></div>
        <div className="toolbar-actions"><button className="quiet" onClick={()=>navigate('maintenance')}><Icon name="tool"/><span>维护</span></button><button className="icon-button" aria-label="设置" title="偏好与下载设置" onClick={()=>navigate('settings')}><Icon name="settings"/></button></div>
      </header>
      <div className="workspace workbench-center">
        <aside className="sidebar tool-panel" inert={!!packagePanel}><div className="sidebar-heading">开发环境<span>{global?(snapshot?`${snapshot.result.commands?.filter(c=>c.state==='available').length||0} / 5 已检测`:'等待本机检查'):`${Object.keys(view?.desired||{}).length} / 5 已配置`}</span></div><nav className="tool-list" aria-label="选择语言">{tools.map(t=><div className="language-group" key={t}><button className={'language tool '+t} aria-current={tool===t} aria-pressed={tool===t} disabled={switching} onClick={()=>{choose(t);backPages.current=[];Page(global?'machine':'versions')}}><BrandIcon tool={t}/><span className="language-copy tool-info"><strong>{names[t]}</strong><span title={(view?.applied[t]?.runtime_version || view?.applied[t]?.version || '')+' · '+toolStatus(view,t)}>{global?(snapshot?commandLabel(snapshot.result.commands?.find(c=>c.tool===t)):'未检查'):(view?.applied[t]?.runtime_version || view?.applied[t]?.version || toolStatus(view,t))}</span></span><span className={'language-indicator '+(toolStatus(view,t)==='未配置'?'empty':toolStatus(view,t)==='已应用'?'':'pending')}/></button>{tool===t&&managerLinks('sidebar')}</div>)}</nav><div className="sidebar-bottom"><span className="status-dot"/><span title={env.next}>{global?'本机检查与受管工具链':env.label+' · 按项目生效'}</span></div></aside>
        <main className="main workbench-content" inert={taskOpen||!!packagePanel}>
          {!['machine','versions','run','packages'].includes(page)&&<div className="page-navigation"><button className="text-link" onClick={goBack}><span className="back-icon"><Icon name="arrow"/></span>{page==='external'?'返回上一页':'返回工作区'}</button></div>}
          <header className="main-header content-header"><div><div className="title-line"><h1>{({machine:names[tool],packages:names[tool]+' 包与工具',preview:'确认环境变更',planning:'正在核对变更',maintenance:'环境维护',settings:'偏好与下载设置',external:'管理外部安装','external-plan':'核验外部管理计划',manage:'默认工具维护'} as Record<string,string>)[page]||names[tool]}</h1>{['versions','run'].includes(page)&&<span className={'tag '+(state==='已应用'?'green':state==='未配置'?'':'amber')}>{page==='run'&&applied?'已就绪':state}</span>}</div><p>{page==='machine'?'本机已安装 · 命令路径与版本':page==='packages'?'按所属环境查看依赖与管理工具':page==='versions'?source(tool,r.provider):page==='run'?'使用当前项目已生效的环境':contextName+' · '+env.label}</p></div><div className="header-status"><Icon name="folder"/>{contextName}</div></header>
          {['versions','run','packages','machine'].includes(page)&&<div className="tabs" role="tablist" aria-label="语言工作区">{global&&<button role="tab" aria-selected={page==='machine'} onClick={()=>navigate('machine')}>本机已安装</button>}<button role="tab" aria-selected={page==='versions'} onClick={()=>navigate('versions')}>{global?'myEnv 工具链':'版本与环境'}</button>{['node','python'].includes(tool)&&<button role="tab" aria-selected={page==='packages'} onClick={()=>navigate('packages')}>包与工具</button>}<button role="tab" aria-selected={page==='run'} onClick={()=>navigate('run')}>运行命令</button></div>}
          {['versions','run','packages','machine'].includes(page)&&managerLinks('content')}
          {onboardPending&&<div className="inspection-review" role="status"><div><strong>{task?.request.action==='inventory'&&active(task)?'正在检查已有环境':task?.request.action==='inventory'&&task.state==='succeeded'?'检查完成，确认后保留这份记录':'检查尚未完成'}</strong><span>{onboardDirectory||'本机环境'} · 以后可在维护中重新检查</span></div>{task?.request.action==='inventory'&&task.state==='succeeded'?<button className="primary" onClick={rememberInspection}>保存检查记录</button>:<button disabled={busy} onClick={inspectHere}>重新检查</button>}<button className="quiet" disabled={busy} onClick={skipOnboarding}>稍后再说</button></div>}
          {page==='machine'&&<div className="main-content page-scroll"><LocalEnvironment tool={tool} snapshot={snapshot} busy={busy} onRefresh={inspectHere} onManage={manageInstallation} onToolchain={()=>navigate('versions')} onPackages={()=>navigate('packages')}/></div>}
          {page==='packages'&&<div className="main-content page-scroll"><PackageEnvironment tool={tool} snapshot={snapshot} busy={busy} onRefresh={inspectHere} project={!global} onOpenManage={openPackages}/></div>}
          {page==='versions'&&<>
            <div className="main-content page-scroll version-page">
              <div className="version-overview version-comparison"><div className="overview-unit"><div className="label"><span className="status-dot"/>当前生效</div><div className="overview-value"><strong title={applied?.runtime_version||applied?.version}>{applied?.runtime_version||applied?.version||'未安装'}</strong><span>{applied?'运行使用此版本':'安装后即可使用'}</span></div><p>{global?'myEnv 受管版本；不改变终端默认':applied?'项目仍在使用这个环境':'项目环境与本机 PATH 分别管理'}</p></div><div className="overview-arrow"><Icon name="arrow"/></div><div className="overview-unit desired"><div className="label">配置目标{env.tone==='warning'&&<span className="tag amber">待应用</span>}</div><div className="overview-value"><strong title={desired}>{desired||'待选择'}</strong><span>{desired?'预览并同步后生效':'从下方选择版本'}</span></div><p>{view?.empty?'保存时明确创建当前目录配置':'配置适用于整个环境'}</p></div></div>
              <section aria-label="选择版本"><div className="section-heading"><h3>选择版本</h3><button className="text-link" onClick={()=>M('source')} title="查看来源证据与环境目录">{tool==='python'?r.provider==='astral'?'Astral · uv':'python.org':tool==='java'?'Eclipse Temurin':names[tool]+' 官方'}<Icon name="info"/></button></div>
                <div className="catalog-tools catalog-controls">
                  <label className="search-field"><Icon name="search"/><input aria-label="过滤版本" type="search" placeholder={tool==='java'?'搜索 1.8、Java 21、8u…':'筛选版本'} value={filter} onChange={e=>{Q(e.target.value);Limit(80)}}/></label>
                  {tool==='python'&&<select aria-label="Python 来源" value={r.provider} disabled={busy} onChange={e=>{change({provider:e.target.value,selection:''});C(undefined)}}><option value="astral">Astral / uv</option><option value="python.org">python.org</option></select>}
                  {tool==='java'&&<select aria-label="Java 主版本" value={r.major||0} disabled={busy} onChange={e=>change({major:Number(e.target.value)})}><option value={0}>全部主版本</option>{Array.from(new Set([8,11,17,21,25,...(catalog?.rows||[]).map(v=>Number(v.major)).filter(Boolean),r.major].filter(Boolean))).sort((a,b)=>a-b).map(n=><option key={n} value={n}>{n===8?'Java 8（1.8）':'Java '+n}</option>)}</select>}
                  {tool==='rust'&&<select aria-label="Rust 渠道" value={r.channel} disabled={busy} onChange={e=>change({channel:e.target.value})}><option value="">稳定版</option><option value="beta">beta</option><option value="nightly">nightly</option></select>}
                  <Button disabled={busy} onClick={()=>start('versions',{preview:r.preview})}>{querying?<span className="spinner"/>:<Icon name="refresh"/>}{querying?'查询中':'查询版本'}</Button>
                  <details ref={queryOptions} className="query-options"><summary>选项</summary><div className="query-options-content"><label><input type="checkbox" checked={r.preview} disabled={busy} onChange={e=>change({preview:e.target.checked})}/>包含预发布</label>{tool==='rust'&&<label>工具链日期<input aria-label="Rust 工具链日期" type="date" value={r.date} disabled={busy} onChange={e=>change({date:e.target.value})}/></label>}<span>查询条件仅作用于下一次查询。</span></div></details>
                </div>
                {tool==='java'&&<p className="java-hint">JDK 1.8 就是 Java 8。先按项目要求选择主版本；LTS 表示长期支持，预发布仅在选项中开启。</p>}
                <div className="catalog-state" role="status" aria-live="polite">{querying?<><span className="spinner"/>正在查询 {names[tool]} 版本{same?'，保留上次结果…':'…'}</>:same?<>找到 {catalog.rows.length} 个版本{' · '+catalogScope(catalog.request,catalog.coverage)}{' · '+new Date(catalog.checkedAt).toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'})+' 查询'}</>:<>查询可用版本，或在下方手动指定。</>}</div>
                {queryHere&&task?.state==='failed'&&<div className="inline-message error" role="alert"><Icon name="alert"/><div><strong>{queryProblem?.title || '暂时无法查询版本'}</strong><p className="query-error-detail" title={task.error?.message}>{queryProblem?.detail}</p><div className="message-actions"><Button disabled={busy} onClick={()=>start('versions',{preview:r.preview})}>重试查询</Button><Button onClick={()=>document.querySelector<HTMLInputElement>('[aria-label="期望版本"]')?.focus()}>手动输入</Button><Button onClick={()=>navigate('settings')}>检查设置</Button></div></div></div>}
                <div className="version-list version-options" aria-label="可安装版本" aria-busy={querying}>{rows.slice(0,limit).map(v=><button className="version-row version-option" key={v.version+v.provider} aria-pressed={r.selection===tool+'@'+v.version} disabled={busy||v.kind==='release_page'||!v.url} onClick={()=>change({selection:tool+'@'+v.version})}><span className={'radio '+(r.selection===tool+'@'+v.version?'selected':'')}/><span className="version-number mono">{v.display_version||v.version}</span><span className="version-description" title={v.version}>{v.kind==='release_page'?'发布记录 · 不可安装':tool==='java'?v.version:v.provider}</span>{v.lts&&<span className="tag">LTS</span>}{v.channel==='preview'&&<span className="tag amber">预发布</span>}{r.selection===tool+'@'+v.version?<span className="tag green">已选择</span>:<span className="version-choice">{v.kind==='release_page'?'只读':'选择'}</span>}</button>)}</div>
                {!querying&&!rows.length&&<div className="catalog-empty"><Icon name="download"/><div><strong>{same?'没有匹配的版本':'准备 '+names[tool]+' 环境'}</strong><p>{same?'调整筛选，或直接输入所需版本。':tool==='python'&&r.provider==='astral'?'已有 uv 时可查询 Astral 目录；首次使用可手动输入 3.12，保存预览后再安装。':'点击“查询版本”读取可用目录，也可以手动输入目标版本。'}</p></div></div>}
                {rows.length>limit&&<Button onClick={()=>Limit(n=>n+80)}>显示更多（{Math.min(limit,rows.length)} / {rows.length}）</Button>}
              </section>
            </div>
            <div className="action-bar step-actions"><div className="draft-entry"><label htmlFor="target-version">{r.selection?'目标草稿':'手动指定'}</label><input id="target-version" aria-label="期望版本" placeholder={tool+'@版本'} value={r.selection} disabled={busy} onChange={e=>change({selection:e.target.value})}/></div><div className="action-buttons">{r.selection&&<button className="quiet" disabled={busy} onClick={()=>change({selection:''})}>撤销选择</button>}<Button primary disabled={busy||(!r.selection&&(view?.empty||!view))} onClick={()=>r.selection?saveSelection():pendingPlan?openPlan():start('sync',{preview:true})}>{r.selection?'保存并预览':pendingPlan?'查看待确认变更':'预览变更'}<Icon name="arrow"/></Button></div></div>
          </>}
          {page==='planning'&&<><div className="main-content page-scroll planning-page"><TaskResult task={task}/>{task?.state==='failed'&&<div className="inline-message"><Icon name="info"/><div>当前生效环境仍保留。请调整选择或设置后重新预览。</div></div>}</div><div className="action-bar step-actions"><div className="action-context"><strong>{busy?'正在核对当前目标':'请检查任务结果'}</strong><small>确认执行前不会切换生效环境</small></div><Button onClick={()=>navigate('versions')}>返回选择</Button></div></>}
          {page==='preview'&&previewShown&&<><div className="main-content page-scroll"><div className="preview-target"><Icon name="folder"/><div><strong>{contextName}</strong><small>{global?'myEnv 默认工具链':dir.path}</small></div><span className="grow"/><span className="tag">{global?'myEnv 工具链':'仅当前项目'}</span></div><TaskResult task={preview.task}/><div className="inline-message"><Icon name="info"/><div>{preview.request.action==='clean'?'执行时重新核查引用与运行保护，仅清理允许删除的对象。':'新环境通过校验后才会生效。失败或取消时，继续保留当前环境。'}</div></div></div><div className="action-bar step-actions"><div className="action-context"><strong>{preview.result?.plan?.needs_apply===false?'当前环境无需切换':'以下变更尚未应用'}</strong><small>执行计划与当前预览绑定 · #{preview.id}</small></div><div className="action-buttons"><Button onClick={()=>navigate('versions')}>返回修改</Button><Button primary disabled={busy} onClick={confirmPreview}><Icon name="check"/>{preview.request.action==='clean'?'确认清理':'确认同步'}</Button></div></div></>}
          {page==='preview'&&!previewShown&&<div className="main-content page-scroll"><p>此预览已结束。可以重新预览当前环境。</p><Button disabled={busy} onClick={()=>start('sync',{preview:true})}>重新预览</Button></div>}
          {page==='run'&&<><div className="main-content page-scroll run-page"><CommandEditor value={draft.commands[tool]||commands[tool]} onChange={value=>F(d=>({...d,commands:{...d.commands,[tool]:value}}))}/><div className="section-heading"><h3>将交给外部终端</h3><span className="tag">命令预览</span></div><div className="command-preview"><span>&gt;</span> {commandArgs.map(a=>/\s/.test(a)?JSON.stringify(a):a).join(' ')||'请输入有效的运行命令'}</div>{!applied&&<div className="inline-message warning"><Icon name="info"/><div>当前没有已生效的 {names[tool]}。先选择版本并完成同步，运行不会隐式安装。</div></div>}{runTask&&!active(runTask)&&<div className={'inline-message '+(runTask.error?'error':'')}><Icon name={runTask.error?'alert':'terminal'}/><div><strong>{runTask.error?'终端启动失败':'已交给终端'}</strong><p>{runTask.error?.message||runTask.summary?.lines?.[0]}</p></div></div>}</div><div className="action-bar step-actions"><div className="action-context"><strong>{applied?'使用 '+names[tool]+' '+(applied.runtime_version||applied.version):'需要先准备环境'}</strong><small>在终端中查看实际运行结果</small></div><Button primary disabled={busy||!applied||!commandArgs[0]?.trim()} onClick={run}>{active(runTask)?<span className="spinner"/>:<Icon name="terminal"/>}{active(runTask)?'正在启动终端':'在终端运行'}</Button></div></>}
          {page==='external-plan'&&externalPlan&&externalPlan.id===task?.id&&<><div className="main-content page-scroll"><div className="preview-target"><Icon name="tool"/><div><strong>{externalActions[externalPlan.request.externalAction]} {externalPlan.result?.installation?.tool}</strong><small>核验任务 #{externalPlan.id} · 尚未执行</small></div></div><div className="inline-message warning"><Icon name="alert"/><div>由原管理器执行，将修改外部安装；不属于 myEnv 环境代回滚范围。</div></div><dl className="metadata">{Object.entries({'目标安装':externalPlan.result?.installation?.path,'原管理器':externalPlan.result?.manager_executable,'动作':externalActions[externalPlan.request.externalAction],'工作目录':externalPlan.result?.directory}).map(([k,v])=><div key={k}><dt>{k}</dt><dd>{String(v||'见详细信息')}</dd></div>)}</dl><TaskResult task={externalPlan}/></div><div className="action-bar step-actions"><div className="action-context"><strong>仅执行已核验的原计划</strong><small>目标变化会要求重新确认</small></div><div className="action-buttons"><Button onClick={()=>navigate('external')}>修改选择</Button><Button primary disabled={busy||!!shown} onClick={applyExternal}>确认原管理器执行</Button></div></div></>}
          {!['versions','preview','run','planning','external-plan','machine','packages'].includes(page)&&<div className="main-content page-scroll">{page === 'maintenance' && <section className="maintenance-page"><p className="scope-caption">{contextName} · {env.label}</p><div className="maintenance-grid"><article><h3>检查与同步</h3><p>检查整个环境；保存的版本与依赖一起预览。</p><div className="actions"><Button disabled={busy||view?.empty} onClick={()=>start('sync',{preview:true,locked:r.locked})}>预览整个环境</Button><Button disabled={busy} onClick={()=>start('doctor',{deep:r.deep})}>诊断</Button></div><label><input type="checkbox" checked={r.locked} onChange={e=>change({locked:e.target.checked})}/>锁定同步</label><label><input type="checkbox" checked={r.deep} onChange={e=>change({deep:e.target.checked})}/>深度检查</label></article><article><h3>恢复与释放空间</h3><p>恢复上一次受管环境；不会恢复声明、项目包、代码或系统设置。</p><div className="actions"><Button disabled={busy||view?.empty} onClick={()=>start('sync',{preview:true,rebuild:true})}>修复预览</Button><Button disabled={busy||!path} onClick={()=>M('rollback')}>恢复上一次环境</Button><Button disabled={busy} onClick={()=>start('clean')}>清理预览</Button></div></article><article><h3>已有环境检查</h3><p>查看本机命令、已安装版本和项目包。重新检查会更新当前范围的记录。</p><div className="actions"><Button disabled={busy} onClick={()=>{navigate(global?'machine':['node','python'].includes(tool)?'packages':'versions');inspectHere();}}>重新检查当前范围</Button><Button disabled={busy} onClick={()=>{OnboardDirectory('');M('onboard');}}>选择检查范围</Button></div></article><article><h3>工具链与外部安装</h3><p>外部安装交给已核验的原管理器；myEnv 默认工具链单独维护。</p><div className="actions"><Button disabled={busy} onClick={openExternal}>外部环境管理</Button>{global&&<Button onClick={()=>Page('manage')}>默认工具维护</Button>}</div></article></div></section>}

    {page === 'settings' && <><p className="app-version">myEnv {version}</p><p>主题作用于此桌面；运行时下载镜像和 CA 保存在当前项目 / myEnv 默认工具链草稿，只用于随后创建的任务，不改变已运行任务或系统设置。</p><p>下载镜像与 npm / pip 包仓库分别配置。当前支持下列下载镜像；Java 发行版为 Eclipse Temurin，其他发行版尚未接入。</p><label>主题<select aria-label="主题" value={theme} onChange={e => T(e.target.value)}><option value="system">系统</option><option value="light">浅色</option><option value="dark">暗色</option></select></label>{([['nodeMirror', 'Node 镜像 HTTPS'], ['uvMirror', '固定 uv 下载镜像 HTTPS'], ['pythonMirror', 'Astral Python 下载镜像 HTTPS'], ['certificate', 'CA 文件绝对路径']] as const).map(([k, label]) => <label className="field" key={k}>{label}<input value={r[k]} disabled={busy} onChange={e => change({ [k]: e.target.value })}/></label>)}</>}    {page === 'external' && <><div className="actions"><p>选择环境 → 核验原管理器 → 查看计划并确认</p><Button disabled={busy} onClick={()=>start('inventory')}>刷新本机列表</Button>{externalPlan&&externalPlan.id===task?.id&&<Button onClick={()=>Page('external-plan')}>返回已核验计划</Button>}</div><label className="field">选择已发现的环境<select aria-label="外部环境" value={r.id} disabled={busy} onChange={e=>change({id:e.target.value})}><option value="">请选择</option>{installations.map(i=><option key={i.id} value={i.id} disabled={i.owner!=='external'||!i.actions?.length}>{i.tool} · {i.path}{i.owner!=='external'||!i.actions?.length?'（只读）':''}</option>)}</select></label><p className="muted">只管理后端支持的原管理器；未知来源保持只读。需要交互时请使用 CLI。</p><label className="field">原管理器绝对路径<input aria-label="原管理器绝对路径" value={r.manager} disabled={busy} onChange={e=>change({manager:e.target.value})}/></label><Button disabled={busy} onClick={async()=>{try {const p=await api.OpenManager();if(p)change({manager:p});}catch(e){S(String(e));}}}>选择原管理器文件</Button><label className="field">操作<select aria-label="外部管理动作" value={r.externalAction} disabled={busy} onChange={e=>change({externalAction:e.target.value})}>{Object.entries(externalActions).map(([k,v])=><option key={k} value={k}>{v}</option>)}</select></label><details><summary>安装标识（高级）</summary><input aria-label="安装 ID" value={r.id} onChange={e=>change({id:e.target.value})}/></details><div className="actions"><Button primary disabled={busy||!r.id||!r.manager} onClick={()=>start('external')}>核验计划</Button></div></>}{page === 'manage' && global && <><p>{names[tool]} · 当前 myEnv 默认工具链</p>{['upgrade', 'repair', 'remove'].map(a => <Button key={a} disabled={busy} onClick={() => { if (a === 'upgrade') {
        Page('versions');
        return;
    } if (a === 'repair') {
        start('sync', { preview: true, rebuild: true });
        return;
    } M('remove'); }}>{a === 'upgrade' ? '升级' : a === 'repair' ? '重建修复' : '明确移除工具'}</Button>)}</>}</div>}
        </main>
        <TaskDrawer open={taskOpen} task={displayTask} history={scopedHistory} contextKey={key} onClose={closeTask} onConfirm={(id,allow)=>api.Confirm(id,allow).catch(e=>S(String(e)))} onRun={displayTask===task&&runReady?goRun:undefined} onRetry={displayTask===task&&(task?.state==='failed'||task?.state==='canceled')?returnToTaskInput:undefined} onOpenPreview={pendingPlan?openPlan:undefined} onNext={displayTask===task?nextTaskAction?.run:undefined} nextLabel={nextTaskAction?.label} copyText={value=>api.CopyPath(value)}/>
        {packagePanel&&<PackageManager open context={packagePanel.context} items={packagePanel.items} mode={packagePanel.mode} busy={busy} activity={task} onClose={closePackages} onSubmit={submitPackages} onCancel={()=>{if(task)void api.Cancel(task.id).catch(e=>S(String(e)));}}/>}
      </div>
      <TaskDock detailsDisabled={!!packagePanel} task={displayTask} contextName={contextName} open={taskOpen} onToggle={()=>{PackagePanel(undefined);TaskOpen(value=>!value);}} onCancel={id=>api.Cancel(id).catch(e=>S(String(e)))} onOpenPreview={pendingPlan&&page!=='preview'&&page!=='external-plan'?openPlan:undefined}/>
      {message&&<div className="status-toast" role="status"><span>{message}</span><button className="icon-button" aria-label="关闭通知" onClick={()=>S('')}><Icon name="close"/></button></div>}
    <dialog ref={dialog} onCancel={dismissModal} onClose={() => M('')}><div className="modal-shell"><header className="modal-header"><h2>{title}</h2><Button onClick={dismissModal}>{modal==='onboard'?'暂时跳过':'关闭'}</Button></header><div className="modal-content">
    {modal==='onboard'&&<div className="onboarding"><div className="onboarding-mark"><Icon name="search"/></div><h3>从电脑上已有的环境开始</h3><p>检查 Python、Node.js 等命令的版本和位置，再查看项目依赖。已有安装会作为检查记录呈现。</p><div className="onboard-scope"><strong>检查范围</strong><span>本机环境</span><label><input type="checkbox" checked={!!onboardDirectory} onChange={async e=>{if(!e.target.checked){OnboardDirectory('');return;}try{const p=await api.OpenProject();if(p)OnboardDirectory(p);}catch(e){S(String(e));}}}/>同时检查一个项目目录</label>{onboardDirectory&&<button className="project-option" onClick={async()=>{try{const p=await api.OpenProject();if(p)OnboardDirectory(p);}catch(e){S(String(e));}}}><Icon name="folder"/>{onboardDirectory}</button>}</div><p className="inspection-caption">检查在后台进行，不更改终端默认版本。之后可从“维护”重新打开。</p></div>}
    {modal==='tool-locations'&&<><p>选择这次要管理的位置，预览与操作只绑定所选安装。</p><div className="manager-location-list">{managerChoices.map(entry=><button key={entry.context.path} onClick={()=>openChosenManager(entry)}><strong>{entry.context.label} · {entry.version}</strong><code>{entry.context.path}</code></button>)}</div></>}
    {modal === 'projects' && <><Button disabled={busy} onClick={async () => { try {
        const p = await api.OpenProject();
        if (p)
            await selectProject(p);
    }
    catch (e) {
        S(String(e));
    } }}>选择系统文件夹</Button><h3>最近项目</h3>{recent.map(p => <button className="project-option" key={p.key} disabled={busy} onClick={() => selectProject(p.path)}>{p.path}</button>)}</>}

    {modal==='source'&&<><h3>{source(tool,r.provider)}</h3><dl className="metadata">{Object.entries({'目标平台':view?.platform,'期望来源':desired,'生效来源':source(tool,provider(view),applied),'安装后端':applied?.backend,'锁定版本':applied?.version,'实际锁 URL':applied?.url||'锁中未记录 URL','制品 SHA256':applied?.sha256||'未记录；版本证据不等于制品摘要校验','签名验证':'当前后端没有提供签名验证证据','下一任务镜像':tool==='node'?r.nodeMirror||'沿用 CLI 配置 / 官方':tool==='python'?r.pythonMirror||'沿用 CLI 配置 / 上游':'当前不支持此工具自定义镜像'}).map(([k,v])=><div key={k}><dt>{k}</dt><dd>{v||'未配置'}</dd></div>)}</dl><details><summary>期望锁与生效锁原始证据</summary><pre>{JSON.stringify({desired:view?.locked[tool],applied},null,2)}</pre></details><p className="path-value">{path||'尚无生效环境目录'}</p><div className="actions"><Button disabled={!path} onClick={()=>api.RevealPath(path).catch(e=>S(String(e)))}>在文件管理器打开</Button><Button disabled={!path} onClick={()=>api.CopyPath(path).then(()=>S('已复制当前环境目录')).catch(e=>S(String(e)))}>复制路径</Button></div></>}
    {modal === 'remove' && <><p>将 {names[tool]} 移出当前 myEnv 默认工具链并应用；其他项目和外部安装不受影响。旧代继续受回滚与清理保护。</p><Button disabled={busy} onClick={()=>{M('');start('remove');}}>确认移除默认工具</Button></>}
    {modal === 'rollback' && <><p>恢复之前保留的一代 myEnv 工具链。声明和锁文件、npm 项目包、全局包与系统 PATH 都不会随之回退；这不是整套项目备份。恢复后可以显式使用旧代，或重新预览当前声明。</p><Button disabled={busy} onClick={() => { M(''); start('rollback'); }}>确认恢复上一次环境</Button></>}
    </div>{modal==='onboard'&&<footer className="modal-footer"><Button primary disabled={busy} onClick={beginInspection}>开始检查<Icon name="arrow"/></Button></footer>}</div></dialog>
    <dialog ref={saveDialog} className="save-confirmation" onCancel={() => Saving(false)} aria-label="保存配置确认"><div><h3>{view?.empty && !global ? '确认创建本目录配置' : '确认保存声明'}</h3><p className="path-value">{global ? 'myEnv 默认工具链' : dir.path}</p><p>{r.selection} · {r.provider}。将写入配置并预览整个环境，尚不安装。</p><div className="actions"><Button primary onClick={() => saveDesired()}>确认保存并预览</Button><Button onClick={() => Saving(false)}>返回选择</Button></div>{modal==='onboard'&&<footer className="modal-footer"><Button primary disabled={busy} onClick={beginInspection}>开始检查<Icon name="arrow"/></Button></footer>}</div></dialog>

    </div>;
}
createRoot(document.getElementById('root')!).render(<App />);
