'use client';
import {Sidebar,SidebarContent,SidebarGroup,SidebarGroupLabel,SidebarHeader,SidebarProvider,SidebarTrigger} from '@/components/ui/sidebar';
import {navigation} from './navigation';
import type {ReactNode} from 'react';
export function DocsShell({slug,children}:{slug:string;children:ReactNode}){
 const base=process.env.NEXT_PUBLIC_BASE_PATH||'';
 return <SidebarProvider><a href="#article" className="skip">跳到正文</a><Sidebar><SidebarHeader><a className="brand" href={`${base}/`}><span className="brand-mark">&gt;_</span> myEnv <small>文档</small></a></SidebarHeader><SidebarContent><nav aria-label="文档导航">{navigation.map(group=><SidebarGroup key={group.title}><SidebarGroupLabel>{group.title}</SidebarGroupLabel>{group.pages.map(page=><a className="nav-link" aria-current={slug===page.slug?'page':undefined} key={page.slug} href={`${base}/guide/${page.slug}/`}>{page.title}</a>)}</SidebarGroup>)}</nav><p className="nav-version">0.1.0-rc.1 · 发布候选<br/>Windows / 原生 Linux</p></SidebarContent></Sidebar><div className="reading-area"><header className="topbar"><SidebarTrigger aria-label="展开或收起文档导航"/><span>使用文档</span><span className="language">简体中文</span><a href={`${base}/guide/support/`}>支持范围</a></header>{children}<footer>myEnv 使用文档 · 对应 0.1.0-rc.1 · 更新于 2026-09-09</footer></div></SidebarProvider>;
}
