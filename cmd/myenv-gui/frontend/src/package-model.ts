export type PackageTarget = {
    ecosystem: string;
    scope: string;
    root: string;
    directory?: string;
    interpreter?: string;
    manager?: string;
    manager_path?: string;
    tool?: string;
};

export type PackageItem = {
    name: string;
    requested?: string;
    version?: string;
    kind?: string;
    state?: string;
};

export type PackageContext = {
    target: PackageTarget;
    label: string;
    path: string;
};

export type PackageItemRequest = { name: string; desired: string; kind?: string };
export type PackageRequest = {
    target: PackageTarget;
    operation: string;
    items: PackageItemRequest[];
};
export type PackageSubmitPayload = {
    package: PackageRequest;
    packageQuery?: string;
    packageName?: string;
    planTaskId?: number;
    expectedDigest?: string;
};

export type RegistryVersion = {
    version: string;
    preview?: boolean;
    yanked?: boolean;
    deprecated?: string;
    requires_python?: string;
    requires_node?: string;
    available?: boolean;
};
export type RegistryPackage = {
    name: string;
    version?: string;
    latest?: string;
    summary?: string;
    description?: string;
};
export type PackageSearchResult = {
    ecosystem: string;
    query: string;
    mode: 'search' | 'exact';
    packages: RegistryPackage[];
    total: number;
    truncated?: boolean;
};
export type PackageCatalog = {
    ecosystem: string;
    name: string;
    summary?: string;
    homepage?: string;
    registry?: string;
    url?: string;
    state: 'found' | 'not_found';
    latest?: string;
    versions: RegistryVersion[];
    versions_total?: number;
    versions_truncated?: boolean;
    checked_at?: string;
};

export type PackagePlan = {
    id: string;
    request: PackageRequest;
    changes: {name:string;before?:string;requested?:string;desired?:string;version?:string;kind?:string;official_url?:string}[];
    commands: {executable:string;args:string[];directory:string;package:string}[];
    warnings: string[];
    state_digest: string;
    created_at: string;
};
export type PackageApplyResult = {
    plan_id: string;
    outcomes: {name:string;state:string;version?:string;problem?:string}[];
    log?: string;
    log_truncated?: boolean;
};

export function packageContextKey(context: PackageContext) {
    return JSON.stringify(context.target);
}

export function packageKind(kind?: string) {
    return ({ dependency: '依赖', development: '开发依赖', optional: '可选依赖', dependencies: '依赖', devDependencies: '开发依赖', optionalDependencies: '可选依赖', global: '全局工具', global_package_root: '全局工具', installed: '已安装', tool: '管理工具' } as Record<string, string>)[kind || ''] || kind || '包';
}

export function desiredLabel(value?: string) {
    return value === 'latest' ? '最新版' : value || '未设定';
}
