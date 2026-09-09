import {DocsShell} from './docs-shell';
import {navigation} from './navigation';
import {articles} from './articles';
export function Guide({slug}:{slug:string}){
 const article=articles[slug],all=navigation.flatMap(g=>g.pages),i=all.findIndex(p=>p.slug===slug),base=process.env.NEXT_PUBLIC_BASE_PATH||'';
 return <DocsShell slug={slug}><main id="article" className="article" tabIndex={-1}><div className="eyebrow">MYENV / {navigation.find(g=>g.pages.some(p=>p.slug===slug))?.title}</div><h1>{article.title}</h1><p className="lead">{article.intro}</p><div className="article-body">{article.body}</div><nav className="pager" aria-label="相邻文档">{i>0?<a href={`${base}/guide/${all[i-1].slug}/`}><small>上一篇</small>{all[i-1].title}</a>:<span/>}{i<all.length-1&&<a href={`${base}/guide/${all[i+1].slug}/`}><small>下一篇</small>{all[i+1].title} →</a>}</nav></main></DocsShell>;
}
