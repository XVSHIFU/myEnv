import React from 'react';
import type { Task, Lock } from './model';
import { Icon } from './Icon';
const names: Record<string,string> = {python:'Python',node:'Node.js',java:'Java',go:'Go',rust:'Rust'};
export function describeTaskError(task?: Task) {
    const error = task?.error;
    if (!error) return undefined;
    if (task.request.certificate && error.message.toLowerCase().includes(task.request.certificate.toLowerCase())) return {title:'无法读取指定的 CA 文件',detail:'检查文件路径，或在设置中清空自定义 CA 后重试。'};
    const next: Record<string,string> = {
        'Run myenv --help.':'检查输入，或使用 myenv --help 查看命令说明。',
        'Check project input files and retry.':'检查项目配置后重试。',
        'Check that the project directory exists and is accessible, then retry.':'检查相关路径是否存在且可访问，然后重试。',
        'Check the reported cause and retry myenv sync; the prior applied generation is retained.':'检查上述原因后重新预览同步；上一生效环境仍保留。',
    };
    return {title:error.message,detail:next[error.next_action] || error.next_action};
}
const phases: Record<string,string> = {waiting:'等待环境锁',resolving:'正在解析目标版本',backend:'正在准备安装后端',sdk:'正在准备工具链',node:'正在准备 Node.js',python:'正在准备 Python',dependencies:'正在安装项目依赖',verifying:'正在检查新环境',publishing:'正在应用新环境'};
export function TaskResult({task}: {task?: Task}) {
    if (!task) return <p className="muted">任务结果将在这里显示。</p>;
    const pending = ['running','confirm','canceling'].includes(task.state);
    const preview = task.request.action === 'desired' || task.request.action === 'sync' && task.request.preview;
    const plan = task.result?.plan;
    const planned: Record<string,Lock> = {...plan?.tools};
    if (plan?.node?.version) planned.node = plan.node;
    if (plan?.python?.version) planned.python = plan.python;
    const changes = new Map(task.summary?.changes?.map(c=>[c.tool,c]));
    const rows = Object.keys({...task.view?.applied,...task.view?.desired,...planned}).sort().map(tool=>{
        const from = task.view?.applied[tool]?.version || '未安装';
        const want = task.view?.desired[tool];
        const to = planned[tool]?.version || changes.get(tool)?.to || (want ? want + '（待解析）' : '移除');
        const action = to === '移除' ? '移除' : from === '未安装' ? '安装' : task.request.rebuild ? '重建' : changes.has(tool) ? from === to ? '更新来源' : '切换' : '保持不变';
        return {tool,from,to,action};
    });
    const canceled = task.state === 'canceled';
    const problem = describeTaskError(task);
    return <section className="task-result">
        {pending ? <div className="pending-result"><span className="spinner"/><div><h3>{task.state==='confirm'?'等待本次构建许可':task.state==='canceling'?'取消已请求，正在收尾':phases[task.phase] || '正在核对当前环境'}</h3><p>{task.state==='confirm'?task.phase:'完成后显示真实结果。你可以在底部查看进展或取消任务。'}</p></div></div> : canceled ? <><h3>任务已取消</h3><p>{task.request.action==='package-apply'?'已完成的包操作不会自动回退，请核对逐项结果并刷新列表。':task.request.action==='external'?'重新核验外部安装状态后再操作。':'当前生效环境保持不变。'}</p></> : problem ? <div className="inline-message error" role="alert"><Icon name="alert"/><div><strong>{problem.title}</strong><p>{problem.detail}</p></div></div> : preview && plan ? <>
            <table className="change-table"><thead><tr><th>环境</th><th>当前生效</th><th>配置目标</th><th>操作</th></tr></thead><tbody>{rows.map(row=><tr key={row.tool} className={row.action!=='保持不变'?'changed':''}><td><strong>{names[row.tool]||row.tool}</strong></td><td className="mono">{row.from}</td><td className="mono">{row.to}</td><td><span className={'tag '+(row.action!=='保持不变'?'green':'')}>{row.action}</span></td></tr>)}</tbody></table>
            <p className="preview-footnote">{plan.needs_apply ? rows.every(row=>row.action==='保持不变')?'工具版本不变；配置、依赖或重建需求仍需同步。':'这是整个环境的变更预览，确认后开始准备。':'当前环境无需切换。'}</p>
        </> : task.request.action==='versions' ? <><h3>找到 {task.result?.releases?.length || 0} 个 {names[task.request.tool]} 版本</h3><p className="result-line">查询条件和原始目录保留在技术信息中。返回版本页可筛选和选择。</p></> : <><h3>{task.summary?.title || '任务已结束'}</h3>{task.summary?.lines?.map((line,i)=><p className="result-line" key={i}>{line}</p>)}</>}
        {task.request.action==='package-apply'&&!pending&&task.result?.outcomes?.length>0&&<table className="change-table"><thead><tr><th>包</th><th>核验结果</th><th>版本</th></tr></thead><tbody>{task.result.outcomes.map((row:any)=><tr key={row.name}><td>{row.name}</td><td>{({applied:'已应用',removed:'已删除',changed:'已变化',not_applied:'未达到目标',unverified:'未能核验'} as Record<string,string>)[row.state]||row.state}{row.problem&&<small>{row.problem}</small>}</td><td>{row.version||'—'}</td></tr>)}</tbody></table>}<details className="technical"><summary>技术信息 · 任务 #{task.id}</summary><p>{task.request.package?(task.request.package.target.root||task.request.package.target.manager_path||task.request.package.target.interpreter):task.request.global?'myEnv 默认工具链':task.request.directory}</p><pre>{JSON.stringify({request:task.request,result:task.result,error:task.error},null,2)?.slice(0,100000)}</pre></details>
    </section>;
}
export function CommandEditor({ value, onChange }: {
    value: string;
    onChange: (value: string) => void;
}) {
    let args: string[] = [];
    let valid = true;
    try {
        args = JSON.parse(value);
        if (!Array.isArray(args) || args.some(a => typeof a !== 'string')) {
            args = [];
            valid = false;
        }
    }
    catch {
        valid = false;
    }
    const update = (next: string[]) => onChange(JSON.stringify(next));
    return <div className="command-editor">
        <label className="field-group"><span className="form-label">程序<span>使用当前生效环境</span></span><input aria-label="运行程序" disabled={!valid} value={args[0] || ''} onChange={e => update([e.target.value, ...args.slice(1)])}/></label>
        <label className="field-group"><span className="form-label">参数<span>每行一个，无需额外加引号</span></span><textarea rows={3} aria-label="运行参数" disabled={!valid} value={args.slice(1).join('\n')} onChange={e => update([args[0] || '', ...(e.target.value === '' ? [] : e.target.value.split('\n'))])}/></label>
        <details className="precision-args"><summary>高级：精确参数数组</summary><textarea rows={3} aria-label="命令参数数组" value={value} onChange={e => onChange(e.target.value)}/>{!valid && <p className="error">需要 JSON 字符串数组；修正后恢复普通编辑。</p>}</details>
    </div>;
}
