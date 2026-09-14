import React, { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Icon } from './Icon';
import { TaskResult } from './Result';
import type { Task } from './model';

const activeStates = new Set(['running', 'confirm', 'canceling']);
const syncActions = new Set(['sync', 'desired', 'use', 'upgrade', 'repair']);
const phaseNames: Record<string, string> = {
    waiting: '等待环境访问', backend: '准备后端', resolving: '解析版本',
    sdk: '准备工具链', node: '准备 Node.js', python: '准备 Python',
    dependencies: '同步项目依赖', verifying: '校验环境', publishing: '切换环境',
};
const actionNames: Record<string, string> = {
    'package-search':'搜索官方包', 'package-catalog':'查询官方包版本', 'package-plan':'预览包变更', 'package-apply':'管理包与工具',
    sync: '同步环境', desired: '保存并预览环境', use: '调整环境版本',
    upgrade: '更新环境', repair: '修复环境', versions: '查询版本',
    doctor: '检查环境', status: '查看环境状态', init: '创建项目配置',
    rollback: '恢复上一次受管环境', clean: '清理环境', remove: '移除默认工具',
    inventory: '查找本机环境', external: '管理外部环境', workbench: '读取环境', run: '运行命令',
};
const stateNames: Record<string, string> = {
    running: '进行中', confirm: '等待构建许可', canceling: '正在收尾',
    succeeded: '已完成', failed: '未完成', canceled: '已取消',
};
const stages = [
    { label: '准备', phases: ['waiting', 'backend'] },
    { label: '解析版本', phases: ['resolving'] },
    { label: '准备环境', phases: ['sdk', 'node', 'python', 'dependencies'] },
    { label: '校验环境', phases: ['verifying'] },
    { label: '切换环境', phases: ['publishing'] },
];

