import type { Metadata } from 'next';
import './globals.css';
import './docs.css';
export const metadata: Metadata = {title:{default:'myEnv 使用文档',template:'%s · myEnv'},description:'myEnv 中文使用指南、真实安装演示和 Node/Python 环境管理参考。'};
export default function RootLayout({children}:{children:React.ReactNode}) {return <html lang="zh-CN"><body>{children}</body></html>;}
