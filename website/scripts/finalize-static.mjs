import {cpSync,copyFileSync,existsSync,mkdirSync,readdirSync,readFileSync,writeFileSync} from 'node:fs';
import {join} from 'node:path';
// The pinned exporter emits .html files. GitHub Pages directory URLs need index.html.
const root='dist/client';
const prefix=process.env.NEXT_PUBLIC_BASE_PATH||'';
if(prefix&&!/^\/(?:[\w.-]+\/?)+$/.test(prefix))throw new Error('Invalid Pages base path');
if(prefix.split('/').some(s=>s==='.'||s==='..'))throw new Error('Invalid Pages path segment');
// Pages already mounts the output at the repository prefix. Vinext also nests
// prefixed assets on disk; copy that asset tree to the actual Pages root.
if(prefix)cpSync(join(root,prefix.slice(1),'_next'),join(root,'_next'),{recursive:true});
for(const file of readdirSync(join(root,'guide'))){
 if(!file.endsWith('.html'))continue;
 const dir=join(root,'guide',file.slice(0,-5));mkdirSync(dir,{recursive:true});
 copyFileSync(join(root,'guide',file),join(dir,'index.html'));
}
writeFileSync(join(root,'.nojekyll'),'');
let checked=0;
for(const file of [join(root,'index.html'),...readdirSync(join(root,'guide')).filter(n=>n.endsWith('.html')).map(n=>join(root,'guide',n))]){
 let html=readFileSync(file,'utf8');
 if(prefix){html=html.replaceAll('href="/favicon.svg"',`href="${prefix}/favicon.svg"`);writeFileSync(file,html);}
 if(file!==join(root,'index.html'))copyFileSync(file,file.replace(/\.html$/,'/index.html'));
 if(!html.includes('myEnv')||html.includes('Untitled site'))throw new Error(`Invalid page: ${file}`);
 for(const match of html.matchAll(/(?:href|src)="([^"]+)"/g)){
  let target=match[1].split(/[?#]/)[0];
  if(!target.startsWith('/')||target.startsWith('//'))continue;
  if(prefix&&!target.startsWith(prefix+'/'))throw new Error(`Missing repository prefix: ${target}`);
  target=decodeURIComponent(target.slice(prefix.length)).replace(/^\//,'');
  if(target.endsWith('/')||!target)target+='index.html';
  if(!existsSync(join(root,target)))throw new Error(`Missing local link: ${file} -> ${target}`);
  checked++;
 }
}
console.log(`Static docs ready: 11 pages; ${checked} internal asset/page references checked.`);
