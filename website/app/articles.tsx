import type {ReactNode} from 'react';
const base=process.env.NEXT_PUBLIC_BASE_PATH||'';
const Code=({children}:{children:string})=><pre><code>{children}</code></pre>;
const Note=({children}:{children:ReactNode})=><aside className="note">{children}</aside>;
const Shot=({file,caption}:{file:string;caption:string})=><figure><a href={`${base}/images/installation/${file}`} target="_blank" rel="noreferrer"><img src={`${base}/images/installation/${file}`} alt={caption} loading="lazy"/></a><figcaption>{caption} · 点击查看原图</figcaption></figure>;
export const articles:Record<string,{title:string;intro:string;body:ReactNode}>={
 projects:{title:'Node 与 Python 项目',intro:'运行时由 myEnv 准备，项目原生配置仍由对应生态负责。',body:<>
 <h2>Node 项目</h2><p>在 myenv.yaml 中声明 Node 版本。myEnv 安装 Node 并提供 npm/npx，但不会自动替项目执行 npm install。</p><Code>{'schema: 1\ntools:\n  node: "22"'}</Code><Code>{'myenv sync\nmyenv run npm install\nmyenv run npm run dev'}</Code><p>npm install 会按项目定义安装依赖并可能执行生命周期脚本，请在自己信任的项目中使用。</p>
 <h2>Python 项目</h2><p>解释器版本写在 myenv.yaml，依赖继续写在 pyproject.toml，使用原生 uv.lock。myEnv 用固定 uv 后端准备新的 venv。</p><Code>{'schema: 1\ntools:\n  python: "3.12"\npython:\n  project: "."'}</Code><Code>{'myenv sync\nmyenv run python main.py'}</Code><p>依赖组通过 python.groups 显式选择。构建代码需要本次终端确认或 --allow-build，该选项允许以当前用户身份运行构建代码，不提供沙箱。</p>
 <h2>混合项目</h2><p>同一 tools 映射可以同时声明 node 与 python，sync 准备整套环境。已有环境不会被就地覆盖，成功后才切换。</p><Code>{'schema: 1\ntools:\n  node: "22"\n  python: "3.12"'}</Code>
 <h2>参数与路径</h2><Code>{'myenv -C ./project run node hello.js "hello world" 中文'}</Code><p>用户命令之后的参数原样传递。run 不解释 Shell 管道或重定向。项目代码和业务数据不属于环境回滚范围。</p></>},
 versions:{title:'版本、同步与回滚',intro:'声明描述想要的环境，锁文件记录解析结果，活动环境是实际执行的版本。',body:<>
 <h2>修改版本</h2><Code>{'myenv use node@22\nmyenv use python@3.12'}</Code><p>use 修改声明并同步；准备失败会保留修改后的声明，但此前活动环境仍可保留使用。</p>
 <h2>查询与选择版本</h2><Code>{'myenv versions java --major 17\nmyenv versions python --provider python.org\nmyenv versions rust --preview\nmyenv versions rust --channel nightly --date 2026-09-09'}</Code><p>Java 查询 Temurin，Python 按来源查询。Rust 历史 nightly 按日期查询，不宣称列出全部历史包。</p><h2>固定依赖与无变更同步</h2><Code>{'myenv sync --locked --no-input'}</Code><p>已有且匹配的锁文件是前提。--locked 不更新 myenv.lock 或 Python 原生锁。没有变化时复用环境，不必重新安装。</p>
 <h2>检查后再应用</h2><Code>{'myenv sync --dry-run\nmyenv sync'}</Code><p>预览不安装、不修改环境或锁。尚未解析的 Python 选择器会在实际同步时确定。</p>
 <h2>失败后继续工作</h2><Code>{'myenv run --current node --version\nmyenv rollback\nmyenv run --current node --version'}</Code><p>rollback 需要已有上一完整环境。它切换活动引用，不回退声明、锁文件或业务代码。--current 明确允许使用与当前声明不同的已应用环境。</p>
 <h2>重新准备环境</h2><Code>{'myenv sync --rebuild --locked'}</Code><p>按匹配的当前锁准备新环境，不重装共享基础解释器或清空缓存。主/次版本默认选择稳定系列；预览版必须显式选择。</p></>},
 global:{title:'默认工具与已有环境',intro:'项目环境与当前用户默认环境相互独立，系统安装不会自动被接管。',body:<>
 <h2>当前用户的默认 Python</h2><Code>{'myenv use --global python@3.12\nmyenv run --global python --version\nmyenv doctor --global'}</Code><p>--global 指当前用户的独立 profile，不是全机器安装，不需要管理员权限。项目不会隐式继承 profile 中的工具。</p>
 <h2>希望直接输入 python</h2><p>已有 profile 后，在 PowerShell 显式执行：</p><Code>{'myenv shell-init powershell | Out-String | Invoke-Expression'}</Code><p>这会在当前 Shell 定义包装函数，通过 run --global 使用默认工具，不修改系统 PATH 或 Shell 文件。关闭终端可丢弃这些函数；添加工具或移动 myEnv 后重新生成。被覆盖的同名函数不会自动恢复，移除工具后应打开新 Shell。</p>
 <h2>电脑已经有 Node 或 Python</h2><p>myEnv 不自动导入现有安装，也不卸载它们。声明 node/python 并同步时使用受管环境。doctor 显示系统 PATH 与实际运行路径，便于区分。</p><Code>{'myenv doctor'}</Code>
 <h2>系统环境发现与管理</h2><Code>{'myenv list\nmyenv system list --details\nmyenv system doctor\nmyenv system install java@17\nmyenv system upgrade java\nmyenv system repair java\nmyenv system remove java\nmyenv system clean --dry-run'}</Code><p>system 管理当前用户默认环境，不是全机器安装。upgrade 遵守原版本范围；repair 重建整个默认环境；remove 移除声明，旧环境仍按运行与回退保护保留。</p><h2>外部安装</h2><Code>{'myenv system external --help'}</Code><p>使用详细列表的安装 ID 和原管理器绝对路径生成操作计划，默认不修改；只有显式 --apply 执行。Conda remove 会移除整个目标环境及包。未知管理器不能由 myEnv 删除。</p></>},
 troubleshooting:{title:'排障与清理',intro:'先查看状态和路径，再决定是否同步、重建或清理。',body:<>
 <h2>找不到 myenv 命令</h2><p>安装后退出整个终端应用并重开。Windows 运行 Get-Command myenv -All，检查同名函数/别名和 PATH；也可直接用安装目录下的 exe。</p>
 <h2>声明与环境不一致</h2><Code>{'myenv status\nmyenv sync\nmyenv doctor'}</Code><p>如需暂时使用旧活动环境，显式选择 run --current。run 不会隐式帮你下载缺少的工具。</p>
 <h2>检查文件内容</h2><Code>{'myenv doctor --deep'}</Code><p>深度检查比较环境代内容基线，可能较慢。它不检查外部共享运行时内容、Python 字节码缓存或 ACL，也不自动修复。</p>
 <h2>安全清理</h2><Code>{'myenv clean --dry-run\nmyenv clean'}</Code><p>保留当前、上一代和仍受运行/准备保护的环境。无法确认后代进程退出时继续保留保护；不要手动删除租约绕过检查。clean 不接受 --global。</p><Code>{'myenv clean --cache node --dry-run\nmyenv clean --cache uv --dry-run'}</Code><p>缓存清理不卸载共享 Python 解释器。项目 venv 可能引用共享解释器，不要随意手动删除。</p>
 <h2>颜色与进度</h2><p>交互终端自动显示颜色与单行阶段动画，重定向保留普通文本。NO_COLOR=1 关闭装饰；--no-input、--verbose 或 --json 关闭动画。命中已有环境时可能没有准备阶段。用户程序输出保持原样。</p><h2>反馈问题</h2><p>保留 myenv --version、失败命令、完整输出及紧接着读取的 $LASTEXITCODE。可以附原始截图；公开分享前检查是否包含凭据或不希望公开的信息。</p></>},
 commands:{title:'命令速查',intro:'日常只需 init → sync → run，其余命令按需要使用。',body:<>
 <table><thead><tr><th>命令</th><th>作用</th></tr></thead><tbody>{[['myenv / myenv status','只读查看当前项目状态'],['myenv init','识别声明并创建配置，不覆盖已有文件'],['myenv sync','准备并应用新环境'],['myenv use node@22','修改版本声明并同步'],['myenv run <command>','使用已应用环境执行命令'],['myenv rollback','切换到上一完整环境'],['myenv doctor --deep','只读检查路径和内容基线'],['myenv clean --dry-run','预览可清理的项目环境'],['myenv help manual','完整离线手册'],['myenv completion powershell','生成补全脚本'],['myenv shell-init powershell','生成默认工具包装函数']].map(([cmd,desc])=><tr key={cmd}><td><code>{cmd}</code></td><td>{desc}</td></tr>)}</tbody></table>
 <h2>获取单个命令帮助</h2><Code>{'myenv help sync\nmyenv run --help\nmyenv --lang en --help'}</Code><p>参数支持范围以对应命令帮助为准。--global 不是每个命令都接受，--json 不适用于 run/help/completion/shell-init。</p>
 <h2>退出码</h2><p>myEnv 成功为 0，执行或环境错误为 1，用法或配置错误为 2，需要输入为 3，取消通常为 130。用户程序启动后，run 保留程序自己的退出码；Linux 信号终止使用 128 加信号编号。收尾错误不会覆盖已经完成的子程序退出码。</p></>},
 configuration:{title:'配置与自动化',intro:'配置保持可读，自动化依赖稳定的 JSON 与退出码。',body:<>
 <h2>myenv.yaml</h2><Code>{'schema: 1\ntools:\n  node: "22"\n  python: "3.12"\npython:\n  project: "."\n  groups: [dev]\nenv:\n  APP_MODE: "development"'}</Code><p>只声明需要的工具；Python 依赖组必须与项目匹配。python.project 必须是工作区内的相对路径。版本值加引号。env 用于普通字符串，凭据通过调用环境传入。</p>
 <h2>非交互调用</h2><Code>{'myenv sync --locked --no-input --json\nmyenv status --json'}</Code><p>JSON 使用 schema/ok/changed/data/error，错误含 code/message/next_action。显示语言不改变 JSON 合同；诊断写到 stderr。--json 和 --no-input 不等待终端输入。</p>
 <h2>语言设置</h2><Code>{'myenv --lang zh-CN --help\nmyenv --lang en --help'}</Code><p>优先级为 --lang、MYENV_LANG、系统语言、英文回退。auto 在中文系统显示简体中文。参数、路径、错误码和第三方输出不翻译；run 后属于子程序的 --lang 原样传递。</p>
 <h2>镜像与证书</h2><p>MYENV_NODE_MIRROR、MYENV_UV_MIRROR、MYENV_PYTHON_MIRROR 分别指定下载镜像。需要遵循原官方目录布局；Node 新解析会记录来源和摘要，已锁定 URL 不变。uv 的固定版本和摘要验证不变。</p><p>SSL_CERT_FILE 使用绝对路径 PEM 文件；myEnv 下载器不支持 SSL_CERT_DIR。无效证书会报错，不关闭 TLS 校验。更多约束见 myenv help manual。</p></>},
 release:{title:'版本记录与已知限制',intro:'文档依据实际交付与测试记录编写，计划中的能力会明确标注。',body:<>
 <h2>0.1.0-rc.1</h2><ul><li>新增 Java、Go、Rust 与 Python 官网来源。</li><li>系统发现、原管理器受限操作、预览版选择。</li><li>分组列表、中文使用提示、终端颜色与动态阶段进度。</li></ul><h2>历史：0.1.0-dev.2</h2><ul><li>中文帮助、手册与日常提示，支持英文切换。</li><li>status 与裸命令共用状态入口。</li><li>Windows 当前用户安装向导及便携包。</li><li>Windows/Kali 各 35 项定向测试通过；用户实测安装、PATH 与语言切换成功。</li></ul>
 <h2>语言切换实测</h2><Shot file="image-20260909123308407.png" caption="显式切换为英文帮助"/><Shot file="image-20260909123314813.png" caption="切回简体中文帮助"/>
 <h2>当前已知限制</h2><ul><li>旧截图中 help 简介漏翻译已修复；原图继续保留对应 dev.2 的真实状态。</li><li>部分 Windows/Kali 延迟与聚合内存未达到原性能预算，仍是开发版。</li><li>已支持 Temurin JDK；未知外部安装来源不自动接管。</li><li>macOS、WSL、musl 不在本次支持范围。</li><li>Linux 调用者暂停期间无法执行自身取消回调；绝对截止时间由独立监督器执行。</li><li>安装器未签名，尚无公开发行下载地址。</li></ul><p>这些限制不会通过修改截图或改换统计口径来隐藏。后续发行演示应保留原图来源，同时标明对应版本。</p></>},
 quickstart:{title:'快速开始',intro:'安装一个命令，管理项目与用户默认开发环境。',body:<>
 <Note>当前为 0.1.0-rc.1 发布候选，支持 Windows amd64 和原生 Linux amd64/glibc。首次同步需要联网，性能验收仍开放。</Note>
 <h2>1. 安装 myEnv</h2><p>Windows 用户运行安装向导，保留“加入当前用户 PATH”。完成后退出终端应用，重新打开 PowerShell。</p><Code>{'myenv --version\nmyenv --help'}</Code><p>能输出版本号，就可以继续。具体步骤见<a href={`${base}/guide/installation/`}>安装与真实演示</a>。</p>
 <h2>先安装一个默认 JDK</h2><Code>{'myenv system install java@17\nmyenv run --global java -version\nmyenv list'}</Code><p>run 后必须指定要运行的命令。默认环境不会自动加入系统 PATH。</p><h2>2. 创建第一个项目</h2><p>在新项目目录中用编辑器建立 <code>.node-version</code>，内容为 <code>22.23.2</code>，然后在该目录打开终端。</p><Code>{'myenv init\nmyenv sync\nmyenv status\nmyenv run node --version'}</Code><p>预期输出 v22.23.2。init 创建配置，sync 下载并准备环境，run 使用已经准备好的环境。</p>
 <h2>3. 运行代码</h2><p>建立 hello.js：</p><Code>{'console.log("你好，myEnv");\nconsole.log(process.version);'}</Code><Code>{'myenv run node hello.js'}</Code>
 <h2>如果你使用 Python</h2><p>另建项目目录，建立 .python-version 并写入 3.12.13：</p><Code>{'myenv init\nmyenv sync\nmyenv run python --version'}</Code>
 <h2>以后怎么用</h2><p>配置或依赖改变后运行 sync，平时用 run 执行命令。已有 myenv.yaml 无需重复 init。系统原有 Node/Python 不会自动被替换。</p></>},
 installation:{title:'安装与演示',intro:'从双击安装到终端输入 myenv。截图来自用户在 Windows 上的真实试用。',body:<>
 <h2>Windows 安装向导</h2><p>使用开发版交付的 <code>myenv-0.1.0-rc.1-windows-amd64-setup.exe</code>。请在 <a href="https://github.com/XVSHIFU/myEnv/releases">GitHub Releases</a> 查看已发布包；若列表为空，表示尚未上传发行包。</p><ol><li>双击安装向导。</li><li>保留默认目录 %LOCALAPPDATA%\Programs\myEnv 和“加入当前用户 PATH”。</li><li>点击“安装 / 升级”，无需管理员权限。</li></ol>
 <Shot file="image-20260909123106601.png" caption="选择安装目录，勾选加入当前用户 PATH"/><Shot file="image-20260909123140890.png" caption="安装完成，提示重新打开终端"/>
 <h2>确认命令可用</h2><p>退出终端应用后重新打开。只新增标签页不一定刷新继承的环境变量。</p><Code>{'myenv --version\nmyenv --help\nmyenv help manual'}</Code><Shot file="image-20260909123218913.png" caption="重新打开 PowerShell，直接使用 myenv 查看版本和中文帮助"/>
 <h2>便携安装与 PATH</h2><p>解压 portable.zip 可直接使用 myenv.exe。若希望输入 myenv，把所在目录（如 C:\Tools\myEnv）加入“编辑账户的环境变量 → 用户变量 → Path”。添加的是目录，不是 exe 文件。</p><Code>{'Get-Command myenv -All'}</Code><p>安装器检查 PATH 中同名程序；PowerShell 同名函数或别名需要自行检查。</p>
 <h2>原生 Linux</h2><p>使用 amd64/glibc 制品，不用于 WSL。先检查同名命令，再选择自己的工具目录：</p><Code>{'command -v myenv\nchmod +x ./myenv-linux-amd64\n./myenv-linux-amd64 --version'}</Code><p>需要直接输入 myenv 时，将制品以 myenv 为名放入已在 PATH 中的个人目录；不要覆盖来源不明的同名命令。</p>
 <h2>升级与卸载</h2><p>退出运行中的 myEnv，用新安装向导沿用原目录升级。Windows“已安装的应用”提供卸载入口，仅移除程序与安装器添加的 PATH 项，保留项目、环境和缓存。临时卸载助手可由系统临时文件清理移除。便携版删除自行放置的程序，并撤销自己的 PATH 项。</p><Note>原图未经编辑，拍摄于 2026-09-09，版本为 0.1.0-dev.2。截图中的路径仅代表测试机器，不是必须照填的配置。当前安装器未签名。</Note></>},
 support:{title:'支持范围与开发工具链',intro:'Node、Python、Temurin JDK、Go 与 Rust 可由 myEnv 安装并用于项目或用户默认环境。',body:<>
 <h2>安装 JDK 17</h2><Code>{'myenv system install java@17\nmyenv run --global java -version\nmyenv run --global javac -version'}</Code><p>Java 使用 Adoptium 提供的 Eclipse Temurin 归档。myEnv 向子进程设置 JAVA_HOME 和 PATH，不修改系统默认 Java。项目中使用 myenv use java@17。</p>
 <h2>Python 来源</h2><Code>{'myenv system install python@3.14 --provider python.org\nmyenv system install python@3.14 --provider astral'}</Code><p>Windows 支持 python.org 完整 ZIP；默认来源是 Astral。Linux 使用 Astral，尚不支持 python.org 源码构建。不会自动安装 Anaconda。</p>
 <h2>Go 与 Rust</h2><Code>{'myenv system install go@1.26\nmyenv system install rust@1.98\nmyenv run --global go version\nmyenv run --global rustc --version'}</Code><p>使用官方归档和摘要。Windows 托管 Rust 默认使用自带 LLD，避免已定位的 MSVC vctip 等待；自定义链接配置保留，可用 run --rust-linker system 显式选择系统流程。原生依赖仍可能需要 MSVC 与 Windows SDK。</p>
 <h2>C/C++ 与外部安装</h2><p>C/C++ 仅检测和官方安装指引，不自动安装。外部 uv、rustup、Conda 仅在核验原管理器及所属环境后支持部分操作；未知来源不自动接管。Conda base 受保护，Conda 尚无真实安装验收；rustup 不支持强制 repair。</p>
 <h2>平台与限制</h2><p>Windows amd64、原生 Linux amd64/glibc 已有 SDK 验证；macOS、WSL、musl 不在本次支持范围。并非每个历史版本、发行厂商与变体均已验收，缺少受信下载或校验信息时拒绝安装。部分延迟和聚合内存仍未达原预算，当前为发布候选。</p></>},
};
