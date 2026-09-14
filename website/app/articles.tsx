import type {ReactNode} from 'react';
const base=process.env.NEXT_PUBLIC_BASE_PATH||'';
const Code=({children}:{children:string})=><pre><code>{children}</code></pre>;
const Note=({children}:{children:ReactNode})=><aside className="note">{children}</aside>;
const Shot=({file,caption,folder='installation'}:{file:string;caption:string;folder?:string})=><figure><a href={`${base}/images/${folder}/${file}`} target="_blank" rel="noreferrer"><img src={`${base}/images/${folder}/${file}`} alt={caption} loading="lazy"/></a><figcaption>{caption} · 点击查看原图</figcaption></figure>;
export const articles:Record<string,{title:string;intro:string;body:ReactNode}>={
 gui:{title:'GUI 工作台',intro:'先选工作范围，再选版本。保存、预览和执行各有明确的一步。',body:<>
 <Note>Windows GUI 从完整便携包启动，仍需 WebView2。<a href="https://github.com/XVSHIFU/myEnv/releases/tag/v0.1.0">下载 0.1.0 首版</a>，安装方法见<a href={`${base}/guide/installation/`}>安装与演示</a>。</Note>
 <h2>第一次打开：检查电脑已有环境</h2>
 <p>首次检查框会查找本机 Python、Node.js 等命令的版本与位置。勾选“同时检查一个项目目录”后，选择自己的项目文件夹，再点击“开始检查”。不想现在检查时可选“暂时跳过”；之后从“维护 → 选择检查范围”重新打开。</p>
 <p>检查在后台运行，并保存记录，后续启动不再反复弹出。它不会安装软件或修改终端的默认版本。安装、删除或手动切换过环境后，可用“维护 → 重新检查当前范围”刷新记录。</p>
 <h2>本机环境、myEnv 工具链和项目有什么区别</h2>
 <table><thead><tr><th>范围</th><th>这里查看或管理什么</th></tr></thead><tbody>
 <tr><td>本机环境 → 本机已安装</td><td>应用启动时 PATH 中的命令和其他已发现安装。用版本和完整路径确认现有环境。</td></tr>
 <tr><td>本机环境 → myEnv 工具链</td><td>myEnv 为当前用户准备的独立默认工具链，对应 CLI 的 <code>--global</code>。它不会自动替换系统 PATH。</td></tr>
 <tr><td>项目环境</td><td>顶部目录选择器中那个项目的声明、环境和命令。切换目录会切换操作范围，项目草稿分别保留。</td></tr>
 </tbody></table>
 <p>例如 PowerShell 的 <code>node --version</code> 是 v24，而 myEnv 工具链尚未配置 Node，这两个结果可以同时成立。本机检查显示的是实际路径，myEnv 工具链显示的是由 myEnv 准备的版本。若终端用了别名、函数或启动脚本，它与 GUI 继承的 PATH 还可能不同，见<a href={`${base}/guide/global/`}>本机与默认工具</a>。</p>
 <h2>选择并应用一个开发环境</h2>
 <ol><li>选择“项目环境”及项目目录，或在“本机环境”中打开“myEnv 工具链”。</li><li>在左侧选择语言，点击“查询版本”。用搜索框和主版本筛选缩小范围；也可手动输入目标版本。</li><li>选中版本，查看“当前生效”和“配置目标”。Java 的 JDK 1.8 对应 Java 8；厂商和完整版本保留在详情中。</li><li>点击“保存并预览”。空项目会先确认是否在该目录创建配置；保存声明本身尚未安装。</li><li>在中央变更预览中核对项目、语言和目标版本，再确认同步。完成后查看环境状态，或进入“运行命令”。</li></ol>
 <Shot folder="rc11" file="workbench.png" caption="rc.11 原生 GUI：示例项目筛选 Node 24 并选中 24.21.0，尚未保存或安装；底部任务区与保存入口常驻"/>
 <p>受管环境同步会先准备新环境，成功后切换；失败或取消保留原活动环境。查询结果只说明上游提供记录，真正安装还取决于平台和制品是否可用。</p>
 <h2>从语言直接进入包与工具</h2>
 <p>选中 Node.js 或 Python 后，当前语言下方显示已检查的 npm/pnpm 或 uv/pip 快捷入口。小窗口将这些入口移到页签下方。点击名称可管理对应工具，“更多…”打开完整的“包与工具”页；尚未检查时先点击“检查工具”。</p>
 <p>同名工具有多个位置时先选择要管理的那一个。pip 随具体 Python 解释器归属，uv 是独立工具。包的安装、更新与批量操作见<a href={`${base}/guide/packages/`}>包与工具管理</a>。</p>
 <h2>查看任务进度与结果</h2>
 <p>底部任务区始终显示当前操作、状态和耗时。点击“查看进度”展开阶段与日志；能获得下载总量时显示确定进度，查询或没有总量的步骤显示持续活动状态，不把等待时间伪装成完成百分比。</p>
 <p>可取消的任务会显示“取消任务”。点击后等待收尾，再看最终结果；“任务记录”可查看历史，但不会重新执行。详细日志单独滚动，确认与取消入口保留在窗口内。查询速度受网络和上游目录大小影响，失败后可重试或返回版本选择。</p>
 <h2>运行、返回与维护</h2>
 <p>“运行命令”中填写程序名和参数，普通参数逐行填写，高级选项支持 JSON 参数数组。确认运行后打开独立终端，命令结束显示退出码并等待 Enter，短命令的输出也会保留。终端已经接手的命令不随 GUI 关闭而取消。</p>
 <p>维护、设置和外部管理页面都有返回工作区的入口。切换项目之前先看顶部目录；来源详情、原始证据和任务日志可按需展开。恢复上一次环境只切换 myEnv 保留的环境代，不恢复项目代码、npm 包、声明或系统设置。</p>
 </>},
 packages:{title:'包与工具管理',intro:'先确认包属于哪里，再从右侧抽屉完成查询、预览和执行。',body:<>
 <h2>打开管理入口</h2>
 <p>在 GUI 中选 Node.js 或 Python，再打开“包与工具”。先点击“刷新检查”，确认列表来自当前项目、全局安装位置，还是某个 Python 解释器。管理工具全部展示；包列表默认显示10项，可展开其余内容。</p>
 <p>每行右侧“管理”打开侧边抽屉。语言下的工具快捷入口也会直接打开这里。同名工具或包可能存在于不同位置：检查抽屉顶部的位置后再继续，一次操作只针对该位置。</p>
 <h2>这些工具分别做什么</h2>
 <table><thead><tr><th>工具</th><th>管理对象与归属</th></tr></thead><tbody>
 <tr><td>npm / pnpm</td><td>管理 Node.js 包。项目依赖与全局工具分开显示；pnpm 项目使用对应项目管理器。</td></tr>
 <tr><td>pip</td><td>管理某个 Python 解释器中的包。软件绑定具体解释器，不会把项目 pip 换成本机的另一个 pip。</td></tr>
 <tr><td>uv</td><td>Astral 的独立工具，可处理 Python、虚拟环境和包。放在 Python 下方便找到，不代表它依赖当前 Python 才能运行。</td></tr>
 </tbody></table>
 <p>工具本身和它管理的包分别操作。例如更新 pip 与更新某个 Python 包是两件事。管理器自身需单项处理；uv 独立更新须能核对官方安装记录，独立安装的卸载需使用原安装方式。没有足够归属信息时，界面会说明原因。</p>
 <h2>从官方目录安装新包</h2>
 <ol><li>在目标位置打开管理抽屉，进入“npm 官方目录”或“PyPI 官方目录”。</li><li>npm 可输入包名或关键词，例如 <code>typescript</code>；PyPI 请输入完整包名，例如 <code>requests</code>。</li><li>找到目标包后点“加入变更”，查看简介和官方版本。</li><li>在“目标与操作”中选“安装”，设置期望；Node 项目还可选择依赖、开发依赖或可选依赖。</li><li>生成变更预览，核对安装位置、包名、原版本和目标版本，再确认执行。</li></ol>
 <p>PyPI 在这里不提供关键词搜索。“官方未找到”表示没有对应官方记录；网络或服务失败会单独提示，不能当作包不存在。本机已有但官方缺失的包仍会保留在列表中，能否卸载取决于是否能确认安装位置和管理器。</p>
 <h2>更新、切换与卸载已有包</h2>
 <p>点击包右侧“管理”，在“本次操作”里选择“更新到期望版本”“切换到指定版本”或“卸载”。更新默认选最新版；切换版本从当前版本开始编辑。卸载预览会列出将移除的具体包和位置。</p>
 <Shot folder="rc11" file="package-manager.png" caption="rc.11 原生 GUI：is-number 的 npm 官方信息和最新版期望，尚未执行安装"/>
 <table><thead><tr><th>期望</th><th>实际行为</th></tr></thead><tbody>
 <tr><td>最新版</td><td>预览时查询官方稳定版本，将具体版本列进本次计划。之后上游发布新版本，也不会悄悄改变已确认计划或后台自动升级。</td></tr>
 <tr><td>指定版本</td><td>选择官方版本或输入完整版本号。本次计划固定该版本；预发布、撤回等状态会单独标明。</td></tr>
 </tbody></table>
 <p>Node 项目的期望写入原生依赖声明，锁文件记录精确安装结果。全局包或外部 Python 的“最新版”只代表本次操作，不会自动改写 <code>pyproject.toml</code> 或 <code>requirements.txt</code>。官方版本列表有数量上限；找不到很旧的版本时可以尝试输入精确版本查询。</p>
 <h2>搜索和批量管理</h2>
 <ol><li>用包列表的小搜索框筛选名称。</li><li>进入选择状态，勾选需要的包，或全选当前筛选结果。全选也包括折叠的匹配项，请留意已选数量。</li><li>打开批量管理抽屉，选择本次操作。可逐包调整期望、取消个别包，也可将已选包全部设为最新版。</li><li>预览整批变更，逐项确认目标版本与安装位置，再执行。</li></ol>
 <Shot folder="rc11" file="package-batch.png" caption="rc.11 原生 GUI：同一示例项目中 is-number 7.0.0 与 is-odd 3.0.1 的安装预览，尚未确认执行"/>
 <p>同次最多100包，超过时请分批选择；软件不会只处理其中一部分却继续显示“全选”。不同项目或安装位置需要分开管理，npm/pip 等正在执行的管理器自身也需要单独操作。</p>
 <h2>执行中与执行后</h2>
 <p>抽屉和底部任务区显示真实阶段，可展开原始日志。取消后等待当前操作收尾；执行结果逐项标明已完成、失败、取消或未执行。关闭已完成的抽屉后，列表会重新检查。</p>
 <p>批量操作不保证全部成功或全部撤销。已经完成的外部包变更不会因后续失败、取消或 myEnv 的环境回退而自动恢复。先查看结果，再只处理未完成项。</p>
 <h2>当前可以管理到哪里</h2>
 <p>0.1.0 支持 npm/pnpm 项目包、明确 npm 全局位置，以及明确外部 Python 解释器的包操作。GUI 禁用 Node 安装脚本，Python 仅安装 wheel；需要编译、复杂 workspace 或无法确认管理器归属的情况，请用原管理器处理。</p>
 <p>myEnv 已应用的 Python 环境不能原地改包。请修改项目的 <code>pyproject.toml</code>，再通过 myEnv 同步准备新环境。CLI/TUI 保留现有工具链流程，本版包管理入口位于 Windows GUI。</p>
 </>},
 tui:{title:'TUI 终端界面',intro:'在真实终端里选择语言、查询版本、确认同步，然后运行程序。',body:<>
 <h2>启动与选择范围</h2>
 <Code>{'myenv tui\nmyenv tui --global\nmyenv -C ./project tui'}</Code>
 <p>第一条使用当前项目；<code>--global</code> 打开 myEnv 当前用户默认工具链；<code>-C</code> 明确项目目录。Windows 与原生 Linux 使用真实交互式终端，建议至少80列×24行。重定向、<code>--json</code> 和 <code>--no-input</code> 不进入 TUI；脚本请用 CLI。</p>
 <h2>从选择语言到安装完成</h2>
 <ol><li>用上下键选语言，按 Enter 查询版本。五种语言都保留查询与手动输入入口。</li><li>结果列表中按 <code>/</code> 筛选，Enter 选择目标。查询失败时可重试、手动输入或返回。</li><li>明确选择保存配置，再查看整个环境的变更预览。仅选中版本不会写入配置。</li><li>核对目标后按 Enter 确认同步，查看当前阶段；完成后可直接进入运行命令。</li></ol>
 <p>Esc 返回上一层并保留筛选。选择目标、保存声明和执行同步是三个步骤，方便检查项目和版本。安装失败或取消时，原活动环境仍保留。</p>
 <h2>常用按键</h2>
 <table><thead><tr><th>按键</th><th>用途</th></tr></thead><tbody>
 <tr><td>↑ / ↓、Enter</td><td>移动焦点、选择或确认当前步骤</td></tr><tr><td>Esc</td><td>返回上一层</td></tr><tr><td>/</td><td>版本列表中筛选</td></tr><tr><td>m</td><td>打开菜单，进入项目、来源、主题和记录</td></tr><tr><td>g / 1</td><td>切换项目与默认工具链 / 输入目录路径</td></tr><tr><td>c</td><td>取消当前可取消任务，等待收尾</td></tr><tr><td>q / Ctrl-C</td><td>退出界面</td></tr>
 </tbody></table>
 <p>以上按键以当前页面底部提示为准。在参数输入框内，字符用于输入；构建询问出现时，y/n 只回答当前任务，不是永久授权。</p>
 <h2>编辑参数与运行</h2>
 <p>“运行命令”分别编辑程序和参数，支持光标键、Home/End、Delete、Ctrl+U 和粘贴；高级 JSON 入口用于精确参数数组。先检查完整命令，再明确确认运行。</p>
 <p>确认后 TUI 暂时把终端交给程序，保持正常输入、输出和信号；程序结束后返回界面并显示真实退出码。仅返回、取消或退出 TUI 不会误执行等待中的命令。</p>
 <h2>维护与当前边界</h2>
 <p>“更多操作”提供诊断、锁定同步、修复预览、恢复上一次环境及清理。来源和主题在菜单中；复杂外部管理请先查看 CLI 帮助。0.1.0 的包管理入口位于 Windows GUI，TUI 没有包管理抽屉或专用包命令。</p>
 <p>原生 Linux CLI/TUI 已有验证，Linux GUI 暂缓；WSL TUI 门禁仍在，Windows TUI 空闲 CPU 尚未达到目标。完整范围见<a href={`${base}/guide/release/`}>版本记录与已知限制</a>。</p>
 </>},
 projects:{title:'Node 与 Python 项目',intro:'运行时由 myEnv 准备，项目原生配置仍由对应生态负责。',body:<>
 <h2>Node 项目</h2><p>在 myenv.yaml 中声明 Node 版本。myEnv 安装 Node 并提供 npm/npx，但不会自动替项目执行 npm install。</p><Code>{'schema: 1\ntools:\n  node: "22"'}</Code><Code>{'myenv sync\nmyenv run npm install\nmyenv run npm run dev'}</Code><p>npm install 会按项目定义安装依赖并可能执行生命周期脚本，请在自己信任的项目中使用。</p>
 <h2>Python 项目</h2><p>解释器版本写在 myenv.yaml，依赖继续写在 pyproject.toml，使用原生 uv.lock。myEnv 用固定 uv 后端准备新的 venv。</p><Code>{'schema: 1\ntools:\n  python: "3.12"\npython:\n  project: "."'}</Code><Code>{'myenv sync\nmyenv run python main.py'}</Code><p>依赖组通过 python.groups 显式选择。构建代码需要本次终端确认或 --allow-build，该选项允许以当前用户身份运行构建代码，不提供沙箱。</p>
 <h2>通过 GUI 管理包</h2><p>在“包与工具”页按项目、全局位置或 Python 解释器查看包，每行右侧“管理”打开抽屉。可查询官方目录、编辑版本期望、批量选择并预览执行，完整步骤见<a href={`${base}/guide/packages/`}>包与工具管理</a>。</p><p>Node 项目保留原生声明和锁文件；myEnv 已应用的 Python 环境不直接改包，请修改 pyproject.toml 后同步。GUI 包操作与命令行原管理器的安装脚本规则不同，先看各入口的操作说明。</p>
 <h2>混合项目</h2><p>同一 tools 映射可以同时声明 node 与 python，sync 准备整套环境。已有环境不会被就地覆盖，成功后才切换。</p><Code>{'schema: 1\ntools:\n  node: "22"\n  python: "3.12"'}</Code>
 <h2>参数与路径</h2><Code>{'myenv -C ./project run node hello.js "hello world" 中文'}</Code><p>用户命令之后的参数原样传递。run 不解释 Shell 管道或重定向。项目代码和业务数据不属于环境回滚范围。</p></>},
 versions:{title:'版本、同步与回滚',intro:'声明描述想要的环境，锁文件记录解析结果，活动环境是实际执行的版本。',body:<>
 <h2>修改版本</h2><Code>{'myenv use node@22\nmyenv use python@3.12'}</Code><p>use 修改声明并同步；准备失败会保留修改后的声明，但此前活动环境仍可保留使用。</p>
 <h2>查询与选择版本</h2><Code>{'myenv versions java --major 21\nmyenv versions node --search 22\nmyenv versions rust --channel nightly --date 2026-09-09'}</Code><p>查询结果取决于来源和当前平台。Java 可选多个主版本，Rust 历史 nightly 按日期查询；Python 的官网来源与 Astral 目录不同。各工具的来源、版本语法和安装示例见<a href={`${base}/guide/support/`}>支持范围</a>。</p><h2>固定依赖与无变更同步</h2><Code>{'myenv sync --locked --no-input'}</Code><p>已有且匹配的锁文件是前提。--locked 不更新 myenv.lock 或 Python 原生锁。没有变化时复用环境，不必重新安装。</p>
 <h2>检查后再应用</h2><Code>{'myenv sync --dry-run\nmyenv sync'}</Code><p>预览不安装、不修改环境或锁。尚未解析的 Python 选择器会在实际同步时确定。</p>
 <h2>失败后继续工作</h2><Code>{'myenv run --current node --version\nmyenv rollback\nmyenv run --current node --version'}</Code><p>rollback 需要已有上一完整环境。它切换活动引用，不回退声明、锁文件或业务代码。--current 明确允许使用与当前声明不同的已应用环境。</p>
 <h2>重新准备环境</h2><Code>{'myenv sync --rebuild --locked'}</Code><p>按匹配的当前锁准备新环境，不重装共享基础解释器或清空缓存。主/次版本默认选择稳定系列；预览版必须显式选择。</p></>},
 global:{title:'本机与默认工具',intro:'看清命令来自哪条路径，避免把电脑已有环境和 myEnv 工具链混为一谈。',body:<>
 <h2>本机环境显示什么</h2>
 <p>GUI 的“本机已安装”记录应用启动 PATH 中的 Python、Node.js 等命令和其他已发现的安装。首次检查或“维护 → 重新检查当前范围”会刷新版本、路径与包信息；检查不会自动导入、升级或删除已有安装。</p>
 <p>终端中的版本还可能来自 PowerShell 函数、别名或 Shell 初始化脚本。GUI 不会执行你的所有终端启动脚本，因此应同时核对版本和完整路径，而不是只比较一个版本号。</p>
 <Code>{'Get-Command node -All\nGet-Command python -All\nnode --version\npython --version'}</Code>
 <p>如果刚安装过软件或修改过 PATH，先退出 GUI 再重开，然后重新检查。仅刷新列表不会改变应用已经继承的环境变量。</p>
 <h2>myEnv 默认工具链</h2>
 <p>这是当前用户的独立工具环境，在 GUI“本机环境 → myEnv 工具链”中选择版本。CLI 的 <code>--global</code> 和 <code>system install</code> 对应这一范围，不是管理员或全机器安装。</p>
 <Code>{'myenv system install java@21\nmyenv use --global python@3.12\nmyenv run --global python --version\nmyenv doctor --global'}</Code>
 <p>项目与默认工具链相互独立，项目不会隐式继承这里的工具。准备默认 Python 后，直接输入 <code>python</code> 仍可能运行原有 PATH 版本；用 <code>myenv run --global python</code> 可明确选择。</p>
 <h2>让当前终端直接使用默认工具</h2>
 <p>已有 myEnv 默认工具链后，在 PowerShell 显式执行：</p>
 <Code>{'myenv shell-init powershell | Out-String | Invoke-Expression'}</Code>
 <p>它在当前 Shell 定义包装函数，通过 <code>run --global</code> 使用默认工具，不修改系统 PATH 或 Shell 文件。关闭终端即可丢弃这些函数；添加工具或移动 myEnv 后重新生成。移除工具后应打开新 Shell，已有同名用户函数不会自动恢复。Bash、Zsh、Fish 用法见离线手册。</p>
 <h2>更新、修复与移除默认工具</h2>
 <Code>{'myenv system list --details\nmyenv system upgrade java\nmyenv system repair java\nmyenv system remove java\nmyenv system clean --dry-run'}</Code>
 <p>upgrade 在原版本范围内更新，repair 重新准备默认环境，remove 移除对应声明。GUI 从“维护 → 默认工具维护”进入。实际写入前先确认范围；旧环境继续按运行和恢复保护保留。</p>
 <h2>电脑上原来安装的工具</h2>
 <p>包与工具页中的 npm、pnpm、pip、uv 按具体位置管理，详见<a href={`${base}/guide/packages/`}>包与工具管理</a>。对于由其他管理器安装的环境，可从“维护 → 外部环境管理”选环境、核验原管理器，再查看计划并确认。</p>
 <Code>{'myenv list\nmyenv system doctor\nmyenv system external --help'}</Code>
 <p>外部管理要求原管理器和安装目标能够核验；未知来源保持只读。CLI 默认只生成计划，只有显式 <code>--apply</code> 执行。Conda 的移除会删除整个目标环境及其中的包，不能理解成只移除一个 Python 命令。</p>
 </>},
 troubleshooting:{title:'排障与清理',intro:'先查看状态和路径，再决定是否同步、重建或清理。',body:<>
 <h2>找不到 myenv 命令</h2><p>安装后退出整个终端应用并重开。Windows 运行 Get-Command myenv -All，检查同名函数/别名和 PATH；也可直接用安装目录下的 exe。</p>
 <h2>声明与环境不一致</h2><Code>{'myenv status\nmyenv sync\nmyenv doctor'}</Code><p>如需暂时使用旧活动环境，显式选择 run --current。run 不会隐式帮你下载缺少的工具。</p>
 <h2>检查文件内容</h2><Code>{'myenv doctor --deep'}</Code><p>深度检查比较环境代内容基线，可能较慢。它不检查外部共享运行时内容、Python 字节码缓存或 ACL，也不自动修复。</p>
 <h2>安全清理</h2><Code>{'myenv clean --dry-run\nmyenv clean'}</Code><p>保留当前、上一代和仍受运行/准备保护的环境。无法确认后代进程退出时继续保留保护；不要手动删除租约绕过检查。用户默认环境用 myenv system clean --dry-run 预览，clean 不接受 --global。</p><Code>{'myenv clean --cache node --dry-run\nmyenv clean --cache uv --dry-run'}</Code><p>缓存清理不卸载共享 Python 解释器。项目 venv 可能引用共享解释器，不要随意手动删除。</p>
 <h2>GUI 检查不到终端中的 Node / Python</h2><p>先在终端核对命令的完整路径，再退出 GUI 重开并刷新检查。GUI 使用启动时继承的 PATH，不会自动执行 PowerShell 别名或 Shell 启动脚本。myEnv 工具链未配置某个语言，也不代表本机没有安装，详见<a href={`${base}/guide/global/`}>本机与默认工具</a>。</p><h2>官方查询失败或没有结果</h2><p>“官方未找到”与网络错误是不同结果。npm 可以搜索关键词，PyPI 需要完整包名；先检查拼写，再重试。网络失败时检查代理、证书和网络连接，不要把失败结果当成安装包已消失。版本查询较慢时看底部活动状态；需要时取消并等待收尾。</p><h2>包操作部分失败或已取消</h2><p>展开执行结果，区分已完成、失败、取消和未执行的条目，然后刷新原列表。已完成的外部包变更不会自动撤销；只对仍需处理的条目重新生成计划。遇到权限、安装脚本或不支持的工作区说明时，按提示使用原管理器。</p><h2>颜色与进度</h2><p>交互终端自动显示颜色与单行阶段动画，重定向保留普通文本。NO_COLOR=1 关闭装饰；--no-input、--verbose 或 --json 关闭动画。命中已有环境时可能没有准备阶段。用户程序输出保持原样。</p><h2>反馈问题</h2><p>保留 myenv --version、失败命令、完整输出及紧接着读取的 $LASTEXITCODE。可以附原始截图；公开分享前检查是否包含凭据或不希望公开的信息。</p></>},
 commands:{title:'命令速查',intro:'日常只需 init → sync → run，其余命令按需要使用。',body:<>
 <table><thead><tr><th>命令</th><th>作用</th></tr></thead><tbody>{[['myenv tui','打开交互式终端界面'],['myenv / myenv status','只读查看当前项目状态'],['myenv init','识别声明并创建配置，不覆盖已有文件'],['myenv sync','准备并应用新环境'],['myenv use node@22','修改版本声明并同步'],['myenv run <command>','使用已应用环境执行命令'],['myenv versions java --major 8','查询 Java 8（JDK 1.8）版本'],['myenv rollback','切换到上一完整环境'],['myenv doctor --deep','只读检查路径和内容基线'],['myenv clean --dry-run','预览可清理的项目环境'],['myenv help manual','完整离线手册'],['myenv completion powershell','生成补全脚本'],['myenv shell-init powershell','生成默认工具包装函数']].map(([cmd,desc])=><tr key={cmd}><td><code>{cmd}</code></td><td>{desc}</td></tr>)}</tbody></table>
 <h2>获取单个命令帮助</h2><Code>{'myenv help sync\nmyenv run --help\nmyenv --lang en --help'}</Code><p>参数支持范围以对应命令帮助为准。--global 不是每个命令都接受，--json 不适用于 run/help/completion/shell-init。</p>
 <h2>退出码</h2><p>myEnv 成功为 0，执行或环境错误为 1，用法或配置错误为 2，需要输入为 3，取消通常为 130。用户程序启动后，run 保留程序自己的退出码；Linux 信号终止使用 128 加信号编号。收尾错误不会覆盖已经完成的子程序退出码。</p></>},
 configuration:{title:'配置与自动化',intro:'配置保持可读，自动化依赖稳定的 JSON 与退出码。',body:<>
 <h2>myenv.yaml</h2><Code>{'schema: 1\ntools:\n  node: "22"\n  python: "3.12"\npython:\n  project: "."\n  groups: [dev]\nenv:\n  APP_MODE: "development"'}</Code><p>只声明需要的工具；Python 依赖组必须与项目匹配。python.project 必须是工作区内的相对路径。版本值加引号。env 用于普通字符串，凭据通过调用环境传入。</p>
 <h2>非交互调用</h2><Code>{'myenv sync --locked --no-input --json\nmyenv status --json'}</Code><p>JSON 使用 schema/ok/changed/data/error，错误含 code/message/next_action。显示语言不改变 JSON 合同；诊断写到 stderr。--json 和 --no-input 不等待终端输入。</p>
 <h2>语言设置</h2><Code>{'myenv --lang zh-CN --help\nmyenv --lang en --help'}</Code><p>优先级为 --lang、MYENV_LANG、系统语言、英文回退。auto 在中文系统显示简体中文。参数、路径、错误码和第三方输出不翻译；run 后属于子程序的 --lang 原样传递。</p>
 <h2>镜像与证书</h2><p>MYENV_NODE_MIRROR、MYENV_UV_MIRROR、MYENV_PYTHON_MIRROR 分别指定下载镜像。需要遵循原官方目录布局；Node 新解析会记录来源和摘要，已锁定 URL 不变。uv 的固定版本和摘要验证不变。</p><p>SSL_CERT_FILE 使用绝对路径 PEM 文件；myEnv 下载器不支持 SSL_CERT_DIR。无效证书会报错，不关闭 TLS 校验。更多约束见 myenv help manual。</p></>},
 release:{title:'版本记录与已知限制',intro:'文档依据实际交付与测试记录编写，计划中的能力会明确标注。',body:<>
 <h2>0.1.0 · 首版</h2><p>首版提供 Windows GUI、TUI、CLI 和原生 Linux CLI/TUI。安装包与校验清单见 <a href="https://github.com/XVSHIFU/myEnv/releases/tag/v0.1.0">GitHub Release v0.1.0</a>。</p><ul><li>Windows GUI 区分本机检查、myEnv 默认工具链和项目环境，保留原生图标及固定任务区。</li><li>Node.js / Python 语言下提供管理工具快捷入口，按实际位置打开右侧抽屉。</li><li>包支持搜索、选择、批量管理及最新版/精确期望；官方 npm/PyPI 查询、预览确认和逐项结果贯通。</li><li>Java 支持常见版本名称与 1.8 别名；Windows/Kali TUI 明确确认后才交接运行，信号退出不会误执行命令。</li></ul><h2>历史：0.1.0-rc.1</h2><ul><li>新增 Java、Go、Rust 与 Python 官网来源。</li><li>系统发现、原管理器受限操作、预览版选择。</li><li>分组列表、中文使用提示、终端颜色与动态阶段进度。</li></ul><h2>历史：0.1.0-dev.2</h2><ul><li>中文帮助、手册与日常提示，支持英文切换。</li><li>status 与裸命令共用状态入口。</li><li>Windows 当前用户安装向导及便携包。</li><li>Windows/Kali 各 35 项定向测试通过；用户实测安装、PATH 与语言切换成功。</li></ul>
 <h2>语言切换实测</h2><Shot file="image-20260909123308407.png" caption="显式切换为英文帮助"/><Shot file="image-20260909123314813.png" caption="切回简体中文帮助"/>
 <h2>当前已知限制</h2><ul><li>旧截图中 help 简介漏翻译已修复；原图继续保留对应 dev.2 的真实状态。</li><li>既有 GUI 启动、Windows TUI 空闲 CPU 及其他延迟/内存预算仍有未达标项，不将功能通过解释为性能达标。</li><li>Linux GUI 暂缓，WSL TUI 门禁仍在；macOS、musl 不在本次支持范围。</li><li>GUI 需要 WebView2；高 DPI、真实 IME、无 WebView2 干净机器及管理工具自身真实写操作仍有验收缺口。</li><li>GUI 包操作禁用安装脚本，Python 仅 wheel；同次最多100包，执行管理器自身需单项处理。复杂 workspace、未知归属和独立 uv 卸载交回原管理器。</li><li>Linux 调用者暂停期间无法执行自身取消回调；绝对截止时间由独立监督器执行。</li><li>便携包包含 Windows GUI，setup 仅安装 CLI/TUI；制品未签名，安装包见 <a href="https://github.com/XVSHIFU/myEnv/releases/tag/v0.1.0">0.1.0 下载页</a>。</li></ul><p>截图对应 dev.2；当前功能与限制以上述文字为准。</p></>},
 quickstart:{title:'快速开始',intro:'选择适合自己的入口，先完成一次环境准备，再运行项目。',body:<>
 <Note>0.1.0 首版提供 Windows GUI、TUI 和 CLI，原生 Linux 提供 CLI/TUI。<a href="https://github.com/XVSHIFU/myEnv/releases/tag/v0.1.0">下载 0.1.0 与校验清单</a>，按需要选择便携包或安装向导。</Note>
 <h2>选一个入口</h2>
 <table><thead><tr><th>希望怎样使用</th><th>从这里开始</th></tr></thead><tbody>
 <tr><td>用图形界面检查环境、选择版本、管理包</td><td>完整解压 Windows 便携包，启动 <code>myenv-gui.exe</code>，然后阅读<a href={`${base}/guide/gui/`}>GUI 工作台</a>。</td></tr>
 <tr><td>在终端里用菜单操作</td><td>运行 <code>myenv tui</code>，按页面提示选择语言，详见<a href={`${base}/guide/tui/`}>TUI 终端界面</a>。</td></tr>
 <tr><td>用命令行开发或自动化</td><td>安装 CLI 后按下方步骤准备项目；完整参数见<a href={`${base}/guide/commands/`}>命令速查</a>。</td></tr>
 </tbody></table>
 <p>Windows setup 只安装 CLI/TUI。GUI 在便携包内，需要 WebView2；便携包无需安装 Go 或 Node。下载和系统要求见<a href={`${base}/guide/installation/`}>安装与演示</a>。</p>
 <h2>先分清现有环境与受管环境</h2>
 <p>“本机环境”查看电脑已有的命令和路径；“myEnv 工具链”是当前用户的独立默认环境；“项目环境”属于选中的目录。检查只保存发现记录，不把系统 Node/Python 自动替换成 myEnv 版本。</p>
 <p>GUI 首次可检查本机并自选一个项目目录，也可跳过，之后从“维护”重新检查。已有 npm、pnpm、pip、uv 会按实际位置出现在语言快捷入口与“包与工具”页。</p>
 <h2>CLI：准备第一个 Node 项目</h2>
 <p>先在项目目录创建 <code>.node-version</code>，写入 <code>22</code>，然后打开终端：</p>
 <Code>{'myenv init\nmyenv sync\nmyenv run node --version'}</Code>
 <p><code>init</code> 创建配置；<code>sync</code> 准备并应用环境；<code>run</code> 使用已应用的版本。首次同步选择可用的 Node 22 稳定补丁并记录精确结果，后续复用锁定版本。已有 <code>myenv.yaml</code> 时从 sync 开始。</p>
 <p>创建 <code>hello.js</code>：</p><Code>{'console.log("你好，myEnv");\nconsole.log(process.version);'}</Code><Code>{'myenv run node hello.js'}</Code>
 <h2>Python、Java 与混合项目</h2>
 <p>Python 项目可以创建 <code>.python-version</code> 并写入 <code>3.12</code>，再执行 init、sync 和 <code>myenv run python --version</code>。项目依赖继续放在原生配置中，见<a href={`${base}/guide/projects/`}>Node 与 Python 项目</a>。</p>
 <p>只需要一个独立默认 JDK 时：</p><Code>{'myenv system install java@21\nmyenv run --global java -version'}</Code>
 <p>JDK 1.8 对应 Java 8，可以用 <code>java@8</code>。示例版本不代表支持范围的上下限，更多语言、官方来源和选择方式见<a href={`${base}/guide/support/`}>支持范围与工具链</a>。</p>
 <h2>以后怎样使用</h2>
 <p>平时通过 <code>myenv run</code> 运行代码，修改工具版本或 Python 依赖后再同步。<code>run</code> 不隐式安装；同步失败时保留原活动环境。GUI 的包操作另有预览与结果页，详见<a href={`${base}/guide/packages/`}>包与工具管理</a>。</p>
 <p>忘记命令时运行 <code>myenv help manual</code>，遇到路径或查询问题时先看<a href={`${base}/guide/troubleshooting/`}>排障与清理</a>。</p>
 </>},
 installation:{title:'安装与演示',intro:'Windows 便携包打开 GUI，安装向导用于 CLI/TUI；原生 Linux 使用命令行制品。',body:<>
 <Note>从 <a href="https://github.com/XVSHIFU/myEnv/releases/tag/v0.1.0">GitHub Release v0.1.0</a> 下载安装包和 SHA256SUMS。当前制品未签名；GUI 需要 WebView2。</Note>
 <h2>选择安装方式</h2>
 <table><thead><tr><th>制品</th><th>包含内容</th></tr></thead><tbody><tr><td>Windows portable.zip</td><td>GUI、CLI/TUI、第三方许可。完整解压即可启动 GUI。</td></tr><tr><td>Windows setup.exe</td><td>CLI/TUI，可加入当前用户 PATH，提供升级和卸载入口。</td></tr><tr><td>myenv-linux-amd64</td><td>原生 Linux amd64/glibc 的 CLI/TUI，不包含 GUI。</td></tr></tbody></table>
 <p>使用发行包无需安装 Go 或 Node。Windows 需要 Windows 10 / Server 2016 或更新版本；setup 使用系统 .NET Framework 4.x。GUI 另外需要 WebView2。更多范围见<a href={`${base}/guide/support/`}>支持范围与工具链</a>。</p>
 <h2>Windows：启动便携 GUI</h2>
 <ol><li>完整解压 <code>myenv-0.1.0-windows-amd64-portable.zip</code>，放到固定位置，例如 <code>C:\Tools\myEnv</code>。</li><li>保留同目录的 <code>myenv-gui.exe</code>、<code>myenv.exe</code> 和许可文件。</li><li>双击 <code>myenv-gui.exe</code>，按首次检查提示选择本机或项目范围。</li></ol>
 <p>缺少 WebView2 Runtime 时，当前 GUI 提示使用微软联网 bootstrapper；离线机器须预先安装微软官方 x64 Evergreen Standalone Installer，包内没有离线运行时。完成后进入<a href={`${base}/guide/gui/`}>GUI 工作台指南</a>。</p>
 <h2>Windows：安装 CLI/TUI</h2>
 <ol><li>双击 <code>myenv-0.1.0-windows-amd64-setup.exe</code>。</li><li>默认目录为 <code>%LOCALAPPDATA%\Programs\myEnv</code>，保持“加入当前用户 PATH”勾选。</li><li>点击“安装 / 升级”，无需管理员权限。</li><li>退出整个终端应用并重开 PowerShell，执行下列命令。只新增终端标签页不一定刷新环境变量。</li></ol>
 <Code>{'myenv --version\nmyenv --help\nmyenv tui'}</Code>
 <p>便携版也可通过绝对路径使用 CLI/TUI。若要直接输入 myenv，把所在目录加入用户 Path，添加的是目录而不是 exe 文件。安装器检查 PATH 同名程序；PowerShell 函数或别名可用 <code>Get-Command myenv -All</code> 检查。</p>
 <h2>核对下载文件</h2>
 <Code>{'Get-FileHash ./myenv-0.1.0-windows-amd64-portable.zip -Algorithm SHA256'}</Code>
 <p>将结果与同次提供的 SHA256SUMS 比对。摘要检查文件完整性，不是发行者签名。安装 myenv 本身不会替换系统 Node/Python；使用 <code>myenv run</code> 明确选择项目环境。</p>
 <h2>原生 Linux</h2>
 <p>使用 amd64/glibc 制品。先检查同名命令，再选择自己的工具目录：</p>
 <Code>{'command -v myenv\nchmod +x ./myenv-linux-amd64\n./myenv-linux-amd64 --version\n./myenv-linux-amd64 tui'}</Code>
 <p>需要直接输入 myenv 时，将制品以 myenv 为名放入 PATH 中的个人目录。Linux GUI 暂缓；WSL TUI 门禁仍在，macOS、musl 和 ARM 不在本次支持范围。</p>
 <h2>升级与卸载</h2>
 <p>先退出正在运行的 myEnv，用新 setup 沿用原目录升级，更换目录前先卸载旧安装。Windows“已安装的应用”中的卸载只移除程序和安装器添加的 PATH 项，保留项目、环境和缓存；便携版删除自行放置的程序，并撤销自己的 PATH 项。</p>
 <h2>历史安装实拍</h2>
 <p>以下原图来自用户在 Windows 上安装 dev.2 的过程，保留原样。它们展示安装与 PATH 操作，不代表 0.1.0 GUI 的当前界面；截图路径只是示例。</p>
 <Shot file="image-20260909123106601.png" caption="dev.2 历史实拍：选择安装目录与当前用户 PATH"/>
 <Shot file="image-20260909123140890.png" caption="dev.2 历史实拍：安装完成，提示重新打开终端"/>
 <Shot file="image-20260909123218913.png" caption="dev.2 历史实拍：重新打开 PowerShell 后查看 myenv 版本和中文帮助"/>
 </>},
 support:{title:'支持范围与开发工具链',intro:'先确认发行来源，再查询版本。示例版本不是安装范围的上限或下限。',body:<>
 <h2>可安装的工具与来源</h2><p>myEnv 根据上游目录选择当前平台的安装包，不维护一张静态的“全部版本”名单。以下范围以目录仍提供兼容制品为前提，不能理解为所有历史版本均已验证。</p>
 <table><thead><tr><th>工具与来源</th><th>可以怎样选择</th></tr></thead><tbody>
 <tr><td>Java：<a href="https://adoptium.net/temurin/releases/">Eclipse Temurin</a></td><td>HotSpot JDK x64。可选 8、11、17、21、25 等目录提供的主版本，或完整发行名（如 jdk-21.0.12.1+1、jdk8u462-b08）。不包含其他厂商、OpenJ9 或 JRE。</td></tr>
 <tr><td>Node：<a href="https://nodejs.org/dist/index.json">Node.js 官方归档</a></td><td>当前平台有归档的版本；支持 22、22.23、22.23.2 等版本族、完整版本和范围。附带该归档的 npm/npx。</td></tr>
 <tr><td>Python：<a href="https://docs.astral.sh/uv/concepts/python-versions/">Astral CPython</a>（默认）</td><td>固定 uv 0.11.26 目录中的默认 CPython 变体；支持 3.12、3.14 等版本族、完整版本和范围。该目录可能落后于 Python 官网。</td></tr>
 <tr><td>Python：<a href="https://www.python.org/ftp/python/index-windows.json">python.org</a>（可选）</td><td>仅 Windows 完整 x64 runtime ZIP；排除嵌入式、free-threaded 和测试变体。使用 --provider python.org，不运行 EXE/MSI。</td></tr>
 <tr><td>Go：<a href="https://go.dev/dl/">Go 官方归档</a></td><td>当前平台 amd64 归档中的版本族或完整版本，如 1.26、1.26.6；历史精确名称可加 =，如 go@=1.2。</td></tr>
 <tr><td>Rust：<a href="https://forge.rust-lang.org/infra/archive-stable-version-installers.html">Rust 官方归档</a></td><td>完整工具链的版本族或完整稳定版，如 1.98、1.98.0。Windows 使用 x86_64-pc-windows-msvc，Linux 使用 x86_64-unknown-linux-gnu。</td></tr>
 </tbody></table>
 <h2>查询后安装</h2><Code>{'myenv versions java --major 21\nmyenv versions node --search 22\nmyenv versions go --search 1.26\nmyenv versions rust --search 1.98'}</Code><p>Java 的 --major 只是缩小查询；换成 8、17、25 等即可查询其他主版本。复制查询中的完整版本可固定到具体发行。下面演示当前用户默认环境；项目内将 system install 换成 use。</p><Code>{'myenv system install java@21\nmyenv system install node@22\nmyenv system install go@1.26\nmyenv system install rust@1.98\nmyenv run --global java -version'}</Code>
 <h2>Python 的两种来源</h2><Code>{'myenv system install python@3.12 --provider astral\nmyenv run --global python --version'}</Code><p>Windows 还可查询并安装官网包：</p><Code>{'myenv versions python --provider python.org\nmyenv system install python@3.14 --provider python.org'}</Code><p>来源随声明保存，不会静默替换。Windows/Linux 的 versions python 默认查询 Astral，与默认安装一致；查询复用已有固定 uv，不隐式下载。选择官网包须显式保留 --provider python.org；Linux 官网查询仅提供发布记录，尚无 python.org 二进制或源码构建后端。</p>
 <h2>稳定版、预览版与锁定</h2><p>普通版本族只选择稳定版。Node/Python 另支持比较范围；Node 支持 ^、~，Python 支持 ~=。Java/Go/Rust 使用版本族或完整名称，不支持这类比较表达式。</p><Code>{'myenv versions java --major 26 --preview\nmyenv versions node --preview\nmyenv versions go --preview\nmyenv versions rust --channel nightly --date 2026-09-09\nmyenv use rust@nightly-2026-09-09'}</Code><p>Node nightly/rc/v8-canary、Python a/b/rc、Temurin EA 和 Go beta/rc 须指定完整预览名；Rust 支持 beta、nightly 及日期渠道。--preview 扩展查询，不自动升级环境。Rust 没有全部历史 nightly 索引，指定日期的制品也可能缺失。成功同步后以 myenv.lock 记录的精确结果为准。</p>
 <h2>平台与已验证样本</h2><p>0.1.0 提供 Windows amd64 CLI/TUI/GUI 与原生 Linux amd64/glibc CLI/TUI；Linux GUI 暂缓，WSL TUI 门禁仍在，macOS、musl 和 ARM 暂不支持。Windows 需要 Windows 10/Server 2016 或更新版本；GUI 还需 WebView2。Linux TTY 需要允许 pidfd 系统调用，通常为 Linux 5.3 及以上。上游运行时自身的系统要求仍须满足。</p><p>已有验证记录包含 Node 22.23.2、Astral Python 3.12.13，以及 Windows/Kali 的 Temurin jdk-21.0.12.1+1、Go 1.26.6、Rust 1.98.0。Windows 官网 Python 3.14.7 与部分预览版、Kali 日期 Rust nightly 也有定向验证。这些是实测样本，不是所有可选版本或所有操作系统的兼容承诺；0.1.0 仍有未达性能预算的项目。GUI 包写操作的真实闭环限 Windows 隔离样本，Linux 包管理已有离线核心回归，尚无真实包写操作闭环。</p>
 <h2>编译依赖与外部环境</h2><p>Windows 托管 Rust 默认使用包内 LLD；自定义链接配置优先，也可用 run --rust-linker system 选择系统流程。Rust 原生依赖仍可能需要 MSVC 与 Windows SDK。Go 设置 GOTOOLCHAIN=local，运行时不隐式下载其他工具链。</p><p>C/C++ 仅检测并提供官方安装指引。外部 uv、rustup、Conda 的操作须核验原管理器和目标；Conda base 受保护，Conda 尚无真实安装验收，rustup 不支持强制 repair。myEnv 不自动安装 Anaconda。</p>
 <h2>开发 myEnv 本身</h2><p>使用发行包无需预装 Go 或 Node。从源码构建 CLI 使用 go.mod 固定的 Go 1.26.6；开发文档站使用 Node ≥22.13.0。它们是项目构建要求，与上面的受管工具版本范围相互独立。</p></>},
};
