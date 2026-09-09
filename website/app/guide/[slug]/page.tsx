import {Guide} from '../../guide';
import {articles} from '../../articles';
export function generateStaticParams(){return Object.keys(articles).map(slug=>({slug}));}
export const dynamicParams=false;
export async function generateMetadata({params}:{params:Promise<{slug:string}>}){const {slug}=await params;return {title:articles[slug]?.title||'文档'};}
export default async function Page({params}:{params:Promise<{slug:string}>}){const {slug}=await params;return <Guide slug={slug}/>;}