function titleFor(task: Task) {
    const title = actionNames[task.request.action] || task.request.action;
    return isPreview(task) && task.request.action !== 'desired' ? `${title}预览` : title;
}
function isPreview(task: Task) {
    const request = task.request;
    return request.action === 'desired' || request.action === 'sync' && request.preview || ['clean','external'].includes(request.action) && !request.apply;
}
function phaseFor(task: Task) {
    if (task.request.action === 'run') {
        if (task.state === 'succeeded' && task.result?.handoff) return '已交给终端';
        if (task.state === 'running') return '正在打开终端';
        if (task.state === 'failed') return '终端启动未完成';
    }
    if (task.state !== 'running') return stateNames[task.state] || task.state;
    return phaseNames[task.phase] || task.phase || '正在处理';
}
function duration(task: Task | undefined, now: number) {
    if (!task?.startedAt) return '';
    const end = task.finishedAt || (activeStates.has(task.state) ? now : task.updatedAt || task.startedAt);
    const seconds = Math.max(0, Math.floor((end - task.startedAt) / 1000));
    if (seconds < 60) return `${seconds}秒`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}分${seconds % 60}秒`;
    return `${Math.floor(seconds / 3600)}小时${Math.floor(seconds % 3600 / 60)}分`;
}
function useElapsed(task?: Task, enabled = true) {
    const [now, setNow] = useState(Date.now);
    const active = enabled && !!task?.startedAt && activeStates.has(task.state);
    useEffect(() => {
        if (!active) return;
        setNow(Date.now());
        const timer = window.setInterval(() => setNow(Date.now()), 500);
        return () => window.clearInterval(timer);
    }, [active, task?.id]);
    return duration(task, enabled ? now : task?.updatedAt || now);
}
function bytes(value: number) {
    const amount = Number.isFinite(value) ? Math.max(0, value) : 0;
    if (amount < 1024) return `${Math.floor(amount)} B`;
    const unit = Math.min(4, Math.floor(Math.log2(amount) / 10));
    return `${(amount / 1024 ** unit).toFixed(1)} ${['B', 'KiB', 'MiB', 'GiB', 'TiB'][unit]}`;
}
function downloadFor(task?: Task) {
    const progress = task?.state === 'running' ? task.progress : undefined;
    if (progress?.kind !== 'download') return undefined;
    const completed = Math.max(0, Number.isFinite(progress.completed) ? progress.completed : 0);
    const total = progress.total && Number.isFinite(progress.total) && progress.total > 0 ? progress.total : undefined;
    return { ...progress, completed, total, percent: total ? Math.min(100, Math.floor(completed / total * 100)) : undefined };
}
function Meter({ task, detail = false }: { task: Task; detail?: boolean }) {
    const download = downloadFor(task);
    const percent = download?.percent;
    return <div className={detail ? 'detail-meter' : 'dock-meter'}>
        <div className={detail ? 'task-detail-line' : 'dock-meter-text'}>
            <span title={download?.label}>{detail && download
                ? `${bytes(download.completed)}${download.total ? ` / ${bytes(download.total)}` : ' · 总大小未知'}`
                : download ? '下载文件' : '任务进行中'}</span>
            <span>{percent !== undefined ? `${detail ? '当前文件 ' : ''}${percent}%` : ''}</span>
        </div>
        <div className={`track${percent === undefined ? ' indeterminate' : ''}`} role="progressbar"
            aria-label={download ? '当前文件下载进度' : '当前阶段正在处理'}
            aria-valuemin={0} aria-valuemax={100} aria-valuenow={percent}>
            <span style={percent === undefined ? undefined : { width: `${percent}%` }} />
        </div>
        {detail && download && <p className="muted download-label" title={download.label}>{download.label} · 进度仅表示当前文件下载</p>}
    </div>;
}
function StateTag({ task }: { task: Task }) {
    const tone = task.state === 'failed' ? 'red' : task.state === 'confirm' ? 'amber' : task.state === 'succeeded' ? 'green' : '';
    return <span className={`tag ${tone}`}>{task.request.action === 'run' ? phaseFor(task) : stateNames[task.state] || task.state}</span>;
}

export function TaskDock({ task, contextName, open, onToggle, onCancel, onOpenPreview, detailsDisabled=false }: {
    task?: Task; contextName: string; open: boolean; onToggle: () => void; detailsDisabled?: boolean;
    onCancel: (id: number) => void; onOpenPreview?: () => void;
}) {
    const current = task?.id && task.request.action !== 'workbench' ? task : undefined;
    const elapsed = useElapsed(current);
    const busy = current && activeStates.has(current.state);
    return <footer id="task-dock" className="task-dock task-area" aria-label="当前任务">
        <div className={`dock-icon ${current?.state === 'failed' ? 'error' : current?.state === 'canceled' ? 'cancelled' : ''}`}>
            {current?.state === 'running' || current?.state === 'canceling' ? <span className="spinner" aria-hidden="true" />
                : <Icon name={current?.state === 'failed' ? 'alert' : current?.state === 'canceled' ? 'minus' : current?.state === 'confirm' ? 'info' : 'check'} />}
        </div>
        <div className="dock-copy">
            <div className="dock-title"><strong title={current ? `${titleFor(current)} · 任务 #${current.id}` : undefined}>
                {current ? `${titleFor(current)} · #${current.id}` : '暂无进行中的任务'}</strong>{current && <StateTag task={current} />}</div>
            <div className="dock-subtitle" aria-live="polite" aria-atomic="true">
                {current ? `${phaseFor(current)} · ${contextName}` : `${contextName} · 选择版本并预览，任务进度会显示在这里`}
            </div>
        </div>
        {current?.state === 'running' && <Meter task={current} />}
        {elapsed && <span className="dock-time" aria-label={`已用时 ${elapsed}`}>{elapsed}</span>}
        <div className="dock-buttons">
            {busy && current && current.request.action !== 'run' && <button disabled={current.state === 'canceling'} onClick={() => onCancel(current.id)} aria-label={`取消任务 #${current.id}`}>
                {current.state === 'canceling' ? '收尾中' : '取消任务'}</button>}
            {current?.state === 'succeeded' && isPreview(current) && onOpenPreview && <button onClick={onOpenPreview}>查看变更</button>}
            <button className="quiet detail-toggle" disabled={detailsDisabled} title={detailsDisabled?'请在当前管理抽屉查看进度与结果':undefined} onClick={onToggle} aria-expanded={open} aria-controls="task-panel">
                {open ? '收起详情' : current?.state === 'confirm' ? '查看构建许可' : busy ? '查看进度' : current?.state === 'failed' ? '查看原因' : current ? '查看结果' : '任务记录'} <Icon name="chevron" />
            </button>
        </div>
    </footer>;
}

