import type { PackageRequest } from './package-model';
export type Request = {
    package?: PackageRequest;
    packageQuery?: string;
    packageName?: string;
    rebuild?: boolean;
    planTaskId?: number;
    action: string;
    directory: string;
    global: boolean;
    tool: string;
    selection: string;
    provider: string;
    preview: boolean;
    locked: boolean;
    deep: boolean;
    apply: boolean;
    major: number;
    channel: string;
    date: string;
    nodeMirror: string;
    uvMirror: string;
    pythonMirror: string;
    certificate: string;
    id: string;
    manager: string;
    externalAction: string;
    expectedDigest: string;
};
export type Lock = {
    version: string;
    runtime_version?: string;
    backend: string;
    evidence: string;
    url?: string;
    sha256?: string;
};
export type View = {
    directory: string;
    empty: boolean;
    platform: string;
    desired: Record<string, string>;
    applied: Record<string, Lock>;
    locked: Record<string, Lock>;
    path: string;
    digest: string;
    status?: {
        environment: string;
    };
};
export type Task = {
    summary?: {
        title: string;
        lines?: string[];
        changes?: {
            tool: string;
            from: string;
            to: string;
        }[];
    };
    startedAt?: number;
    updatedAt?: number;
    finishedAt?: number;
    phases?: string[];
    progress?: { kind: "download"; label: string; completed: number; total?: number };
    id: number;
    request: Request;
    state: string;
    phase: string;
    view?: View;
    result?: any;
    error?: {
        code: string;
        message: string;
        next_action: string;
    };
    log?: string;
    logTruncated?: boolean;
};
