import type { Metadata } from 'next';
import './globals.css';
import './docs.css';
export const metadata: Metadata = {title:{default:'myEnv 使用文档',template:'%s · myEnv'},description:'myEnv 中文使用指南：Windows GUI、终端 TUI、CLI、开发环境与包管理，配真实界面截图。'};
export default function RootLayout({children}:{children:React.ReactNode}) {return <html lang="zh-CN"><body>{children}</body></html>;}