function StageSequence({ task }: { task: Task }) {
    if (!syncActions.has(task.request.action) || !task.phases?.length) return null;
    const reached = new Set(task.phases);
    const currentPhase = phaseNames[task.phase] ? task.phase : task.phases[task.phases.length - 1];
    const current = stages.findIndex(stage => stage.phases.includes(currentPhase));
    return <div className="stage-list" aria-label="环境处理阶段">
        {stages.map((stage, index) => {
            const observed = stage.phases.some(phase => reached.has(phase));
            const done = observed && (task.state === 'succeeded' || index !== current);
            const status = done ? 'done' : index === current ? task.state === 'failed' ? 'failed' : activeStates.has(task.state) ? 'active' : '' : '';
            return <div key={stage.label} className={`stage-item ${status}`} aria-current={index === current && activeStates.has(task.state) ? 'step' : undefined}>
                <span className="stage-node">{done ? <Icon name="check" /> : index + 1}</span>
                <span>{stage.label}</span>
            </div>;
        })}
    </div>;
}

type PanelTab = 'overview' | 'logs' | 'history';
const tabs: { id: PanelTab; label: string }[] = [{ id: 'overview', label: '概览' }, { id: 'logs', label: '日志' }, { id: 'history', label: '历史' }];

export function TaskDrawer({ open, task, history, contextKey, onClose, onConfirm, onRun, onRetry, onOpenPreview, onNext, nextLabel, copyText }: {
    open: boolean; task?: Task; history: Task[]; contextKey: string; onClose: () => void;
    onConfirm: (id: number, allow: boolean) => void; onRun?: () => void; onRetry?: () => void;
    onOpenPreview?: () => void; onNext?: () => void; nextLabel?: string; copyText: (value: string) => Promise<void>;
}) {
    const [tab, setTab] = useState<PanelTab>('overview');
    const [historyId, setHistoryId] = useState<number>();
    const [decisionId, setDecisionId] = useState<number>();
    const [copyStatus, setCopyStatus] = useState('');
    const contentRef = useRef<HTMLDivElement>(null);
    const tabRefs = useRef<Partial<Record<PanelTab, HTMLButtonElement>>>({});
    const confirmationSeen = useRef<number | undefined>(undefined);
    const logPosition = useRef({ owner: '', top: 0, atBottom: true });
    const historical = historyId !== undefined;
    const shown = historical ? history.find(item => item.id === historyId) : task;
    const elapsed = useElapsed(shown, open && !historical);

    useEffect(() => { setHistoryId(undefined); setTab('overview'); setCopyStatus(''); }, [contextKey]);
    useEffect(() => {
        if (task?.state !== 'confirm') { setDecisionId(undefined); return; }
        if (confirmationSeen.current === task.id) return;
        confirmationSeen.current = task.id;
        setHistoryId(undefined);
        setTab('overview');
        if (open) tabRefs.current.overview?.focus();
    }, [task?.id, task?.state, open]);
    useLayoutEffect(() => {
        if (open) tabRefs.current[tab]?.focus();
        // Streaming events must never move focus. Opening is the only trigger.
    }, [open]);
    useLayoutEffect(() => {
        const host = contentRef.current;
        if (!open || tab !== 'logs' || !host) return;
        const owner = `${contextKey}/${shown?.id || 0}`;
        if (logPosition.current.owner !== owner) logPosition.current = { owner, top: 0, atBottom: true };
        host.scrollTop = logPosition.current.atBottom ? host.scrollHeight : logPosition.current.top;
    }, [open, tab, shown?.id, shown?.log, contextKey]);
    useEffect(() => { setCopyStatus(''); }, [shown?.id]);

    function selectTab(next: PanelTab) {
        setTab(next);
        tabRefs.current[next]?.focus();
    }
    function navigateTabs(event: React.KeyboardEvent<HTMLDivElement>) {
        if (!(event.target instanceof HTMLButtonElement) || event.target.getAttribute('role') !== 'tab') return;
        const index = tabs.findIndex(item => item.id === tab);
        const next = event.key === 'ArrowRight' ? (index + 1) % tabs.length : event.key === 'ArrowLeft' ? (index + tabs.length - 1) % tabs.length : event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : -1;
        if (next < 0) return;
        event.preventDefault();
        selectTab(tabs[next].id);
    }
    function confirm(allow: boolean) {
        if (!shown || historical || shown.id !== task?.id || task.state !== 'confirm' || decisionId === task.id) return;
        setDecisionId(task.id);
        onConfirm(task.id, allow);
    }
    if (!open) return null;
    return <section id="task-panel" className="task-panel" aria-labelledby="task-panel-title" onKeyDown={event => {
        if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); onClose(); }
    }}>
        <div className="panel-header">
            <div className="panel-title"><h2 id="task-panel-title">{historical ? '历史任务' : shown ? titleFor(shown) : '任务记录'}</h2>
                <small>{shown ? `#${shown.id} · ${historical ? '只读记录' : '本次任务'}` : '当前作用范围'}</small></div>
            <button className="icon-button" onClick={onClose} aria-label="收起任务详情"><Icon name="close" /></button>
        </div>
        <div className="panel-tabs" role="tablist" aria-label="任务信息" onKeyDown={navigateTabs}>
            {tabs.map(item => <button key={item.id} ref={node => { if (node) tabRefs.current[item.id] = node; }}
                id={`task-tab-${item.id}`} role="tab" aria-selected={tab === item.id} aria-controls="task-tabpanel"
                tabIndex={tab === item.id ? 0 : -1} onClick={() => selectTab(item.id)}>{item.label}</button>)}
            {historical && <button className="text-link" onClick={() => { setHistoryId(undefined); selectTab('overview'); }}>返回当前任务</button>}
        </div>
        <div id="task-tabpanel" className="panel-content" role="tabpanel" aria-labelledby={`task-tab-${tab}`} tabIndex={0}
            ref={contentRef} onScroll={event => {
                if (tab !== 'logs') return;
                const host = event.currentTarget;
                logPosition.current.top = host.scrollTop;
                logPosition.current.atBottom = host.scrollHeight - host.scrollTop - host.clientHeight < 32;
            }}>
            {tab === 'history' ? history.length ? [...history].sort((a, b) => (b.startedAt || 0) - (a.startedAt || 0)).map(item => <button className="history-row" key={item.id} onClick={() => { setHistoryId(item.id); selectTab('overview'); }}>
                <Icon name={item.state === 'failed' ? 'alert' : item.state === 'canceled' ? 'minus' : 'clock'} />
                <span className="history-copy">{titleFor(item)}<small>#{item.id}{item.startedAt ? ` · ${new Date(item.startedAt).toLocaleString('zh-CN', { hour12: false })}` : ''}</small></span>
                <StateTag task={item} />
            </button>) : <p className="history-empty">当前作用范围还没有任务记录。执行后可在这里查看结果。</p>
                : !shown ? <p className="history-empty">准备好后，从版本选择页开始。执行期间，底部会持续显示当前阶段和取消入口。</p>
                    : tab === 'logs' ? <>
                        <div className="logs-toolbar"><span>{shown.logTruncated ? '日志已达到保留上限，部分内容已截断' : '任务日志 · 最多保留 64 KiB'}</span>
                            <button disabled={!shown.log} onClick={async () => {
                                try { await copyText(shown.log || ''); setCopyStatus('日志已复制'); }
                                catch { setCopyStatus('复制未完成，请选中日志后手动复制'); }
                            }}><Icon name="copy" />复制日志</button></div>
                        {copyStatus && <p className="muted" role="status">{copyStatus}</p>}
                        <div className="log-list"><pre className="logs" id="log-text">{shown.log || '当前任务尚无日志输出。阶段和结果仍会显示在概览中。'}</pre></div>
                    </> : <>
                        <StageSequence task={shown} />
                        <div className="task-detail-line"><strong aria-live="polite" aria-atomic="true">{phaseFor(shown)}</strong>
                            <span>任务 #{shown.id}{elapsed ? ` · ${elapsed}` : ''}</span></div>
                        {shown.state === 'running' && <Meter task={shown} detail />}
                        {activeStates.has(shown.state) ? <div className="task-outcome"><p>{shown.state === 'canceling'
                            ? '取消已请求，正在等待环境收尾。'
                            : shown.state === 'confirm' ? shown.phase || '本次任务需要运行构建代码，等待你的选择。'
                                : shown.request.action === 'run' ? '正在打开独立终端，命令输出与退出结果会保留在终端中。'
                                    : phaseNames[shown.phase] ? `${phaseNames[shown.phase]}，完成后会显示实际结果。` : shown.phase || '正在处理，完成后会显示实际结果。'}</p></div>
                            : shown.state === 'canceled' && shown.request.action !== 'package-apply' ? <div className="task-outcome"><p>任务已取消，已完成收尾。可以检查当前环境后再试。</p></div>
                                : <TaskResult task={shown} />}
                        {historical ? <div className="inline-message"><Icon name="info" /><div>这是只读任务记录。当前任务的操作入口保留在底部。</div></div>
                            : shown.id === task?.id && <div className="task-outcome">
                                {shown.state === 'succeeded' && isPreview(shown) && onOpenPreview && <button onClick={onOpenPreview}>查看待确认变更<Icon name="arrow" /></button>}
                                {shown.state === 'succeeded' && !isPreview(shown) && shown.request.action === 'sync' && onRun && <button className="primary" onClick={onRun}>运行命令<Icon name="arrow" /></button>}
                                {shown.state === 'succeeded' && onNext && <button onClick={onNext}>{nextLabel}<Icon name="arrow" /></button>}
                                {['failed', 'canceled'].includes(shown.state) && onRetry && <button onClick={onRetry}>返回调整<Icon name="back" /></button>}
                            </div>}
                        {(activeStates.has(shown.state) || shown.state === 'canceled' && shown.request.action !== 'package-apply') && <details className="technical"><summary>技术信息</summary>
                            <p>任务 #{shown.id} · {shown.request.global ? 'myEnv 默认工具链' : shown.request.directory}</p>
                            <pre>{JSON.stringify({ request: shown.request, phases: shown.phases, error: shown.error }, null, 2)}</pre>
                        </details>}
                    </>}
        </div>
        {!historical && tab==='overview' && task?.state==='confirm' && <div className="panel-decision">
            <span>{decisionId===task.id?'已提交，等待任务响应':'本次需要构建许可'}</span>
            <button className="primary" disabled={decisionId===task.id} onClick={()=>confirm(true)}>允许本次构建</button>
            <button disabled={decisionId===task.id} onClick={()=>confirm(false)}>拒绝本次构建</button>
        </div>}
    </section>;
}
