import React, { useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from '../i18n';

type MenuKey = 'overview' | 'repositories' | 'newReview' | 'sessions' | 'sessionDetail' | 'reports' | 'issueLists' | 'rules' | 'settings';

interface RepoItem {
  encodedPath: string;
  displayName: string;
  repoPath: string;
  remoteURL?: string;
  sessionCount: number;
  fileCount?: number;
  hasToken?: boolean;
}

interface SessionSummary {
  sessionID: string;
  encodedRepo?: string;
  timestamp: string;
  cwd: string;
  repoName?: string;
  repoPath?: string;
  gitBranch: string;
  model: string;
  reviewMode: string;
  diffFrom: string;
  diffTo: string;
  diffCommit: string;
  filesReviewed: string[];
  durationSec: number;
  fileCount: number;
  llmFailures: number;
  warningCount?: number;
  status?: string;
}

interface SessionDetail {
  summary: SessionSummary;
  tokenUsage: {
    totalPromptTokens: number;
    totalCompletionTokens: number;
    totalCacheReadTokens: number;
    totalCacheWriteTokens: number;
    requestCount: number;
  };
  files: Array<{
    filePath: string;
    tasks: Record<string, Array<{
      requestNo: number;
      responseContent: string;
      durationMs: number;
      error: string;
      model: string;
      promptTokens: number;
      completionTokens: number;
      cacheReadTokens: number;
      cacheWriteTokens: number;
      toolCalls: Array<{
        name: string;
        arguments: string;
        result: string;
        ok: boolean;
        durationMs: number;
      }>;
    }>>;
  }>;
}

interface ReviewIssue {
  path: string;
  content: string;
  suggestionCode: string;
  existingCode: string;
  thinking: string;
}

interface AuditReport {
  id: string;
  encodedRepo: string;
  repoName: string;
  sessionID: string;
  sessionTime: string;
  generatedAt: string;
  model: string;
  issueCount: number;
  fileCount: number;
  status: string;
  content: string;
  structured?: AuditReportStructured;
  promptTokens?: number;
  outputTokens?: number;
}

interface AuditReportStructured {
  title: string;
  executiveSummary: string;
  riskLevel: string;
  riskReason: string;
  categories?: Array<{ name: string; riskLevel: string; count?: number; description: string }>;
  keyModules?: Array<{ path: string; riskLevel: string; finding: string; suggestion: string }>;
  topFixes?: Array<{ priority: number; riskLevel: string; title: string; file: string; impact: string; suggestion: string }>;
  roadmap?: Array<{ phase: string; goal: string; items?: string[] }>;
  manualReview?: string[];
}

interface RuleLayer {
  priority: number;
  source: string;
  title: string;
  path: string;
  description?: string;
  available: boolean;
  defaultRule?: string;
  include?: string[];
  exclude?: string[];
  rules?: Array<{ path: string; rule: string }>;
  pathRules?: Array<{ pattern: string; rule: string }>;
}

interface IssueListItem {
  path: string;
  content: string;
  severity: string;
  category: string;
  suggestion?: string;
  existingCode?: string;
  lineStart?: number;
  lineEnd?: number;
}

interface IssueListData {
  id: string;
  encodedRepo: string;
  repoName: string;
  sessionID: string;
  sessionTime: string;
  generatedAt: string;
  model: string;
  issues: IssueListItem[];
  fileCount: number;
}

interface ReviewTask {
  taskID: string;
  encodedRepo: string;
  repoName: string;
  repoPath: string;
  status: string;
  startedAt: string;
  finishedAt?: string;
  exitCode?: number;
  errorMessage?: string;
  output?: string;
}

interface LLMConfigState {
  url: string;
  authToken: string;
  model: string;
  useAnthropic: boolean;
  extraBody: string;
}

interface AddRepoResponse extends RepoItem {
  error?: string;
}

interface ReviewFormState {
  encodedRepo: string;
  reviewMode: string;
  baseRef: string;
  targetRef: string;
  commitRef: string;
  encodedBranch?: string;
  background: string;
  format: string;
  timeout: string;
  concurrency: string;
  rulePath: string;
}

const AdminConsolePage: React.FC = () => {
  const { t, language, setLanguage } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();

  const [activeMenu, setActiveMenu] = useState<MenuKey>('overview');
  const [selectedSession, setSelectedSession] = useState<string>('');

  // Data states
  const [repos, setRepos] = useState<RepoItem[]>([]);
  const [reposLoading, setReposLoading] = useState(false);
  const [selectedRepo, setSelectedRepo] = useState<string>('');
  const [sessions, setSessions] = useState<SessionSummary[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const [sessionDetail, setSessionDetail] = useState<SessionDetail | null>(null);
  const [sessionDetailLoading, setSessionDetailLoading] = useState(false);
  const [showIssueList, setShowIssueList] = useState(false);
  const [issueListExpanded, setIssueListExpanded] = useState(false);
  const [reviewFilesExpanded, setReviewFilesExpanded] = useState(false);
  const [ruleLayers, setRuleLayers] = useState<RuleLayer[]>([]);
  const [rulesLoading, setRulesLoading] = useState(false);
  const [llmConfig, setLlmConfig] = useState<LLMConfigState>({
    url: '',
    authToken: '',
    model: '',
    useAnthropic: true,
    extraBody: '',
  });
  const [llmConfigPath, setLlmConfigPath] = useState('');
  const [llmResolvedVia, setLlmResolvedVia] = useState('');
  const [llmResolvedUrl, setLlmResolvedUrl] = useState('');
  const [llmProtocol, setLlmProtocol] = useState('');
  const [llmConfigLoading, setLlmConfigLoading] = useState(false);
  const [llmConfigSaving, setLlmConfigSaving] = useState(false);
  const [llmConfigMessage, setLlmConfigMessage] = useState('');
  const [reviewTask, setReviewTask] = useState<ReviewTask | null>(null);
  const [reviewForm, setReviewForm] = useState<ReviewFormState>({
    encodedRepo: '',
    reviewMode: 'full',
    baseRef: '',
    targetRef: '',
    commitRef: '',
    encodedBranch: '',
    background: '',
    format: 'text',
    timeout: '10',
    concurrency: '8',
    rulePath: '',
  });
  const [reviewSubmitting, setReviewSubmitting] = useState(false);
  const [reviewMessage, setReviewMessage] = useState('');
  const [branches, setBranches] = useState<string[]>([]);
  const [branchesLoading, setBranchesLoading] = useState(false);
  const [localClonePath, setLocalClonePath] = useState<string>('');
  const [repoActionMessage, setRepoActionMessage] = useState('');
  const [auditRepoFilter, setAuditRepoFilter] = useState<string>('');
  const [auditDateFilter, setAuditDateFilter] = useState<string>('');
  const [reports, setReports] = useState<AuditReport[]>([]);
  const [reportsLoading, setReportsLoading] = useState(false);
  const [reportRepoFilter, setReportRepoFilter] = useState<string>('');
  const [reportDateFilter, setReportDateFilter] = useState<string>('');
  const [expandedReport, setExpandedReport] = useState<string>('');
  const [generatingReport, setGeneratingReport] = useState(false);
  const [reportMessage, setReportMessage] = useState('');
  const [issueLists, setIssueLists] = useState<IssueListData[]>([]);
  const [issueListsLoading, setIssueListsLoading] = useState(false);
  const [issueListRepoFilter, setIssueListRepoFilter] = useState<string>('');
  const [issueListFileFilter, setIssueListFileFilter] = useState<string>('');
  const [issueListSeverityFilter, setIssueListSeverityFilter] = useState<string>('');
  const [issueListCategoryFilter, setIssueListCategoryFilter] = useState<string>('');
  const [selectedIssueListId, setSelectedIssueListId] = useState<string>('');
  const [syncingRepos, setSyncingRepos] = useState<Set<string>>(new Set());
  const [deletingSessions, setDeletingSessions] = useState<Set<string>>(new Set());

  const menuItems = [
    { key: 'overview' as MenuKey, icon: 'fa-chart-pie' },
    { key: 'repositories' as MenuKey, icon: 'fa-code-branch' },
    { key: 'newReview' as MenuKey, icon: 'fa-plus-circle' },
    { key: 'sessions' as MenuKey, icon: 'fa-clock-rotate-left' },
    { key: 'sessionDetail' as MenuKey, icon: 'fa-file-lines' },
    { key: 'reports' as MenuKey, icon: 'fa-chart-column' },
    { key: 'issueLists' as MenuKey, icon: 'fa-list-check' },
    { key: 'rules' as MenuKey, icon: 'fa-shield-halved' },
    { key: 'settings' as MenuKey, icon: 'fa-gear' },
  ];

  const formatToolArguments = (value: string) => {
    if (!value) return '';
    try {
      return JSON.stringify(JSON.parse(value), null, 2);
    } catch {
      return value;
    }
  };

  const getSessionStatusView = (status?: string, durationSec?: number, llmFailures?: number) => {
    const normalized = status || (durationSec ? ((llmFailures ?? 0) === 0 ? 'completed' : 'failed') : 'running');
    if (normalized === 'completed') return { status: normalized, text: '成功', className: 'text-green-600' };
    if (normalized === 'completed_with_warnings') return { status: normalized, text: '部分完成', className: 'text-amber-600' };
    if (normalized === 'failed') return { status: normalized, text: '失败', className: 'text-red-600' };
    return { status: normalized, text: '运行中', className: 'text-amber-600' };
  };

  const getRiskView = (riskLevel?: string) => {
    const level = (riskLevel || '').toLowerCase();
    if (level === 'critical') return { text: 'Critical', badge: 'bg-red-100 text-red-700 border-red-200', panel: 'border-red-100 bg-red-50' };
    if (level === 'high') return { text: 'High', badge: 'bg-orange-100 text-orange-700 border-orange-200', panel: 'border-orange-100 bg-orange-50' };
    if (level === 'medium') return { text: 'Medium', badge: 'bg-amber-100 text-amber-700 border-amber-200', panel: 'border-amber-100 bg-amber-50' };
    return { text: riskLevel || 'Low', badge: 'bg-emerald-100 text-emerald-700 border-emerald-200', panel: 'border-emerald-100 bg-emerald-50' };
  };

  const renderStructuredReport = (structured: AuditReportStructured) => {
    const risk = getRiskView(structured.riskLevel);
    return (
      <div className="mt-4 space-y-4">
        <div className={`rounded-2xl border p-5 ${risk.panel}`}>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="text-lg font-semibold text-slate-900">{structured.title || '审计综合诊断报告'}</div>
              <div className="mt-2 whitespace-pre-wrap text-sm leading-6 text-slate-700">{structured.executiveSummary}</div>
            </div>
            <span className={`rounded-full border px-3 py-1 text-xs font-semibold ${risk.badge}`}>{risk.text}</span>
          </div>
          {structured.riskReason && <div className="mt-3 text-sm leading-6 text-slate-600">{structured.riskReason}</div>}
        </div>

        {structured.categories && structured.categories.length > 0 && (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {structured.categories.map((category, index) => {
              const categoryRisk = getRiskView(category.riskLevel);
              return (
                <div key={`${category.name}-${index}`} className="rounded-2xl border border-slate-200 bg-white p-4">
                  <div className="flex items-center justify-between gap-3">
                    <div className="font-semibold text-slate-900">{category.name}</div>
                    <span className={`rounded-full border px-2.5 py-1 text-xs font-medium ${categoryRisk.badge}`}>{categoryRisk.text}</span>
                  </div>
                  <div className="mt-2 text-sm leading-6 text-slate-600">{category.description}</div>
                  {category.count ? <div className="mt-3 text-xs text-slate-500">{category.count} 条相关问题</div> : null}
                </div>
              );
            })}
          </div>
        )}

        {structured.keyModules && structured.keyModules.length > 0 && (
          <div className="rounded-2xl border border-slate-200 bg-white p-5">
            <div className="text-base font-semibold text-slate-900">重点文件或模块</div>
            <div className="mt-4 space-y-3">
              {structured.keyModules.map((module, index) => {
                const moduleRisk = getRiskView(module.riskLevel);
                return (
                  <div key={`${module.path}-${index}`} className="rounded-xl border border-slate-100 bg-slate-50 p-4">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className={`rounded-full border px-2.5 py-1 text-xs font-medium ${moduleRisk.badge}`}>{moduleRisk.text}</span>
                      <span className="break-all text-sm font-semibold text-slate-900">{module.path}</span>
                    </div>
                    <div className="mt-2 text-sm leading-6 text-slate-700">{module.finding}</div>
                    <div className="mt-2 text-sm leading-6 text-slate-500">建议：{module.suggestion}</div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {structured.topFixes && structured.topFixes.length > 0 && (
          <div className="rounded-2xl border border-slate-200 bg-white p-5">
            <div className="text-base font-semibold text-slate-900">Top 优先修复项</div>
            <div className="mt-4 overflow-x-auto">
              <table className="w-full min-w-[900px] text-left text-sm">
                <thead className="border-b border-slate-200 text-xs uppercase text-slate-500">
                  <tr>
                    <th className="px-3 py-2">优先级</th>
                    <th className="px-3 py-2">风险</th>
                    <th className="px-3 py-2">问题</th>
                    <th className="px-3 py-2">文件</th>
                    <th className="px-3 py-2">影响</th>
                    <th className="px-3 py-2">建议</th>
                  </tr>
                </thead>
                <tbody>
                  {structured.topFixes.map((fix, index) => {
                    const fixRisk = getRiskView(fix.riskLevel);
                    return (
                      <tr key={`${fix.title}-${index}`} className="border-b border-slate-100 align-top">
                        <td className="px-3 py-3 font-semibold text-slate-900">#{fix.priority || index + 1}</td>
                        <td className="px-3 py-3"><span className={`rounded-full border px-2 py-1 text-xs font-medium ${fixRisk.badge}`}>{fixRisk.text}</span></td>
                        <td className="px-3 py-3 font-medium text-slate-900">{fix.title}</td>
                        <td className="px-3 py-3 break-all text-slate-600">{fix.file || '-'}</td>
                        <td className="px-3 py-3 text-slate-600">{fix.impact}</td>
                        <td className="px-3 py-3 text-slate-600">{fix.suggestion}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {structured.roadmap && structured.roadmap.length > 0 && (
          <div className="rounded-2xl border border-slate-200 bg-white p-5">
            <div className="text-base font-semibold text-slate-900">修复路线</div>
            <div className="mt-4 grid gap-3 md:grid-cols-3">
              {structured.roadmap.map((phase, index) => (
                <div key={`${phase.phase}-${index}`} className="rounded-xl border border-slate-100 bg-slate-50 p-4">
                  <div className="text-sm font-semibold text-slate-900">{phase.phase}</div>
                  <div className="mt-1 text-sm text-slate-600">{phase.goal}</div>
                  {phase.items && phase.items.length > 0 && (
                    <div className="mt-3 space-y-2 text-sm text-slate-600">
                      {phase.items.map((item, itemIndex) => <div key={`${item}-${itemIndex}`}>- {item}</div>)}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {structured.manualReview && structured.manualReview.length > 0 && (
          <div className="rounded-2xl border border-slate-200 bg-white p-5">
            <div className="text-base font-semibold text-slate-900">需要人工复核的点</div>
            <div className="mt-3 grid gap-2 text-sm text-slate-600 md:grid-cols-2">
              {structured.manualReview.map((item, index) => <div key={`${item}-${index}`} className="rounded-lg bg-slate-50 p-3">{item}</div>)}
            </div>
          </div>
        )}
      </div>
    );
  };

  const extractReviewIssues = (detail: SessionDetail | null): ReviewIssue[] => {
    if (!detail) return [];

    const issues: ReviewIssue[] = [];
    const seen = new Set<string>();

    for (const file of detail.files || []) {
      for (const cards of Object.values(file.tasks || {})) {
        for (const card of cards || []) {
          for (const toolCall of card.toolCalls || []) {
            if (toolCall.name !== 'code_comment' || !toolCall.arguments) continue;

            let payload: any;
            try {
              payload = JSON.parse(toolCall.arguments);
            } catch {
              continue;
            }

            let comments = Array.isArray(payload.comments) ? payload.comments : [];
            if (typeof payload.comments === 'string') {
              try {
                const parsedComments = JSON.parse(payload.comments);
                comments = Array.isArray(parsedComments) ? parsedComments : [];
              } catch {
                comments = [];
              }
            }

            for (const comment of comments) {
              if (!comment || typeof comment !== 'object') continue;
              const path = typeof payload.path === 'string' && payload.path ? payload.path : file.filePath;
              const content = typeof comment.content === 'string' ? comment.content.trim() : '';
              if (!path || !content) continue;

              const issue: ReviewIssue = {
                path,
                content,
                suggestionCode: typeof comment.suggestion_code === 'string' ? comment.suggestion_code.trim() : '',
                existingCode: typeof comment.existing_code === 'string' ? comment.existing_code.trim() : '',
                thinking: typeof comment.thinking === 'string' ? comment.thinking.trim() : '',
              };
              const key = `${issue.path}\n${issue.content}\n${issue.suggestionCode}`;
              if (seen.has(key)) continue;
              seen.add(key);
              issues.push(issue);
            }
          }
        }
      }
    }

    return issues;
  };

  const menuPathMap: Record<MenuKey, string> = {
    overview: '/admin',
    repositories: '/admin/repositories',
    newReview: '/admin/new-task',
    sessions: '/admin/task-history',
    sessionDetail: '/admin/task-detail',
    reports: '/admin/reports',
    issueLists: '/admin/issues',
    rules: '/admin/rules',
    settings: '/admin/settings',
  };

  const pathToMenuMap: Record<string, MenuKey> = {};
  for (const [key, path] of Object.entries(menuPathMap)) {
    pathToMenuMap[path] = key as MenuKey;
  }

  useEffect(() => {
    const menu = pathToMenuMap[location.pathname];
    if (menu) setActiveMenu(menu);
    const params = new URLSearchParams(location.search);
    const session = params.get('session');
    const repo = params.get('repo');
    if (session) setSelectedSession(session);
    if (repo) setSelectedRepo(repo);
  }, [location.pathname, location.search]);

  useEffect(() => {
    if (activeMenu !== 'newReview' || !reviewForm.encodedRepo) {
      setBranches([]);
      setLocalClonePath('');
      return;
    }

    const repo = repos.find(r => r.encodedPath === reviewForm.encodedRepo);
    if (repo && repo.repoPath) {
      setLocalClonePath(repo.repoPath);
    }

    let cancelled = false;

    const loadBranches = async () => {
      setBranchesLoading(true);
      try {
        const res = await fetch(`/api/branches?repo=${encodeURIComponent(reviewForm.encodedRepo)}`);
        const data = await res.json();
        if (!cancelled && data.branches) {
          setBranches(data.branches);
          if (!reviewForm.encodedBranch && data.branches.length > 0) {
            setReviewForm(prev => ({ ...prev, encodedBranch: data.branches[0], reviewMode: 'branch-diff', targetRef: data.branches[0], baseRef: prev.baseRef || 'main' }));
          }
        }
      } catch {
        // ignore
      } finally {
        if (!cancelled) {
          setBranchesLoading(false);
        }
      }
    };

    loadBranches();
    return () => { cancelled = true; };
  }, [activeMenu, reviewForm.encodedRepo, repos]);

  const handleMenuChange = (key: MenuKey) => {
    setActiveMenu(key);
    if (key === 'sessions') {
      setSelectedSession('');
      navigate(menuPathMap[key], { replace: true });
    } else if (key === 'sessionDetail') {
      const params = new URLSearchParams();
      if (selectedRepo) params.set('repo', selectedRepo);
      if (selectedSession) params.set('session', selectedSession);
      navigate(`${menuPathMap[key]}?${params.toString()}`, { replace: true });
    } else {
      navigate(menuPathMap[key], { replace: true });
    }
  };

  useEffect(() => {
    if (activeMenu !== 'sessions') {
      return;
    }

    let cancelled = false;

    const loadAllSessions = async () => {
      setSessionsLoading(true);
      try {
        const response = await fetch('/api/sessions');
        const data = await response.json();
        if (!cancelled) {
          const nextSessions = Array.isArray(data.sessions) ? data.sessions : [];
          setSessions(nextSessions);
          if (nextSessions.length > 0 && !selectedSession) {
            setSelectedSession(nextSessions[0]?.sessionID || '');
          }
        }
      } catch {
        if (!cancelled) {
          setSessions([]);
        }
      } finally {
        if (!cancelled) {
          setSessionsLoading(false);
        }
      }
    };

    loadAllSessions();
    return () => { cancelled = true; };
  }, [activeMenu]);

  useEffect(() => {
    if (!selectedRepo || !selectedSession) {
      setSessionDetail(null);
      setShowIssueList(false);
      setIssueListExpanded(false);
      setReviewFilesExpanded(false);
      return;
    }

    setShowIssueList(false);
    setIssueListExpanded(false);
    setReviewFilesExpanded(false);

    let cancelled = false;

    const loadSessionDetail = async () => {
      if (!sessionDetail) {
        setSessionDetailLoading(true);
      }
      try {
        const response = await fetch(`/api/repos/${encodeURIComponent(selectedRepo)}/sessions/${encodeURIComponent(selectedSession)}`);
        const data = await response.json();
        if (!cancelled) {
          setSessionDetail(data.session ?? null);
        }
      } catch {
        if (!cancelled) {
          setSessionDetail(null);
        }
      } finally {
        if (!cancelled) {
          setSessionDetailLoading(false);
        }
      }
    };

    loadSessionDetail();
    const timer = window.setInterval(() => {
      if (sessionDetail?.summary.status === 'running') {
        loadSessionDetail();
      }
    }, 2000);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [selectedRepo, selectedSession, sessionDetail?.summary.status]);

  useEffect(() => {
    if (activeMenu !== 'reports') {
      return;
    }

    let cancelled = false;

    const loadReports = async () => {
      setReportsLoading(true);
      try {
        const params = new URLSearchParams();
        if (reportRepoFilter) params.set('repo', reportRepoFilter);
        if (reportDateFilter) params.set('date', reportDateFilter);
        const response = await fetch(`/api/reports${params.toString() ? `?${params.toString()}` : ''}`);
        const data = await response.json();
        if (!cancelled) {
          setReports(Array.isArray(data.reports) ? data.reports : []);
        }
      } catch {
        if (!cancelled) {
          setReports([]);
        }
      } finally {
        if (!cancelled) {
          setReportsLoading(false);
        }
      }
    };

    loadReports();
    return () => {
      cancelled = true;
    };
  }, [activeMenu, reportRepoFilter, reportDateFilter]);

  useEffect(() => {
    if (activeMenu !== 'issueLists') {
      return;
    }

    let cancelled = false;

    const loadIssueLists = async () => {
      setIssueListsLoading(true);
      try {
        const params = new URLSearchParams();
        if (issueListRepoFilter) params.set('repo', issueListRepoFilter);
        const response = await fetch(`/api/issues${params.toString() ? `?${params.toString()}` : ''}`);
        const data = await response.json();
        if (!cancelled) {
          setIssueLists(Array.isArray(data.issueLists) ? data.issueLists : []);
          if (!selectedIssueListId && data.issueLists?.length > 0) {
            setSelectedIssueListId(data.issueLists[0].id);
          }
        }
      } catch {
        if (!cancelled) {
          setIssueLists([]);
        }
      } finally {
        if (!cancelled) {
          setIssueListsLoading(false);
        }
      }
    };

    loadIssueLists();
    return () => {
      cancelled = true;
    };
  }, [activeMenu, issueListRepoFilter]);

  useEffect(() => {
    if (activeMenu !== 'rules') {
      return;
    }

    let cancelled = false;

    const loadRules = async () => {
      setRulesLoading(true);
      try {
        const response = await fetch('/api/rules');
        const data = await response.json();
        if (!cancelled) {
          setRuleLayers(Array.isArray(data.layers) ? data.layers : []);
        }
      } catch {
        if (!cancelled) {
          setRuleLayers([]);
        }
      } finally {
        if (!cancelled) {
          setRulesLoading(false);
        }
      }
    };

    loadRules();
    return () => {
      cancelled = true;
    };
  }, [activeMenu]);

  useEffect(() => {
    if (activeMenu !== 'settings') {
      return;
    }

    let cancelled = false;

    const loadLLMConfig = async () => {
      setLlmConfigLoading(true);
      setLlmConfigMessage('');
      try {
        const response = await fetch('/api/config/llm');
        const data = await response.json();
        if (!cancelled) {
          setLlmConfig({
            url: data.config?.url ?? '',
            authToken: data.config?.authToken ?? '',
            model: data.config?.model ?? '',
            useAnthropic: data.config?.useAnthropic ?? true,
            extraBody: data.config?.extraBody ?? '',
          });
          setLlmConfigPath(data.configPath ?? '');
          setLlmResolvedVia(data.resolvedVia ?? '');
          setLlmResolvedUrl(data.resolvedUrl ?? '');
          setLlmProtocol(data.protocol ?? '');
        }
      } catch {
        if (!cancelled) {
          setLlmConfigMessage(t('admin.settings.loadFailed'));
        }
      } finally {
        if (!cancelled) {
          setLlmConfigLoading(false);
        }
      }
    };

    loadLLMConfig();
    return () => {
      cancelled = true;
    };
  }, [activeMenu, t]);

  useEffect(() => {
    if (!reviewTask?.taskID || reviewTask.status !== 'running') {
      return;
    }

    const timer = window.setInterval(async () => {
      try {
        const response = await fetch(`/api/reviews/${encodeURIComponent(reviewTask.taskID)}`);
        const data = await response.json();
        setReviewTask(data);
      } catch {
      }
    }, 2000);

    return () => window.clearInterval(timer);
  }, [reviewTask]);

  const handleSaveLLMConfig = async () => {
    setLlmConfigSaving(true);
    setLlmConfigMessage('');
    try {
      const response = await fetch('/api/config/llm', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(llmConfig),
      });
      const data = await response.json();
      if (!response.ok) {
        setLlmConfigMessage(data.error || t('admin.settings.saveFailed'));
        return;
      }
      setLlmConfigPath(data.configPath ?? '');
      setLlmResolvedVia(data.resolvedVia ?? '');
      setLlmResolvedUrl(data.resolvedUrl ?? '');
      setLlmProtocol(data.protocol ?? '');
      setLlmConfigMessage(t('admin.settings.saveSuccess'));
    } catch {
      setLlmConfigMessage(t('admin.settings.saveFailed'));
    } finally {
      setLlmConfigSaving(false);
    }
  };

  const handleOpenRepoSessions = (encodedPath: string) => {
    setSelectedRepo(encodedPath);
    handleMenuChange('sessions');
  };

  const handleStartReview = async () => {
    if (!reviewForm.encodedRepo) {
      setReviewMessage(t('admin.review.repoRequired'));
      return;
    }
    if (reviewForm.reviewMode === 'branch-diff' && (!reviewForm.baseRef.trim() || !reviewForm.targetRef.trim())) {
      setReviewMessage(t('admin.review.branchRequired'));
      return;
    }
    if (reviewForm.reviewMode === 'commit' && !reviewForm.commitRef.trim()) {
      setReviewMessage(t('admin.review.commitRequired'));
      return;
    }
    setReviewSubmitting(true);
    setReviewMessage('');
    try {
      const response = await fetch('/api/reviews', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          encodedRepo: reviewForm.encodedRepo,
          encodedBranch: reviewForm.encodedBranch || '',
          reviewMode: reviewForm.reviewMode,
          baseRef: reviewForm.baseRef,
          targetRef: reviewForm.targetRef,
          commitRef: reviewForm.commitRef,
          background: reviewForm.background,
          format: reviewForm.format,
          timeout: reviewForm.timeout,
          concurrency: reviewForm.concurrency,
          rulePath: reviewForm.rulePath,
        }),
      });
      const data = await response.json();
      if (!response.ok) {
        setReviewMessage(data.error || t('admin.review.submitFailed'));
        return;
      }
      setReviewTask(data);
      setSelectedRepo(data.encodedRepo);
      setReviewMessage(t('admin.review.submitAccepted'));
    } catch {
      setReviewMessage(t('admin.review.submitFailed'));
    } finally {
      setReviewSubmitting(false);
    }
  };

  const handleGenerateReport = async () => {
    if (!selectedRepo || !selectedSession || !sessionDetail) return;
    const status = sessionDetail.summary.status;
    if (status !== 'completed' && status !== 'completed_with_warnings') {
      setReportMessage(t('admin.reports.completedOnly'));
      return;
    }
    setGeneratingReport(true);
    setReportMessage('');
    try {
      const response = await fetch(`/api/repos/${encodeURIComponent(selectedRepo)}/sessions/${encodeURIComponent(selectedSession)}/report`, {
        method: 'POST',
      });
      const data = await response.json();
      if (!response.ok) {
        setReportMessage(data.error || t('admin.reports.generateFailed'));
        return;
      }
      setReportMessage(t('admin.reports.generateSuccess'));
      setReports((prev) => [data, ...prev.filter((item) => item.id !== data.id)]);
      setExpandedReport(data.id);
      setActiveMenu('reports');
      navigate(menuPathMap.reports, { replace: true });
    } catch {
      setReportMessage(t('admin.reports.generateFailed'));
    } finally {
      setGeneratingReport(false);
    }
  };

  const handleGenerateIssueList = async () => {
    if (!selectedRepo || !selectedSession || !sessionDetail) return;
    const status = sessionDetail.summary.status;
    if (status !== 'completed' && status !== 'completed_with_warnings') {
      setReportMessage(t('admin.issues.completedOnly'));
      return;
    }
    setGeneratingReport(true);
    setReportMessage('');
    try {
      const response = await fetch(`/api/repos/${encodeURIComponent(selectedRepo)}/sessions/${encodeURIComponent(selectedSession)}/issues`, {
        method: 'POST',
      });
      const data = await response.json();
      if (!response.ok) {
        setReportMessage(data.error || t('admin.issues.generateFailed'));
        return;
      }
      setReportMessage(t('admin.issues.generateSuccess'));
      setIssueLists((prev) => [data, ...prev.filter((item) => item.id !== data.id)]);
      setSelectedIssueListId(data.id);
      setActiveMenu('issueLists');
      navigate(menuPathMap.issueLists, { replace: true });
    } catch {
      setReportMessage(t('admin.issues.generateFailed'));
    } finally {
      setGeneratingReport(false);
    }
  };

  useEffect(() => {
    let cancelled = false;
    const loadRepos = async () => {
      setReposLoading(true);
      try {
        const response = await fetch('/api/repos');
        const data = await response.json();
        if (!cancelled) {
          setRepos(Array.isArray(data.repos) ? data.repos : []);
        }
      } catch {
        if (!cancelled) {
          setRepos([]);
        }
      } finally {
        if (!cancelled) {
          setReposLoading(false);
        }
      }
    };
    loadRepos();
    return () => { cancelled = true; };
  }, []);

  const selectedRepoInfo = repos.find((repo) => repo.encodedPath === selectedRepo);
  const detailRepoName = sessionDetail?.summary.repoName || selectedRepoInfo?.displayName || selectedRepo || '';
  const detailRepoPath = sessionDetail?.summary.repoPath || sessionDetail?.summary.cwd || selectedRepoInfo?.repoPath || '';

  const statCards = [
    {
      title: t('admin.stats.repos'),
      value: String(repos.length),
      detail: t('admin.stats.reposDetail'),
    },
    {
      title: t('admin.stats.sessions'),
      value: String(sessions.length),
      detail: t('admin.stats.sessionsDetail'),
    },
    {
      title: t('admin.stats.rules'),
      value: '4',
      detail: t('admin.stats.rulesDetail'),
    },
    {
      title: t('admin.stats.models'),
      value: sessionDetail?.summary.model ? '1' : '0',
      detail: t('admin.stats.modelsDetail'),
    },
  ];

  const sectionContent: Record<MenuKey, { title: string; description: string; cards: Array<{ title: string; body: string }> }> = {
    overview: {
      title: t('admin.menu.overview'),
      description: t('admin.section.overviewDesc'),
      cards: [
        { title: t('admin.card.pendingReviews'), body: t('admin.card.pendingReviewsDesc') },
        { title: t('admin.card.recentActivity'), body: t('admin.card.recentActivityDesc') },
        { title: t('admin.card.systemHealth'), body: t('admin.card.systemHealthDesc') },
      ],
    },
    repositories: {
      title: t('admin.menu.repositories'),
      description: t('admin.section.repositoriesDesc'),
      cards: [
        { title: t('admin.card.repoAccess'), body: t('admin.card.repoAccessDesc') },
        { title: t('admin.card.repoStatus'), body: t('admin.card.repoStatusDesc') },
        { title: t('admin.card.repoBinding'), body: t('admin.card.repoBindingDesc') },
      ],
    },
    newReview: {
      title: t('admin.menu.newReview'),
      description: '',
      cards: [],
    },
    sessions: {
      title: t('admin.menu.sessions'),
      description: '',
      cards: [],
    },
    sessionDetail: {
      title: t('admin.menu.sessionDetail'),
      description: '',
      cards: [],
    },
    reports: {
      title: t('admin.menu.reports'),
      description: t('admin.section.reportsDesc'),
      cards: [],
    },
    issueLists: {
      title: t('admin.menu.issueLists'),
      description: t('admin.section.issueListsDesc'),
      cards: [],
    },
    rules: {
      title: t('admin.menu.rules'),
      description: t('admin.section.rulesDesc'),
      cards: [
        { title: t('admin.card.ruleLibrary'), body: t('admin.card.ruleLibraryDesc') },
        { title: t('admin.card.ruleScope'), body: t('admin.card.ruleScopeDesc') },
        { title: t('admin.card.rulePreview'), body: t('admin.card.rulePreviewDesc') },
      ],
    },
    settings: {
      title: t('admin.menu.settings'),
      description: t('admin.section.settingsDesc'),
      cards: [
        { title: t('admin.card.modelConfig'), body: t('admin.card.modelConfigDesc') },
        { title: t('admin.card.languagePolicy'), body: t('admin.card.languagePolicyDesc') },
        { title: t('admin.card.auditPolicy'), body: t('admin.card.auditPolicyDesc') },
      ],
    },
  };

  const currentSection = sectionContent[activeMenu];
  const detailPageTitle = activeMenu === 'sessionDetail' && (detailRepoName || detailRepoPath)
    ? `${detailRepoName || '-'}${detailRepoPath ? ` · ${detailRepoPath}` : ''}`
    : currentSection.title;

  const showStatCards = activeMenu === 'overview';

  const [editingRepoRow, setEditingRepoRow] = useState<number | null>(null);
  const [newRepoRow, setNewRepoRow] = useState<{ repoName: string; localPath: string; remoteURL: string; token: string }>({ repoName: '', localPath: '', remoteURL: '', token: '' });
  const [addingRepo, setAddingRepo] = useState(false);
  const [editingTokenRepo, setEditingTokenRepo] = useState<string | null>(null);
  const [tokenInput, setTokenInput] = useState('');
  
  const handleSaveToken = async (encodedRepo: string) => {
    try {
      await fetch(`/api/repos/${encodeURIComponent(encodedRepo)}/token`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: tokenInput }),
      });
      setEditingTokenRepo(null);
    } catch {
    }
  };

  const handleSaveNewRepo = async () => {
    if (!newRepoRow.repoName.trim()) {
      setRepoActionMessage('仓库名称不能为空');
      return;
    }
    if (!newRepoRow.localPath.trim()) {
      setRepoActionMessage('本地地址不能为空');
      return;
    }
    setAddingRepo(true);
    setRepoActionMessage('');
    try {
      const response = await fetch('/api/repos', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          repoName: newRepoRow.repoName.trim(),
          localPath: newRepoRow.localPath.trim(),
          remoteURL: newRepoRow.remoteURL?.trim() || '',
          token: newRepoRow.token?.trim() || '',
        }),
      });
      const data: AddRepoResponse = await response.json();
      if (!response.ok) {
        setRepoActionMessage(data.error || t('admin.repo.addFailed'));
        return;
      }
      setRepos((prev) => [...prev, data]);
      setNewRepoRow({ repoName: '', localPath: '', remoteURL: '', token: '' });
      setEditingRepoRow(null);
      setRepoActionMessage(t('admin.repo.addSuccess'));
    } catch {
      setRepoActionMessage(t('admin.repo.addFailed'));
    } finally {
      setAddingRepo(false);
    }
  };

  const handleSyncRepo = async (encodedPath: string) => {
    setSyncingRepos((prev) => new Set(prev).add(encodedPath));
    try {
      const response = await fetch(`/api/repos/${encodedPath}/sync`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      const data = await response.json();
      if (!response.ok) {
        setRepoActionMessage(data.error || '同步失败');
        return;
      }
      setRepos((prev) =>
        prev.map((repo) =>
          repo.encodedPath === encodedPath
            ? { ...repo, repoPath: data.localPath, fileCount: data.fileCount }
            : repo
        )
      );
      setRepoActionMessage('同步完成');
    } catch {
      setRepoActionMessage('同步失败，请重试');
    } finally {
      setSyncingRepos((prev) => {
        const next = new Set(prev);
        next.delete(encodedPath);
        return next;
      });
    }
  };

  const handleDeleteRepo = async (encodedPath: string) => {
    if (!confirm('确定要删除这个仓库吗？删除后相关的任务记录也会被清空。')) {
      return;
    }
    try {
      const response = await fetch(`/api/repos/${encodedPath}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        const data = await response.json();
        setRepoActionMessage(data.error || '删除失败');
        return;
      }
      setRepos((prev) => prev.filter((repo) => repo.encodedPath !== encodedPath));
      setRepoActionMessage('删除成功');
    } catch {
      setRepoActionMessage('删除失败');
    }
  };

  const handleDeleteSession = async (session: SessionSummary) => {
    const repo = session.encodedRepo || repos.find((item) => item.repoPath === session.cwd || item.repoPath === session.repoPath)?.encodedPath;
    if (!repo) {
      setRepoActionMessage('无法定位任务所属仓库');
      return;
    }
    if ((session.status || '') === 'running') {
      setRepoActionMessage('运行中的任务不能删除');
      return;
    }
    if (!confirm('确定要删除这条任务记录吗？')) {
      return;
    }
    const key = `${repo}:${session.sessionID}`;
    setDeletingSessions((prev) => new Set(prev).add(key));
    try {
      const response = await fetch(`/api/repos/${encodeURIComponent(repo)}/sessions/${encodeURIComponent(session.sessionID)}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        const data = await response.json();
        setRepoActionMessage(data.error || '删除任务失败');
        return;
      }
      setSessions((prev) => prev.filter((item) => item.sessionID !== session.sessionID || (item.encodedRepo || '') !== repo));
      setRepoActionMessage('任务记录已删除');
      if (selectedSession === session.sessionID) {
        setSelectedSession('');
        setSessionDetail(null);
      }
    } catch {
      setRepoActionMessage('删除任务失败');
    } finally {
      setDeletingSessions((prev) => {
        const next = new Set(prev);
        next.delete(key);
        return next;
      });
    }
  };

  const renderDataPanel = () => {
    if (activeMenu === 'repositories') {
      return (
        <div className="rounded-2xl border border-slate-200 bg-white shadow-sm">
          <div className="flex items-center justify-between border-b border-slate-100 px-5 py-4">
            <div className="text-base font-semibold text-slate-900">{t('admin.data.repositoriesTitle')}</div>
            <button
              onClick={() => { setEditingRepoRow(-1); setNewRepoRow({ repoName: '', localPath: '', remoteURL: '', token: '' }); }}
              className="rounded-lg border border-brand-500/30 bg-brand-500/10 px-4 py-2 text-sm font-medium text-brand-600 hover:bg-brand-500/20"
            >
              + {t('admin.repo.addAction')}
            </button>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-100 bg-slate-50">
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase tracking-[0.15em] text-slate-500">仓库名称</th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase tracking-[0.15em] text-slate-500">本地地址</th>
                  <th className="px-5 py-3 text-left text-xs font-medium uppercase tracking-[0.15em] text-slate-500">远程地址</th>
                  <th className="px-5 py-3 text-center text-xs font-medium uppercase tracking-[0.15em] text-slate-500">文件数</th>
                  <th className="px-5 py-3 text-center text-xs font-medium uppercase tracking-[0.15em] text-slate-500">Token</th>
                  <th className="px-5 py-3 text-center text-xs font-medium uppercase tracking-[0.15em] text-slate-500">任务数</th>
                  <th className="px-5 py-3 text-center text-xs font-medium uppercase tracking-[0.15em] text-slate-500">同步</th>
                  <th className="px-5 py-3 text-right text-xs font-medium uppercase tracking-[0.15em] text-slate-500">操作</th>
                </tr>
              </thead>
              <tbody>
                {repos.map((repo) => (
                  <tr
                    key={repo.encodedPath}
                    className="border-b border-slate-100 hover:bg-slate-50"
                  >
                    <td className="px-5 py-3 font-medium text-slate-900">{repo.displayName}</td>
                    <td className="px-5 py-3 text-slate-600 break-all">{repo.repoPath || '-'}</td>
                    <td className="px-5 py-3 text-slate-600 break-all">{repo.remoteURL || '-'}</td>
                    <td className="px-5 py-3 text-center text-slate-600">{repo.fileCount || 0}</td>
                    <td className="px-5 py-3 text-center">
                      <div className="relative inline-block">
                        <button
                          onClick={() => { setEditingTokenRepo(repo.encodedPath); setTokenInput(''); }}
                          className={`transition-colors ${repo.hasToken ? 'text-green-500 hover:text-green-600' : 'text-slate-400 hover:text-brand-500'}`}
                          title={repo.hasToken ? '已设置 Token（点击修改）' : '未设置 Token（点击设置）'}
                        >
                          <i className={`fa-solid ${repo.hasToken ? 'fa-key' : 'fa-key'}`}></i>
                          {repo.hasToken && <span className="absolute -top-1 -right-1 h-2 w-2 rounded-full bg-green-500"></span>}
                        </button>
                        {editingTokenRepo === repo.encodedPath && (
                          <div className="absolute right-0 top-full z-10 mt-1 w-48 rounded-lg border border-slate-200 bg-white p-2 shadow-lg">
                            <input
                              value={tokenInput}
                              onChange={(e) => setTokenInput(e.target.value)}
                              placeholder="ghp_..."
                              className="w-full rounded border border-slate-300 px-2 py-1 text-xs outline-none"
                            />
                            <div className="mt-1 flex gap-1">
                              <button
                                onClick={() => { setEditingTokenRepo(null); }}
                                className="flex-1 rounded border border-slate-200 px-2 py-1 text-xs hover:bg-slate-50"
                              >
                                取消
                              </button>
                              <button
                                onClick={() => { handleSaveToken(repo.encodedPath); }}
                                className="flex-1 rounded bg-brand-500 px-2 py-1 text-xs font-medium text-slate-950"
                              >
                                保存
                              </button>
                            </div>
                          </div>
                        )}
                      </div>
                    </td>
                    <td className="px-5 py-3 text-center">
                      <button
                        onClick={() => handleOpenRepoSessions(repo.encodedPath)}
                        className="text-brand-600 hover:underline"
                      >
                        {repo.sessionCount}
                      </button>
                    </td>
                    <td className="px-5 py-3 text-center">
                      {repo.remoteURL && (
                        <button
                          onClick={() => handleSyncRepo(repo.encodedPath)}
                          disabled={syncingRepos.has(repo.encodedPath)}
                          className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                            syncingRepos.has(repo.encodedPath)
                              ? 'bg-slate-200 text-slate-500 cursor-not-allowed'
                              : 'bg-blue-500 text-white hover:bg-blue-600'
                          }`}
                        >
                          {syncingRepos.has(repo.encodedPath) ? '同步中...' : '同步'}
                        </button>
                      )}
                    </td>
                    <td className="px-5 py-3 text-right">
                      <button
                        onClick={() => handleDeleteRepo(repo.encodedPath)}
                        className="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50"
                      >
                        删除
                      </button>
                    </td>
                  </tr>
                ))}
                {editingRepoRow === -1 && (
                  <tr className="border-b border-slate-100 bg-brand-500/5">
                    <td className="px-5 py-3">
                      <input
                        value={newRepoRow.repoName}
                        onChange={(e) => setNewRepoRow((prev) => ({ ...prev, repoName: e.target.value }))}
                        placeholder="仓库名称"
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-brand-500/50"
                      />
                    </td>
                    <td className="px-5 py-3">
                      <input
                        value={newRepoRow.localPath}
                        onChange={(e) => setNewRepoRow((prev) => ({ ...prev, localPath: e.target.value }))}
                        placeholder="本地地址（必填）"
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-brand-500/50"
                      />
                    </td>
                    <td className="px-5 py-3">
                      <input
                        value={newRepoRow.remoteURL}
                        onChange={(e) => setNewRepoRow((prev) => ({ ...prev, remoteURL: e.target.value }))}
                        placeholder="远程地址（可选）"
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-brand-500/50"
                      />
                    </td>
                    <td className="px-5 py-3 text-center text-slate-400">-</td>
                    <td className="px-5 py-3">
                      <input
                        value={newRepoRow.token}
                        onChange={(e) => setNewRepoRow((prev) => ({ ...prev, token: e.target.value }))}
                        placeholder="ghp_... (可选)"
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-brand-500/50"
                      />
                    </td>
                    <td className="px-5 py-3 text-center text-slate-400">-</td>
                    <td className="px-5 py-3 text-center text-slate-400">-</td>
                    <td className="px-5 py-3 text-right space-x-2">
                      <button
                        onClick={handleSaveNewRepo}
                        disabled={addingRepo || !newRepoRow.repoName.trim() || !newRepoRow.localPath.trim()}
                        className="rounded-lg bg-brand-500 px-3 py-1.5 text-xs font-medium text-slate-950 disabled:opacity-50"
                      >
                        {addingRepo ? t('admin.repo.addSubmitting') : t('admin.repo.save')}
                      </button>
                      <button
                        onClick={() => { setEditingRepoRow(null); }}
                        className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-600 hover:border-slate-300"
                      >
                        {t('admin.data.cancel')}
                      </button>
                    </td>
                  </tr>
                )}
                {!reposLoading && repos.length === 0 && editingRepoRow === null && (
                  <tr>
                    <td colSpan={4} className="px-5 py-8 text-center text-sm text-slate-500">{t('admin.data.emptyRepos')}</td>
                  </tr>
                )}
                {reposLoading && (
                  <tr>
                    <td colSpan={4} className="px-5 py-8 text-center text-sm text-slate-500">{t('admin.data.loading')}</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          {(repoActionMessage) && (
            <div className={`border-t px-5 py-3 text-sm ${repoActionMessage.includes('失败') || repoActionMessage.includes('Failed') ? 'border-red-200 bg-red-50 text-red-700' : 'border-green-200 bg-green-50 text-green-700'}`}>{repoActionMessage}</div>
          )}
        </div>
      );
    }

    if (activeMenu === 'newReview') {
      const titleText = reviewTask ? `${reviewTask.repoName} - ${reviewTask.status}` : t('admin.review.formTitle');
      return (
        <div className="grid gap-4">
          <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="text-base font-semibold text-slate-900">{titleText}</div>
                {reviewTask && reviewTask.status === 'running' && (
                  <span className="inline-flex h-2 w-2 animate-pulse rounded-full bg-green-500"></span>
                )}
              </div>
              <div className="flex items-center gap-3">
                <button onClick={handleStartReview} disabled={reviewSubmitting || !reviewForm.encodedRepo} className="rounded-xl bg-brand-500 px-5 py-2.5 text-sm font-semibold text-slate-950 transition-opacity disabled:cursor-not-allowed disabled:opacity-60">
                  {reviewSubmitting ? t('admin.review.running') : t('admin.review.execute')}
                </button>
              </div>
            </div>

            <div className="mt-5 grid gap-4 md:grid-cols-2">
              <label className="block text-sm text-slate-700">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">{t('admin.review.repoLabel')}</div>
                <select value={reviewForm.encodedRepo} onChange={(event) => setReviewForm((prev) => ({ ...prev, encodedRepo: event.target.value }))} className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none">
                  <option value="">{t('admin.review.repoPlaceholder')}</option>
                  {repos.map((repo) => (
                    <option key={repo.encodedPath} value={repo.encodedPath}>
                      {repo.displayName} · {repo.repoPath || repo.encodedPath}
                    </option>
                  ))}
                </select>
              </label>

              <label className="block text-sm text-slate-700">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">选择分支</div>
                <select
                  value={reviewForm.encodedBranch || ''}
                  onChange={(event) => setReviewForm((prev) => ({ ...prev, encodedBranch: event.target.value, targetRef: event.target.value || prev.targetRef, reviewMode: event.target.value ? 'branch-diff' : prev.reviewMode }))}
                  disabled={!reviewForm.encodedRepo}
                  className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none disabled:bg-slate-100 disabled:text-slate-400"
                >
                  <option value="">{branchesLoading ? '加载中...' : '选择分支'}</option>
                  {branches.map(branch => (
                    <option key={branch} value={branch}>{branch}</option>
                  ))}
                </select>
              </label>

              <label className="block text-sm text-slate-700">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">仓库路径</div>
                <input
                  value={localClonePath || '请选择仓库'}
                  readOnly
                  disabled
                  className="w-full rounded-xl border border-slate-300 bg-slate-50 px-4 py-3 text-sm text-slate-500 outline-none"
                />
              </label>

              <label className="block text-sm text-slate-700">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">{t('admin.review.modeLabel')}</div>
                <select value={reviewForm.reviewMode} onChange={(event) => setReviewForm((prev) => ({ ...prev, reviewMode: event.target.value, baseRef: event.target.value === 'branch-diff' && !prev.baseRef ? 'main' : prev.baseRef }))} className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none">
                  <option value="working-tree">{t('admin.review.modeWorkingTree')}</option>
                  <option value="full">{t('admin.review.modeFull')}</option>
                  <option value="branch-diff">{t('admin.review.modeBranchDiff')}</option>
                  <option value="commit">{t('admin.review.modeCommit')}</option>
                </select>
              </label>

              {reviewForm.reviewMode === 'branch-diff' && (
                <>
                  <label className="block text-sm text-slate-700">
                    <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">{t('admin.review.baseRefLabel')}</div>
                    <input value={reviewForm.baseRef} onChange={(event) => setReviewForm((prev) => ({ ...prev, baseRef: event.target.value }))} placeholder={t('admin.review.baseRefPlaceholder')} className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none" />
                  </label>

                  <label className="block text-sm text-slate-700">
                    <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">{t('admin.review.targetRefLabel')}</div>
                    <input value={reviewForm.targetRef} onChange={(event) => setReviewForm((prev) => ({ ...prev, targetRef: event.target.value }))} placeholder={t('admin.review.targetRefPlaceholder')} className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none" />
                  </label>
                </>
              )}

              {reviewForm.reviewMode === 'commit' && (
                <label className="block text-sm text-slate-700 md:col-span-2">
                  <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">{t('admin.review.commitRefLabel')}</div>
                  <input value={reviewForm.commitRef} onChange={(event) => setReviewForm((prev) => ({ ...prev, commitRef: event.target.value }))} placeholder={t('admin.review.commitRefPlaceholder')} className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none" />
                </label>
              )}

              <label className="block text-sm text-slate-700 md:col-span-2">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">审查背景</div>
                <textarea
                  value={reviewForm.background}
                  onChange={(event) => setReviewForm((prev) => ({ ...prev, background: event.target.value }))}
                  placeholder="可选：输入本次审查的业务背景、需求说明或关注重点"
                  rows={3}
                  className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none resize-none"
                />
              </label>

              <label className="block text-sm text-slate-700">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">输出格式</div>
                <select value={reviewForm.format} onChange={(event) => setReviewForm((prev) => ({ ...prev, format: event.target.value }))} className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none">
                  <option value="text">文本</option>
                  <option value="json">JSON</option>
                </select>
              </label>

              <label className="block text-sm text-slate-700">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">超时时间 (分钟)</div>
                <input
                  type="number"
                  value={reviewForm.timeout}
                  onChange={(event) => setReviewForm((prev) => ({ ...prev, timeout: event.target.value }))}
                  className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none"
                  min="1"
                  max="60"
                />
              </label>

              <label className="block text-sm text-slate-700">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">并发数</div>
                <input
                  type="number"
                  value={reviewForm.concurrency}
                  onChange={(event) => setReviewForm((prev) => ({ ...prev, concurrency: event.target.value }))}
                  className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none"
                  min="1"
                  max="32"
                />
              </label>

              <label className="block text-sm text-slate-700 md:col-span-2">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">规则文件路径</div>
                <input
                  value={reviewForm.rulePath}
                  onChange={(event) => setReviewForm((prev) => ({ ...prev, rulePath: event.target.value }))}
                  placeholder="可选：自定义规则文件路径"
                  className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none"
                />
              </label>
            </div>

            {reviewMessage && <div className="mt-4 text-sm text-slate-600">{reviewMessage}</div>}
          </div>

          {reviewTask && (
            <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
              <div className="mt-4 space-y-3 text-sm text-slate-400">
              {reviewTask.status !== 'running' && reviewTask.errorMessage && <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 p-3 text-rose-200">{reviewTask.errorMessage}</div>}
              {reviewTask.output && <pre className="max-h-64 overflow-auto whitespace-pre-wrap rounded-xl border border-slate-200 bg-slate-50 p-4 text-xs leading-6 text-slate-700">{reviewTask.output}</pre>}
              </div>
            </div>
          )}
        </div>
      );
    }

    if (activeMenu === 'sessionDetail') {
      const reviewIssues = extractReviewIssues(sessionDetail);
      const detailStatus = getSessionStatusView(sessionDetail?.summary.status, sessionDetail?.summary.durationSec, sessionDetail?.summary.llmFailures);

      return (
        <div className="space-y-6">
          <div className="flex items-center gap-3">
            <button
              onClick={() => handleMenuChange('sessions')}
              className="rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:border-slate-300 hover:text-slate-900"
            >
              &larr; {t('admin.audit.backToList')}
            </button>
            <div className="min-w-0">
              <div className="text-lg font-semibold text-slate-900">{detailRepoName || '-'}</div>
              {detailRepoPath && <div className="mt-1 break-all text-sm text-slate-500">{detailRepoPath}</div>}
            </div>
          </div>

          {sessionDetailLoading && <div className="text-sm text-slate-500">{t('admin.data.loading')}</div>}
          {!sessionDetailLoading && sessionDetail && (
            <>
              <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                <div className="text-base font-semibold text-slate-900">{t('admin.data.sessionDetailTitle')}</div>
                <div className="mt-4 grid grid-cols-2 gap-4 text-sm text-slate-600 md:grid-cols-4">
                  <div>{t('admin.data.sessionModel')}: <span className="text-slate-900">{sessionDetail.summary.model || '-'}</span></div>
                  <div>{t('admin.data.sessionBranch')}: <span className="text-slate-900">{sessionDetail.summary.gitBranch || '-'}</span></div>
                  <div>{t('admin.data.sessionDuration')}: <span className="text-slate-900">{sessionDetail.summary.durationSec?.toFixed(1) || '0.0'}s</span></div>
                  <div>{t('admin.data.sessionTokens')}: <span className="text-slate-900">{(sessionDetail.tokenUsage.totalPromptTokens || 0) + (sessionDetail.tokenUsage.totalCompletionTokens || 0)}</span></div>
                  <div>状态: <span className={`font-medium ${detailStatus.className}`}>{detailStatus.text}</span></div>
                  <div>{t('admin.data.sessionFailures')}: <span className="text-slate-900">{sessionDetail.summary.llmFailures ?? 0}</span></div>
                  <div>警告: <span className="text-slate-900">{sessionDetail.summary.warningCount ?? 0}</span></div>
                  <div>{t('admin.data.reviewMode')}: <span className="text-slate-900">{sessionDetail.summary.reviewMode || '-'}</span></div>
                </div>

                <div className="mt-5 rounded-xl border border-slate-200 bg-slate-50 p-4">
                  <div className="text-sm font-semibold text-slate-900">{t('admin.data.tokenBreakdownTitle')}</div>
                  <div className="mt-3 grid grid-cols-2 gap-3 text-xs md:grid-cols-4">
                    <div>{t('admin.data.promptTokens')}: <span className="text-slate-900">{sessionDetail.tokenUsage.totalPromptTokens ?? 0}</span></div>
                    <div>{t('admin.data.completionTokens')}: <span className="text-slate-900">{sessionDetail.tokenUsage.totalCompletionTokens ?? 0}</span></div>
                    <div>{t('admin.data.cacheReadTokens')}: <span className="text-slate-900">{sessionDetail.tokenUsage.totalCacheReadTokens ?? 0}</span></div>
                    <div>{t('admin.data.cacheWriteTokens')}: <span className="text-slate-900">{sessionDetail.tokenUsage.totalCacheWriteTokens ?? 0}</span></div>
                  </div>
                </div>
              </div>

              <div className="rounded-xl border border-emerald-100 bg-emerald-50 p-5 shadow-sm">
                <div className="flex flex-wrap items-center gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="text-base font-semibold text-slate-900">{t('admin.reports.diagnosisTitle')}</div>
                    <div className="mt-1 text-sm text-slate-500">{t('admin.reports.diagnosisDesc')}</div>
                  </div>
                  <button
                    onClick={handleGenerateReport}
                    disabled={generatingReport || (detailStatus.status !== 'completed' && detailStatus.status !== 'completed_with_warnings')}
                    className="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-700 disabled:cursor-not-allowed disabled:bg-slate-300"
                  >
                    {generatingReport ? t('admin.reports.generating') : t('admin.reports.generateDiagnosis')}
                  </button>
                  {reportMessage && <div className="w-full text-xs text-slate-600">{reportMessage}</div>}
                </div>
              </div>

              <div className="rounded-xl border border-blue-100 bg-blue-50 p-5 shadow-sm">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <div className="text-base font-semibold text-slate-900">审计底稿</div>
                    <div className="mt-1 text-sm text-slate-500">从 code_comment 调用中实时提取的原始底稿，保留审计时的原始状态</div>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <button
                      onClick={handleGenerateIssueList}
                      disabled={generatingReport || (detailStatus.status !== 'completed' && detailStatus.status !== 'completed_with_warnings')}
                      className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-slate-300"
                    >
                      {generatingReport ? t('admin.issues.generating') : t('admin.issues.generateBtn')}
                    </button>
                  </div>
                </div>
                {reportMessage && <div className="mt-2 text-xs text-slate-600">{reportMessage}</div>}
                {reviewIssues.length > 0 && (
                  <div className="mt-4 space-y-3">
                    <div className="text-sm text-slate-600">实提取 {reviewIssues.length} 条问题</div>
                    {reviewIssues.map((issue, index) => (
                      <div key={`${issue.path}-${index}`} className="rounded-xl border border-slate-200 bg-white p-4">
                        <div className="flex flex-wrap items-center gap-2 text-xs text-slate-500">
                          <span className="rounded-full bg-slate-100 px-2 py-1">#{index + 1}</span>
                          <span className="break-all font-medium text-slate-700">{issue.path}</span>
                        </div>
                        <div className="mt-3 text-sm font-semibold text-slate-900">问题</div>
                        <div className="mt-1 whitespace-pre-wrap text-sm leading-6 text-slate-700">{issue.content}</div>
                        {issue.suggestionCode && (
                          <>
                            <div className="mt-3 text-sm font-semibold text-slate-900">改进建议</div>
                            <pre className="mt-1 max-h-64 overflow-auto whitespace-pre-wrap rounded-lg border border-slate-100 bg-slate-50 p-3 text-xs leading-6 text-slate-700">{issue.suggestionCode}</pre>
                          </>
                        )}
                        {issue.existingCode && (
                          <>
                            <div className="mt-3 text-sm font-semibold text-slate-900">相关代码</div>
                            <pre className="mt-1 max-h-48 overflow-auto whitespace-pre-wrap rounded-lg border border-slate-100 bg-slate-50 p-3 text-xs leading-6 text-slate-700">{issue.existingCode}</pre>
                          </>
                        )}
                        {issue.thinking && (
                          <>
                            <div className="mt-3 text-sm font-semibold text-slate-900">分析依据</div>
                            <div className="mt-1 whitespace-pre-wrap text-sm leading-6 text-slate-600">{issue.thinking}</div>
                          </>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {sessionDetail.files && sessionDetail.files.length > 0 && (
                <div className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div>
                      <div className="text-base font-semibold text-slate-900">{t('admin.data.reviewFilesTitle')}</div>
                      <div className="mt-1 text-sm text-slate-500">已写入 {sessionDetail.files.length} 个文件的对话记录</div>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {sessionDetail.summary.status === 'running' && (
                        <div className="rounded-full border border-amber-200 bg-amber-50 px-3 py-1 text-xs font-medium text-amber-600">实时刷新中</div>
                      )}
                      <button
                        onClick={() => setReviewFilesExpanded((prev) => !prev)}
                        className="rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:border-slate-300 hover:text-slate-900"
                      >
                        {reviewFilesExpanded ? '收起' : '展开'}
                      </button>
                    </div>
                  </div>
                  {reviewFilesExpanded && <div className="mt-4 space-y-4 text-sm text-slate-600">
                    {sessionDetail.files.map((file, index) => (
                      <div key={`${file.filePath}-${index}`} className="rounded-2xl border border-slate-200 bg-slate-50 p-5">
                        <div className="flex flex-wrap items-center justify-between gap-3">
                          <div className="font-medium text-slate-900 break-all">{file.filePath}</div>
                          <div className="text-xs text-slate-500">{Object.keys(file.tasks || {}).length} {t('admin.data.taskGroupsUnit')}</div>
                        </div>
                        <div className="mt-4 space-y-4">
                          {Object.entries(file.tasks || {}).map(([taskType, cards]) => (
                            <div key={taskType} className="rounded-2xl border border-slate-200 bg-white p-4">
                              <div className="text-xs font-semibold uppercase tracking-[0.2em] text-brand-300">{taskType}</div>
                              <div className="mt-3 space-y-3">
                                {cards.map((card) => (
                                  <div key={`${taskType}-${card.requestNo}-${card.durationMs}`} className="rounded-xl border border-slate-200 bg-slate-50 p-4">
                                    <div className="flex flex-wrap items-center gap-3 text-xs text-slate-500">
                                      <span>#{card.requestNo || 0}</span>
                                      <span>{card.model || '-'}</span>
                                      <span>{card.durationMs || 0}ms</span>
                                      <span>{(card.promptTokens || 0) + (card.completionTokens || 0)} tokens</span>
                                    </div>
                                    {card.error && <div className="mt-3 text-sm text-rose-500">{card.error}</div>}
                                    {card.responseContent && (
                                      <pre className="mt-3 max-h-64 overflow-auto whitespace-pre-wrap rounded-xl border border-slate-200 bg-white p-3 text-xs leading-6 text-slate-700">{card.responseContent}</pre>
                                    )}
                                    {card.toolCalls?.length > 0 && (
                                      <div className="mt-3 space-y-2">
                                        {card.toolCalls.map((toolCall, toolIndex) => (
                                          <div key={`${toolCall.name}-${toolIndex}`} className="rounded-xl border border-slate-200 bg-white p-3">
                                            <div className="flex flex-wrap items-center gap-2 text-xs text-slate-500">
                                              <span className="rounded-full border border-slate-300 px-2 py-1">{toolCall.name}{toolCall.ok ? '' : ' failed'}</span>
                                              {toolCall.durationMs > 0 && <span>{toolCall.durationMs}ms</span>}
                                            </div>
                                            {toolCall.arguments && (
                                              <pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap rounded-lg border border-slate-100 bg-slate-50 p-3 text-xs leading-6 text-slate-700">{formatToolArguments(toolCall.arguments)}</pre>
                                            )}
                                          </div>
                                        ))}
                                      </div>
                                    )}
                                  </div>
                                ))}
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>}
                </div>
              )}
            </>
          )}
        </div>
      );
    }

    if (activeMenu === 'reports') {
      const dateOptions = [...new Set(reports
        .filter((report) => report.sessionTime && (!reportRepoFilter || report.encodedRepo === reportRepoFilter))
        .map((report) => report.sessionTime.slice(0, 10))
      )];
      dateOptions.sort((a, b) => b.localeCompare(a));

      return (
        <div className="space-y-4">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="flex flex-wrap items-center gap-3">
              <label className="text-sm font-medium text-slate-700">{t('admin.audit.repo')}</label>
              <select
                value={reportRepoFilter}
                onChange={(e) => {
                  setReportRepoFilter(e.target.value);
                  setReportDateFilter('');
                }}
                className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900"
              >
                <option value="">{t('admin.audit.allRepos')}</option>
                {repos.map((r) => (
                  <option key={r.encodedPath} value={r.encodedPath}>{r.displayName || r.encodedPath}</option>
                ))}
              </select>

              <label className="text-sm font-medium text-slate-700">{t('admin.reports.sessionDate')}</label>
              <select
                value={reportDateFilter}
                onChange={(e) => setReportDateFilter(e.target.value)}
                className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900"
              >
                <option value="">{t('admin.audit.allDates')}</option>
                {dateOptions.map((date) => (
                  <option key={date} value={date}>{date}</option>
                ))}
              </select>

              <div className="ml-auto text-sm text-slate-500">{reports.length} {t('admin.reports.unit')}</div>
            </div>
          </div>

          <div className="space-y-3">
            {!reportsLoading && reports.map((report) => (
              <div key={report.id} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div className="min-w-0">
                    <div className="text-base font-semibold text-slate-900">{report.repoName || report.encodedRepo}</div>
                    <div className="mt-1 break-all text-xs text-slate-500">{report.sessionID}</div>
                    <div className="mt-2 flex flex-wrap gap-2 text-xs text-slate-500">
                      <span>{t('admin.reports.generatedAt')}: {report.generatedAt ? new Date(report.generatedAt).toLocaleString('zh-CN', { hour12: false }) : '-'}</span>
                      <span>{t('admin.reports.sessionTime')}: {report.sessionTime ? new Date(report.sessionTime).toLocaleString('zh-CN', { hour12: false }) : '-'}</span>
                      <span>{report.issueCount} {t('admin.reports.issuesUnit')}</span>
                      <span>{report.fileCount} {t('admin.audit.table.files')}</span>
                    </div>
                  </div>
                  <button
                    onClick={() => setExpandedReport((prev) => prev === report.id ? '' : report.id)}
                    className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 hover:border-slate-300 hover:text-slate-900"
                  >
                    {expandedReport === report.id ? t('admin.reports.collapse') : t('admin.reports.expand')}
                  </button>
                </div>
                {expandedReport === report.id && (
                  report.structured ? renderStructuredReport(report.structured) : (
                    <pre className="mt-4 max-h-[640px] overflow-auto whitespace-pre-wrap rounded-xl border border-slate-100 bg-slate-50 p-4 text-sm leading-7 text-slate-700">{report.content}</pre>
                  )
                )}
              </div>
            ))}
            {!reportsLoading && reports.length === 0 && (
              <div className="rounded-xl border border-slate-200 bg-white p-8 text-center text-sm text-slate-500">{t('admin.reports.empty')}</div>
            )}
            {reportsLoading && (
              <div className="rounded-xl border border-slate-200 bg-white p-8 text-center text-sm text-slate-500">{t('admin.data.loading')}</div>
            )}
          </div>
        </div>
      );
    }

    if (activeMenu === 'issueLists') {
      // 如果选中了具体清单，显示详情
      const selectedList = issueLists.find((l) => l.id === selectedIssueListId);
      if (selectedList) {
        // 收集所有文件和类别用于筛选
        const allFiles = [...new Set(selectedList.issues.map((i) => i.path))].sort();
        const allCategories = [...new Set(selectedList.issues.map((i) => i.category))].sort();
        const filteredIssues = selectedList.issues.filter((issue) => {
          if (issueListFileFilter && issue.path !== issueListFileFilter) return false;
          if (issueListSeverityFilter && issue.severity !== issueListSeverityFilter) return false;
          if (issueListCategoryFilter && issue.category !== issueListCategoryFilter) return false;
          return true;
        });

        // 统计
        const severityCounts: Record<string, number> = {};
        const categoryCounts: Record<string, number> = {};
        selectedList.issues.forEach((issue) => {
          severityCounts[issue.severity] = (severityCounts[issue.severity] || 0) + 1;
          categoryCounts[issue.category] = (categoryCounts[issue.category] || 0) + 1;
        });

        const severityBadgeClass: Record<string, string> = {
          critical: 'bg-red-100 text-red-700 border-red-200',
          high: 'bg-orange-100 text-orange-700 border-orange-200',
          medium: 'bg-amber-100 text-amber-700 border-amber-200',
          low: 'bg-emerald-100 text-emerald-700 border-emerald-200',
        };

        return (
          <div className="space-y-4">
            {/* 返回按钮 + 头部 */}
            <div className="flex flex-wrap items-center gap-3">
              <button
                onClick={() => setSelectedIssueListId('')}
                className="rounded-lg border border-slate-200 px-3 py-1.5 text-sm text-slate-700 hover:border-slate-300"
              >
                {t('admin.issues.backToList')}
              </button>
              <div className="text-lg font-semibold text-slate-900">{selectedList.repoName}</div>
              <span className="text-xs text-slate-500">{selectedList.sessionID}</span>
              <button
                onClick={() => {
                  setSelectedIssueListId('');
                  setActiveMenu('sessionDetail');
                  const params = new URLSearchParams();
                  params.set('repo', selectedList.encodedRepo);
                  params.set('session', selectedList.sessionID);
                  navigate(`/admin/task-detail?${params.toString()}`, { replace: true });
                }}
                className="ml-auto rounded-lg bg-blue-50 border border-blue-200 px-3 py-1.5 text-sm text-blue-700 hover:border-blue-300"
              >
                {t('admin.issues.seeInSession')}
              </button>
            </div>

            {/* 统计卡片 */}
            <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
              <div className="rounded-xl border border-slate-200 bg-white p-4">
                <div className="text-2xl font-bold text-slate-900">{selectedList.issues.length}</div>
                <div className="text-xs text-slate-500">{t('admin.issues.totalIssues')}</div>
              </div>
              <div className="rounded-xl border border-slate-200 bg-white p-4">
                <div className="text-2xl font-bold text-slate-900">{selectedList.fileCount}</div>
                <div className="text-xs text-slate-500">{t('admin.issues.totalFiles')}</div>
              </div>
              <div className="rounded-xl border border-slate-200 bg-white p-4">
                <div className="text-2xl font-bold text-red-600">{severityCounts.critical || 0}</div>
                <div className="text-xs text-slate-500">Critical</div>
              </div>
              <div className="rounded-xl border border-slate-200 bg-white p-4">
                <div className="text-2xl font-bold text-orange-600">{severityCounts.high || 0}</div>
                <div className="text-xs text-slate-500">High</div>
              </div>
            </div>

            {/* 筛选器 */}
            <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
              <div className="flex flex-wrap items-center gap-3">
                <label className="text-sm font-medium text-slate-700">{t('admin.issues.filterFile')}</label>
                <select
                  value={issueListFileFilter}
                  onChange={(e) => setIssueListFileFilter(e.target.value)}
                  className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900"
                >
                  <option value="">{t('admin.issues.allFiles')}</option>
                  {allFiles.map((f) => (
                    <option key={f} value={f}>{f}</option>
                  ))}
                </select>

                <label className="text-sm font-medium text-slate-700">{t('admin.issues.filterSeverity')}</label>
                <select
                  value={issueListSeverityFilter}
                  onChange={(e) => setIssueListSeverityFilter(e.target.value)}
                  className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900"
                >
                  <option value="">{t('admin.issues.allSeverities')}</option>
                  <option value="critical">{t('admin.issues.critical')}</option>
                  <option value="high">{t('admin.issues.high')}</option>
                  <option value="medium">{t('admin.issues.medium')}</option>
                  <option value="low">{t('admin.issues.low')}</option>
                </select>

                <label className="text-sm font-medium text-slate-700">{t('admin.issues.filterCategory')}</label>
                <select
                  value={issueListCategoryFilter}
                  onChange={(e) => setIssueListCategoryFilter(e.target.value)}
                  className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900"
                >
                  <option value="">{t('admin.issues.allCategories')}</option>
                  {allCategories.map((c) => (
                    <option key={c} value={c}>{c}</option>
                  ))}
                </select>

                <div className="ml-auto text-sm text-slate-500">
                  {filteredIssues.length}/{selectedList.issues.length} {t('admin.issues.totalIssues')}
                </div>
              </div>
            </div>

            {/* 类别分布 */}
            {Object.keys(categoryCounts).length > 0 && (
              <div className="rounded-xl border border-slate-200 bg-white p-4">
                <div className="text-sm font-semibold text-slate-900 mb-2">类别分布</div>
                <div className="flex flex-wrap gap-2">
                  {Object.entries(categoryCounts).sort((a, b) => b[1] - a[1]).map(([cat, count]) => (
                    <span key={cat} className="rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-700">
                      {cat} ({count})
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* 问题列表 */}
            <div className="space-y-3">
              {filteredIssues.map((issue, index) => (
                <div key={`${issue.path}-${index}`} className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="rounded-full bg-slate-100 px-2 py-1 text-xs font-medium text-slate-600">
                      {t('admin.issues.issueNo')}{index + 1}
                    </span>
                    <span className={`rounded-full border px-2 py-0.5 text-xs font-semibold ${severityBadgeClass[issue.severity] || 'bg-slate-100 text-slate-600 border-slate-200'}`}>
                      {issue.severity}
                    </span>
                    <span className="rounded-full bg-blue-50 px-2 py-0.5 text-xs text-blue-700">{issue.category}</span>
                    <span className="break-all text-xs text-slate-500">{issue.path}</span>
                    {issue.lineStart && issue.lineStart > 0 && (
                      <span className="text-xs text-slate-400">L{issue.lineStart}{issue.lineEnd ? `-L${issue.lineEnd}` : ''}</span>
                    )}
                  </div>
                  <div className="mt-3 text-sm text-slate-800">{issue.content}</div>
                  {issue.suggestion && (
                    <div className="mt-2">
                      <div className="text-xs font-semibold text-slate-900">{t('admin.issues.suggestion')}</div>
                      <pre className="mt-1 max-h-48 overflow-auto whitespace-pre-wrap rounded-lg border border-slate-100 bg-slate-50 p-3 text-xs leading-6 text-slate-700">{issue.suggestion}</pre>
                    </div>
                  )}
                </div>
              ))}
              {filteredIssues.length === 0 && (
                <div className="rounded-xl border border-slate-200 bg-white p-8 text-center text-sm text-slate-500">没有匹配筛选条件的问题。</div>
              )}
            </div>
          </div>
        );
      }

      // 清单列表页
      const repoOptions = [...new Set(issueLists.map((l) => l.encodedRepo))].sort();

      return (
        <div className="space-y-4">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="flex flex-wrap items-center gap-3">
              <label className="text-sm font-medium text-slate-700">{t('admin.issues.filterRepo')}</label>
              <select
                value={issueListRepoFilter}
                onChange={(e) => setIssueListRepoFilter(e.target.value)}
                className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900"
              >
                <option value="">{t('admin.issues.allRepos')}</option>
                {repoOptions.map((r) => (
                  <option key={r} value={r}>{r}</option>
                ))}
              </select>
              <div className="ml-auto text-sm text-slate-500">
                {issueLists.length} {t('admin.issues.totalIssues')}
              </div>
            </div>
          </div>

          <div className="space-y-3">
            {!issueListsLoading && issueLists.map((list) => {
              const severityStats: Record<string, number> = {};
              list.issues.forEach((issue) => {
                severityStats[issue.severity] = (severityStats[issue.severity] || 0) + 1;
              });
              return (
                <div
                  key={list.id}
                  className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm hover:border-slate-300 cursor-pointer transition-colors"
                  onClick={() => setSelectedIssueListId(list.id)}
                >
                  <div className="flex flex-wrap items-start justify-between gap-4">
                    <div className="min-w-0">
                      <div className="text-base font-semibold text-slate-900">{list.repoName || list.encodedRepo}</div>
                      <div className="mt-1 break-all text-xs text-slate-500">{list.sessionID}</div>
                      <div className="mt-2 flex flex-wrap gap-2 text-xs text-slate-500">
                        <span>{t('admin.issues.generatedAt')}: {list.generatedAt ? new Date(list.generatedAt).toLocaleString('zh-CN', { hour12: false }) : '-'}</span>
                        <span>{list.issues.length} {t('admin.issues.totalIssues')}</span>
                        <span>{list.fileCount} {t('admin.issues.totalFiles')}</span>
                      </div>
                      <div className="mt-2 flex flex-wrap gap-2 text-xs">
                        {(severityStats.critical || 0) > 0 && (
                          <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700">Critical: {severityStats.critical}</span>
                        )}
                        {(severityStats.high || 0) > 0 && (
                          <span className="rounded-full bg-orange-100 px-2 py-0.5 text-xs font-medium text-orange-700">High: {severityStats.high}</span>
                        )}
                        {(severityStats.medium || 0) > 0 && (
                          <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700">Medium: {severityStats.medium}</span>
                        )}
                        {(severityStats.low || 0) > 0 && (
                          <span className="rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700">Low: {severityStats.low}</span>
                        )}
                      </div>
                    </div>
                    <div className="rounded-lg bg-blue-50 border border-blue-200 px-4 py-2 text-sm font-medium text-blue-700">
                      {t('admin.issues.viewDetail')}
                    </div>
                  </div>
                </div>
              );
            })}
            {!issueListsLoading && issueLists.length === 0 && (
              <div className="rounded-xl border border-slate-200 bg-white p-8 text-center text-sm text-slate-500">
                {t('admin.issues.empty')}
                <div className="mt-2 text-xs text-slate-400">{t('admin.issues.emptyHint')}</div>
              </div>
            )}
            {issueListsLoading && (
              <div className="rounded-xl border border-slate-200 bg-white p-8 text-center text-sm text-slate-500">{t('admin.data.loading')}</div>
            )}
          </div>
        </div>
      );
    }

    if (activeMenu === 'sessions') {
      const repoFiltered = auditRepoFilter
        ? sessions.filter((s) => s.encodedRepo === auditRepoFilter)
        : sessions;

      const timeFiltered = auditDateFilter
        ? repoFiltered.filter((s) => s.timestamp === auditDateFilter)
        : repoFiltered;

      const dateOptions = [...new Set(sessions
        .filter((s) => s.timestamp && (!auditRepoFilter || s.encodedRepo === auditRepoFilter))
        .map((s) => s.timestamp)
      )];
      dateOptions.sort((a, b) => b.localeCompare(a));

      return (
        <div className="space-y-4">
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
            <div className="flex flex-wrap items-center gap-3">
              <label className="text-sm font-medium text-slate-700">{t('admin.audit.repo')}</label>
              <select
                value={auditRepoFilter}
                onChange={(e) => {
                  setAuditRepoFilter(e.target.value);
                  setAuditDateFilter('');
                }}
                className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900"
              >
                <option value="">{t('admin.audit.allRepos')}</option>
                {repos.map((r) => (
                  <option key={r.encodedPath} value={r.encodedPath}>{r.displayName || r.encodedPath}</option>
                ))}
              </select>

              <label className="text-sm font-medium text-slate-700">{t('admin.audit.dateFilter')}</label>
              <select
                value={auditDateFilter}
                onChange={(e) => setAuditDateFilter(e.target.value)}
                className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900"
              >
                <option value="">{t('admin.audit.allDates')}</option>
                {dateOptions.map((ts) => (
                  <option key={ts} value={ts}>{new Date(ts).toLocaleString('zh-CN', { hour12: false })}</option>
                ))}
              </select>

              <div className="ml-auto text-sm text-slate-500">{timeFiltered.length} {t('admin.data.sessionsUnit')}</div>
            </div>
          </div>

          <div className="rounded-xl border border-slate-200 bg-white shadow-sm overflow-x-auto">
            <table className="w-full text-left text-sm min-w-[1000px]">
              <thead className="border-b border-slate-200 bg-slate-50 text-xs uppercase text-slate-500">
                <tr>
                  <th className="px-4 py-3 whitespace-nowrap">{t('admin.audit.table.time')}</th>
                  <th className="px-4 py-3 whitespace-nowrap">{t('admin.audit.table.repoName')}</th>
                  <th className="px-4 py-3 whitespace-nowrap">{t('admin.audit.table.branch')}</th>
                  <th className="px-4 py-3 whitespace-nowrap">{t('admin.audit.table.mode')}</th>
                  <th className="px-4 py-3 whitespace-nowrap">{t('admin.audit.table.files')}</th>
                  <th className="px-4 py-3 whitespace-nowrap">耗时</th>
                  <th className="px-4 py-3 whitespace-nowrap">{t('admin.audit.table.status')}</th>
                  <th className="px-4 py-3 whitespace-nowrap">{t('admin.audit.table.actions')}</th>
                </tr>
              </thead>
              <tbody>
                {timeFiltered.map((session) => {
                  const statusView = getSessionStatusView(session.status, session.durationSec, session.llmFailures);
                  const durationText = session.durationSec ? `${session.durationSec.toFixed(1)}s` : '-';
                  const repoName = session.repoName || session.repoPath || session.cwd || '-';
                  const sessionRepo = session.encodedRepo || repos.find((item) => item.repoPath === session.cwd || item.repoPath === session.repoPath)?.encodedPath || '';
                  const deletingKey = `${sessionRepo}:${session.sessionID}`;
                  const deleting = deletingSessions.has(deletingKey);

                  return (
                    <tr
                      key={session.sessionID}
                      className="border-b border-slate-100 hover:bg-slate-50"
                    >
                      <td className="px-4 py-3 text-slate-900 whitespace-nowrap">{session.timestamp ? new Date(session.timestamp).toLocaleString('zh-CN', { hour12: false }) : '-'}</td>
                      <td className="px-4 py-3 font-medium text-slate-900 whitespace-nowrap">{repoName}</td>
                      <td className="px-4 py-3 text-slate-600 whitespace-nowrap">{session.gitBranch || '-'}</td>
                      <td className="px-4 py-3 text-slate-600 whitespace-nowrap">{session.reviewMode}</td>
                      <td className="px-4 py-3 text-slate-600">{session.fileCount}</td>
                      <td className="px-4 py-3 text-slate-600">{durationText}</td>
                      <td className="px-4 py-3 whitespace-nowrap">
                        <span className={`text-xs font-medium ${statusView.className}`}>{statusView.text}</span>
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap">
                        <div className="flex items-center gap-2">
                        <button
                          onClick={() => {
                            setSelectedRepo(sessionRepo);
                            setSelectedSession(session.sessionID);
                            navigate(`${menuPathMap.sessionDetail}?repo=${encodeURIComponent(sessionRepo)}&session=${encodeURIComponent(session.sessionID)}`, { replace: true });
                            setActiveMenu('sessionDetail');
                          }}
                          className="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-600 hover:border-brand-500/40 hover:text-brand-600"
                        >
                          明细
                        </button>
                        {statusView.status !== 'running' && (
                          <button
                            onClick={() => handleDeleteSession(session)}
                            disabled={deleting}
                            title="删除任务记录"
                            className="rounded-lg border border-red-200 px-2 py-1.5 text-xs font-medium text-red-500 hover:border-red-300 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50"
                          >
                            <i className={`fa-solid ${deleting ? 'fa-spinner fa-spin' : 'fa-trash'}`}></i>
                          </button>
                        )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
                {!sessionsLoading && timeFiltered.length === 0 && (
                  <tr>
                    <td colSpan={8} className="px-4 py-8 text-center text-sm text-slate-500">{t('admin.data.emptySessions')}</td>
                  </tr>
                )}
                {sessionsLoading && (
                  <tr>
                    <td colSpan={8} className="px-4 py-8 text-center text-sm text-slate-500">{t('admin.data.loading')}</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      );
    }

    if (activeMenu === 'rules') {
      const ruleEntryCount = (layer: RuleLayer) => (layer.rules?.length || 0) + (layer.pathRules?.length || 0) + (layer.defaultRule ? 1 : 0);

      return (
        <div className="grid gap-4">
          <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
            <div className="flex items-center justify-between gap-4">
              <div className="text-base font-semibold text-slate-900">{t('admin.rules.title')}</div>
              <div className="text-xs text-slate-500">{rulesLoading ? t('admin.data.loading') : `${ruleLayers.length}`}</div>
            </div>
            <div className="mt-4 space-y-4">
              {ruleLayers.map((layer) => (
                <div key={`${layer.source}-${layer.path}`} className="rounded-2xl border border-slate-200 bg-slate-50 p-5">
                  <div className="flex items-start justify-between gap-4">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="rounded-full bg-brand-100 px-3 py-1 text-xs font-semibold text-brand-700">
                          P{layer.priority}
                        </span>
                        <span className="text-sm font-semibold text-slate-900">{layer.title || layer.source}</span>
                        <span className={`rounded-full px-2.5 py-1 text-xs font-medium ${layer.available ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-200 text-slate-500'}`}>
                          {layer.available ? t('admin.rules.available') : t('admin.rules.notConfigured')}
                        </span>
                      </div>
                      {layer.description && <div className="mt-2 text-sm text-slate-600">{layer.description}</div>}
                      <div className="mt-2 text-xs break-all text-slate-500">{layer.path}</div>
                    </div>
                    <div className="text-xs text-slate-500">
                      {ruleEntryCount(layer)} {t('admin.rules.entriesUnit')}
                    </div>
                  </div>

                  <div className="mt-4 grid gap-3 text-xs text-slate-500 md:grid-cols-3">
                    <div className="rounded-xl border border-slate-200 bg-white p-3">
                      <div className="font-semibold text-slate-700">{t('admin.rules.source')}</div>
                      <div className="mt-1 break-all">{layer.source}</div>
                    </div>
                    <div className="rounded-xl border border-slate-200 bg-white p-3">
                      <div className="font-semibold text-slate-700">Include</div>
                      <div className="mt-1 break-all">{layer.include && layer.include.length > 0 ? layer.include.join(', ') : '-'}</div>
                    </div>
                    <div className="rounded-xl border border-slate-200 bg-white p-3">
                      <div className="font-semibold text-slate-700">Exclude</div>
                      <div className="mt-1 break-all">{layer.exclude && layer.exclude.length > 0 ? layer.exclude.join(', ') : '-'}</div>
                    </div>
                  </div>

                  {layer.defaultRule && (
                    <div className="mt-4 rounded-xl border border-slate-200 bg-white p-4 text-sm text-slate-700">
                      <div className="mb-2 text-xs uppercase tracking-[0.2em] text-brand-300">{t('admin.rules.defaultRule')}</div>
                      <pre className="whitespace-pre-wrap text-xs leading-6 text-slate-700">{layer.defaultRule}</pre>
                    </div>
                  )}

                  <div className="mt-4 space-y-3">
                    {layer.pathRules?.map((rule) => (
                      <div key={rule.pattern} className="rounded-xl border border-slate-200 bg-white p-4">
                        <div className="text-xs font-semibold text-brand-300">{rule.pattern}</div>
                        <pre className="mt-2 whitespace-pre-wrap text-xs leading-6 text-slate-700">{rule.rule}</pre>
                      </div>
                    ))}
                    {layer.rules?.map((rule) => (
                      <div key={rule.path} className="rounded-xl border border-slate-200 bg-white p-4">
                        <div className="text-xs font-semibold text-brand-300">{rule.path}</div>
                        <pre className="mt-2 whitespace-pre-wrap text-xs leading-6 text-slate-700">{rule.rule}</pre>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
              {!rulesLoading && ruleLayers.length === 0 && <div className="text-sm text-slate-500">{t('admin.rules.empty')}</div>}
            </div>
          </div>
        </div>
      );
    }

    if (activeMenu === 'settings') {
      return (
        <div className="grid gap-4 lg:grid-cols-[1.2fr_0.8fr]">
          <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
            <div className="flex items-center justify-between gap-4">
              <div className="text-base font-semibold text-slate-900">{t('admin.settings.llmTitle')}</div>
              <div className="text-xs text-slate-500">{llmConfigLoading ? t('admin.data.loading') : (llmResolvedVia || t('admin.settings.notConfigured'))}</div>
            </div>

            <div className="mt-4 space-y-4">
              <label className="block text-sm text-slate-700">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">{t('admin.settings.url')}</div>
                <input value={llmConfig.url} onChange={(event) => setLlmConfig((prev) => ({ ...prev, url: event.target.value }))} className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none" />
              </label>

              <label className="block text-sm text-slate-700">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">{t('admin.settings.authToken')}</div>
                <input type="password" value={llmConfig.authToken} onChange={(event) => setLlmConfig((prev) => ({ ...prev, authToken: event.target.value }))} className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none" />
              </label>

              <label className="block text-sm text-slate-700">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">{t('admin.settings.model')}</div>
                <input value={llmConfig.model} onChange={(event) => setLlmConfig((prev) => ({ ...prev, model: event.target.value }))} className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none" />
              </label>

              <label className="block text-sm text-slate-700">
                <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">{t('admin.settings.extraBody')}</div>
                <textarea value={llmConfig.extraBody} onChange={(event) => setLlmConfig((prev) => ({ ...prev, extraBody: event.target.value }))} rows={6} className="w-full rounded-xl border border-slate-300 bg-white px-4 py-3 text-sm text-slate-900 outline-none" />
              </label>

              <label className="flex items-center gap-3 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700">
                <input type="checkbox" checked={llmConfig.useAnthropic} onChange={(event) => setLlmConfig((prev) => ({ ...prev, useAnthropic: event.target.checked }))} />
                <span>{t('admin.settings.useAnthropic')}</span>
              </label>

              <button onClick={handleSaveLLMConfig} disabled={llmConfigSaving} className="rounded-xl bg-brand-500 px-4 py-3 text-sm font-semibold text-slate-950 transition-opacity disabled:cursor-not-allowed disabled:opacity-60">
                {llmConfigSaving ? t('admin.settings.saving') : t('admin.settings.save')}
              </button>

              {llmConfigMessage && <div className="text-sm text-slate-600">{llmConfigMessage}</div>}
            </div>
          </div>

          <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
            <div className="text-base font-semibold text-slate-900">{t('admin.settings.statusTitle')}</div>
            <div className="mt-4 space-y-3 text-sm text-slate-600">
              <div>{t('admin.settings.configPath')}: <span className="break-all text-slate-900">{llmConfigPath || '-'}</span></div>
              <div>{t('admin.settings.resolvedVia')}: <span className="text-slate-900">{llmResolvedVia || t('admin.settings.notConfigured')}</span></div>
              <div>{t('admin.settings.resolvedUrl')}: <span className="break-all text-slate-900">{llmResolvedUrl || '-'}</span></div>
              <div>{t('admin.settings.protocol')}: <span className="text-slate-900">{llmProtocol || '-'}</span></div>
            </div>

          </div>
        </div>
      );
    }

    return (
      <div className="mt-6 grid gap-4 lg:grid-cols-3">
        {currentSection.cards.map((card) => (
          <div key={card.title} className="rounded-2xl border border-slate-800 bg-slate-900/70 p-5">
            <div className="text-base font-semibold text-white">{card.title}</div>
            <p className="mt-3 text-sm leading-6 text-slate-400">{card.body}</p>
          </div>
        ))}
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-white text-slate-900">
      <div className="flex min-h-screen">
        <aside className="hidden md:flex w-52 shrink-0 border-r border-black bg-black">
          <div className="flex w-full flex-col px-3 py-4">
            <div className="mb-5">
              <div className="text-[11px] uppercase tracking-[0.26em] text-brand-400">CODE REVIEW</div>
              <h1 className="mt-2 text-xl font-semibold text-white">代码审计平台</h1>
            </div>

            <nav className="space-y-1.5">
              {menuItems.map((item) => {
                const selected = item.key === activeMenu || (activeMenu === 'sessionDetail' && item.key === 'sessions');
                const hidden = item.key === 'sessionDetail';
                if (hidden) return null;
                return (
                  <button
                    key={item.key}
                      onClick={() => handleMenuChange(item.key)}
                    className={`flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-left transition-all ${
                      selected
                        ? 'bg-brand-500/15 text-white border border-brand-500/30 shadow-lg shadow-brand-500/10'
                        : 'border border-transparent text-slate-400 hover:border-slate-700 hover:bg-slate-900/70 hover:text-white'
                    }`}
                  >
                    <i className={`fa-solid ${item.icon} w-4 text-center text-[13px]`}></i>
                    <span className="text-[13px] font-medium">{t(`admin.menu.${item.key}`)}</span>
                  </button>
                );
              })}
            </nav>

          </div>
        </aside>

        <main className="flex-1 bg-white">
          <header className="sticky top-0 z-20 border-b border-slate-200 bg-white/95 backdrop-blur-xl">
            <div className="flex flex-col gap-3 px-6 py-4 md:px-8 lg:px-10">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h2 className="text-[28px] font-semibold tracking-tight text-slate-900">{detailPageTitle}</h2>
                  {currentSection.description && <p className="mt-1.5 max-w-3xl text-sm leading-6 text-slate-600">{currentSection.description}</p>}
                </div>

                <div className="flex items-center gap-3">
                  <button
                    onClick={() => setLanguage(language === 'en' ? 'zh' : 'en')}
                    className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 transition-colors hover:border-brand-500/40 hover:text-slate-900"
                  >
                    {language === 'en' ? '中文' : 'EN'}
                  </button>
                  <a
                    href="#/docs"
                    className="rounded-lg border border-slate-300 bg-white px-3 py-1.5 text-sm font-medium text-slate-700 transition-colors hover:border-brand-500/40 hover:text-slate-900"
                  >
                    {t('admin.header.docs')}
                  </a>
                </div>
              </div>

              <div className="flex gap-2 overflow-x-auto md:hidden">
                {menuItems.map((item) => {
                  if (item.key === 'sessionDetail') return null;
                  const selected = item.key === activeMenu || (activeMenu === 'sessionDetail' && item.key === 'sessions');
                  return (
                    <button
                      key={item.key}
                    onClick={() => handleMenuChange(item.key)}
                      className={`whitespace-nowrap rounded-full px-4 py-2 text-sm transition-colors ${
                        selected ? 'bg-brand-500 text-slate-950' : 'bg-slate-100 text-slate-700'
                      }`}
                    >
                      {t(`admin.menu.${item.key}`)}
                    </button>
                  );
                })}
              </div>
            </div>
          </header>

          <div className="px-6 py-6 md:px-8 lg:px-10">
            {showStatCards && (
              <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
                {statCards.map((card) => (
                  <div key={card.title} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
                    <div className="text-sm text-slate-400">{card.title}</div>
                    <div className="mt-3 text-3xl font-semibold text-slate-900">{card.value}</div>
                    <div className="mt-2 text-sm text-slate-500">{card.detail}</div>
                  </div>
                ))}
              </section>
            )}

            {activeMenu !== 'overview' && (
              <section className="mt-6">
                {renderDataPanel()}
              </section>
            )}
          </div>
        </main>
      </div>
    </div>
  );
};

export default AdminConsolePage;
