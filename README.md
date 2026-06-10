<p align="center">
  <img src="imgs/logo.svg" alt="OpenCodeReview logo" width="240" height="240">
</p>
<p align="center">The open source AI code review agent.</p>
<p align="center">
  <a href="https://github.com/TheoEquity/Code-Review/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/github/license/TheoEquity/Code-Review?style=flat-square" /></a>
</p>
<p align="center">
  English | <a href="README.zh-CN.md">简体中文</a> | <a href="README.ja-JP.md">日本語</a>
</p>

---

## TheoEquity Code Review

This is an independent maintenance fork of the [Open Code Review](https://github.com/alibaba/open-code-review) project (Apache 2.0 License).

This fork adds enhanced features for enterprise use cases:
- **Web Management Console** - Repository management, task lists, issue tracking, audit reports
- **Full Repository Audit Mode** - Complete codebase scanning with smart non-source filtering
- **Multi-LLM Fallback** - Configure primary and fallback LLM providers
- **Issue List & Audit Reports** - Structured defect tracking and comprehensive diagnosis

**Installation: Always use this repository's `main` branch. The NPM package and upstream releases do not include these enhancements.**

### 自开发功能

- **Web 管理台**：提供仓库管理、新建任务、任务清单、任务详情、规则管理和模型配置页面。
- **本地仓库优先**：仓库以本地路径为主，远程地址用于同步；支持保存仓库名称、本地地址、远程地址、文件数、Token、任务数和同步操作。
- **全量审计模式**：新增 `--full` 模式，可对整个仓库进行审计，并默认过滤明显非源码文件，降低 Token 消耗。
- **任务状态修正**：运行中、成功、失败、部分完成按会话 JSONL 和进程状态推导，避免成功失败混淆。
- **问题清单**：从 `code_comment` 工具调用中提取问题，按当前仓库和当前会话隔离展示，支持去重。
- **审计报告**：对已完成任务生成结构化综合诊断，支持按仓库和会话时间查询，展示风险等级、问题类型、重点模块、Top 修复项和修复路线。
- **任务详情增强**：展示每个文件的 LLM 请求、响应、工具调用、耗时、Token 和失败信息。
- **规则管理展示**：展示当前系统的 4 层规则优先级：`--rule`、项目规则、全局规则、系统内置规则；当前默认启用系统内置 `13` 条路径规则和 `1` 条默认规则。

### 从本仓库安装

#### 方式一：生产模式（推荐，适用于部署到其他服务器）

生产模式将前端静态资源嵌入到二进制文件中，单个二进制文件即可启动 Web 控制台。

```bash
git clone https://github.com/TheoEquity/Code-Review.git
cd Code-Review
git checkout main

# 一键构建（自动编译前端 + 后端）
./build.sh
```

安装到系统命令路径：

```bash
cp ./dist/opencodereview /usr/local/bin/ocr
```

启动生产模式 Web 控制台：

```bash
ocr serve --addr :3030
```

打开浏览器访问 `http://<服务器IP>:3030`。

#### 方式二：开发模式（适用于当前开发服务器）

开发模式下前端使用 webpack dev server，支持热更新。后端用 `serve` 命令启动（不同端口）。

```bash
git clone https://github.com/TheoEquity/Code-Review.git
cd Code-Review
git checkout main

# 步骤 1: 启动后端（端口 5483）
./build.sh
./dist/opencodereview serve --addr :5483 &

# 步骤 2: 新开终端，启动前端（端口 3030）
cd pages
npm run dev
```

前端会自动 proxy `/api/*` 请求到后端 5483。

> **注意**：生产环境请使用方式一（单端口生产模式）。

### 配置模型

```bash
ocr config set llm.url https://your-llm-endpoint
ocr config set llm.auth_token your-api-key
ocr config set llm.model your-model-name
ocr config set llm.use_anthropic true
```

也可以使用环境变量：

```bash
export OCR_LLM_URL=https://your-llm-endpoint
export OCR_LLM_TOKEN=your-api-key
export OCR_LLM_MODEL=your-model-name
export OCR_USE_ANTHROPIC=true
```

测试模型连通性：

```bash
ocr llm test
```

### 启动 Web 管理台

#### 生产模式

```bash
ocr serve --addr :3030
```

#### 开发模式

当前端用 webpack dev server 运行时，前端端口 3030 会自动代理 API 请求到后端 5483。

如果通过域名或反向代理访问，需要配置允许的 Host：

```bash
OCR_VIEWER_ALLOWED_HOSTS=your-domain.example.com ocr viewer --addr 127.0.0.1:5483
```

### CLI 使用方式

在目标项目目录下执行：

```bash
# 审查当前 Git 工作区变更
ocr review

# 全量审计当前仓库
ocr review --full

# 审查两个分支或引用之间的差异
ocr review --from main --to feature-branch

# 审查指定提交
ocr review --commit abc123
```

### Web 管理台使用流程

1. 启动 `ocr serve`。
2. 打开 Web 管理台。
3. 在“仓库管理”中添加仓库名称和本地地址，可选填写远程地址。
4. 在“新建任务”中选择仓库和任务模式，默认推荐使用“全量审计”。
5. 在“任务清单”查看运行中和已完成任务。
6. 在“任务详情”中展开问题清单、生成综合诊断、查看已审查文件和每个文件的工具调用细节。
7. 在“审计报告”中按仓库和会话时间查询结构化综合诊断报告。
8. 在“规则管理”中查看当前生效规则层级和系统内置规则。

### 规则层级

规则按以下优先级解析，每个文件最终命中 1 条规则后交给 LLM 判断：

| Priority | Source | Path | Status |
|----------|--------|------|--------|
| 1 | Custom rule | `--rule <path>` | Only enabled when passed in a review task |
| 2 | Project rule | `<repoDir>/.opencodereview/rule.json` | Enabled when the repository contains this file |
| 3 | Global rule | `~/.opencodereview/rule.json` | Enabled when this file exists |
| 4 | System default | Embedded `system_rules.json` | Always enabled as fallback |

当前默认状态下，自开发系统主要使用第 4 层系统内置规则。

---

## What is Open Code Review?

Open Code Review is an AI-powered code review CLI tool. It originated as Alibaba Group's internal official AI code review assistant — over the past two years, it has served tens of thousands of developers and identified millions of code defects. After thorough validation at massive scale, we incubated it into an open source project for the community. Simply configure a model endpoint to get started.

It reads Git diffs, sends changed files to a configurable LLM via an agent with tool-use capabilities, and generates structured review comments with line-level precision. The agent can read full file contents, search the codebase, inspect other changed files for context, and produce deep reviews — not just surface-level diff feedback.

![Highlights](imgs/highlights-en.png)

## Why Open Code Review?

### The Problem with General-Purpose Agents

If you've used general-purpose agents like Claude Code with Skills for code review, you've likely encountered these pain points:

- **Incomplete coverage** — On larger changesets, agents tend to "cut corners," selectively reviewing only some files and missing others.
- **Position drift** — Reported issues frequently don't match the actual code location, with line numbers or file references drifting off target.
- **Unstable quality** — Natural-language-driven Skills are hard to debug, and review quality fluctuates significantly with minor prompt variations.

The root cause: a purely language-driven architecture lacks hard constraints on the review process.

### Core Design: Deterministic Engineering × Agent Hybrid

Open Code Review's core philosophy is to combine deterministic engineering with an agent, each handling what it does best.

**Deterministic Engineering — Hard Constraints**

For review steps that *must not go wrong*, engineering logic — not the language model — guarantees correctness:

- **Precise file selection** — Determines exactly which files need review and which should be filtered, ensuring no important change is missed.
- **Smart file bundling** — Groups related files into a single review unit (e.g., `message_en.properties` and `message_zh.properties` are bundled together). Each bundle runs as a sub-agent with isolated context — a divide-and-conquer strategy that stays stable on very large changesets and naturally supports concurrent review.
- **Fine-grained rule matching** — Matches review rules to each file's characteristics, keeping the model's attention sharply focused and eliminating information noise at the source. Compared to purely language-driven rule guidance, template-engine-based rule matching is more stable and predictable.
- **External positioning and reflection modules** — Independent comment-positioning and comment-reflection modules systematically improve both the location accuracy and content accuracy of AI feedback.

**Agent — Dynamic Decision-Making**

The agent's strengths are concentrated where they matter most — dynamic decisions and dynamic context retrieval:

- **Scenario-tuned prompts** — Prompt templates deeply optimized for code review, improving effectiveness while reducing token consumption.
- **Scenario-tuned toolset** — Distilled from deep analysis of tool-call traces in large-scale production data — including call frequency distributions, per-tool repetition rates, and the impact of new tools on the overall call chain — resulting in a purpose-built toolset that is more stable and predictable for code review than a generic agent toolkit.

## How to Use

### CLI

#### Install

**Production Mode (Recommended for deployment)**

Production mode embeds frontend static assets into the binary — a single binary file can start the web console.

```bash
git clone https://github.com/TheoEquity/Code-Review.git
cd Code-Review
git checkout main

# One-line build (automatically compiles frontend + backend)
./build.sh
```

Install to system path:

```bash
cp ./dist/opencodereview /usr/local/bin/ocr
```

Start production mode web console:

```bash
ocr serve --addr :3030
```

Open browser at `http://<server-IP>:3030`.

**Development Mode (for local development)**

In development mode, frontend uses webpack dev server with hot reload. Backend runs on a separate port.

```bash
git clone https://github.com/TheoEquity/Code-Review.git
cd Code-Review
git checkout main

# Step 1: Start backend (port 5483)
./build.sh
./dist/opencodereview serve --addr :5483 &

# Step 2: Open new terminal, start frontend (port 3030)
cd pages
npm run dev
```

Frontend automatically proxies `/api/*` requests to backend port 5483.

> **Note**: Use production mode (`ocr serve`) for production environments.

#### Quick Start

**1. Configure LLM**

**You must configure an LLM before reviewing code.**

```bash
# Option A: Interactive config
ocr config set llm.url https://api.anthropic.com/v1/messages
ocr config set llm.auth_token your-api-key-here
ocr config set llm.model claude-opus-4-6
ocr config set llm.use_anthropic true

# Option B: Environment variables (highest priority)
export OCR_LLM_URL=https://api.anthropic.com/v1/messages
export OCR_LLM_TOKEN=your-api-key-here
export OCR_LLM_MODEL=claude-opus-4-6
export OCR_USE_ANTHROPIC=true
```

Config is stored in `~/.opencodereview/config.json`.

It is also compatible with Claude Code environment variables (`ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_MODEL`) and parses `~/.zshrc` / `~/.bashrc` for those exports.

> **Note for CC-Switch Users**: If you are using [CC-Switch](https://github.com/farion1231/cc-switch) with [routing service](https://www.ccswitch.io/en/docs?section=proxy&item=service) enabled, you can point `llm.url` to the CC-Switch proxy address without additional configuration:
> - For **Claude** provider: set `llm.url` to `http://127.0.0.1:15721`
> - For **CodeX** provider: set `llm.url` to `http://127.0.0.1:15721/v1`
> - Set `llm.model` according to your provider settings
> - `llm.auth_token` can be any value
> - `extra_body` settings still apply

**2. Test Connectivity**

```bash
ocr llm test
```

**3. Review**

```bash
cd your-project

# Workspace mode — review all staged, unstaged, and untracked changes
ocr review

# Branch range — compare two refs
ocr review --from main --to feature-branch

# Single commit
ocr review --commit abc123
```

### CLI

#### Quick Start

OCR can be integrated into CI/CD pipelines to automate code review on Merge Requests / Pull Requests.

The core command for CI integration:

```bash
ocr review \
  --from "origin/main" \
  --to "origin/feature-branch" \
  --format json
```

The `--format json` flag outputs machine-readable results suitable for parsing in CI scripts.

See the [`examples/`](./examples/) directory for integration examples:

- [`github_actions/`](./examples/github_actions/) — GitHub Actions integration example
- [`gitlab_ci/`](./examples/gitlab_ci/) — GitLab CI integration example

## Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `ocr review` | `ocr r` | Start a code review |
| `ocr rules check <file>` | — | Preview which review rule applies to a file path |
| `ocr config set <key> <value>` | — | Set configuration values |
| `ocr llm test` | — | Test LLM connectivity |
| `ocr serve` | — | Launch production Web console on `localhost:3030` |
| `ocr version` | — | Show version info |

### `ocr review` Flags

| Flag | Shorthand | Default | Description |
|------|-----------|---------|-------------|
| `--repo` | — | current dir | Git repository root |
| `--from` | — | — | Source ref (e.g., `main`) |
| `--to` | — | — | Target ref (e.g., `feature-branch`) |
| `--commit` | `-c` | — | Single commit to review |
| `--preview` | `-p` | `false` | Preview which files will be reviewed without running the LLM |
| `--format` | `-f` | `text` | Output format: `text` or `json` |
| `--concurrency` | — | `8` | Max concurrent file reviews |
| `--timeout` | — | `10` | Concurrent task timeout in minutes |
| `--audience` | — | `human` | `human` (show progress) or `agent` (summary only) |
| `--rule` | — | — | Path to custom JSON review rules |
| `--max-tools` | — | built-in | Max tool call rounds per file; only takes effect when greater than template default |
| `--max-git-procs` | — | built-in | Max concurrent git subprocesses |
| `--tools` | — | — | Path to custom JSON tools config |

## Examples

```bash
# Preview which files will be reviewed (no LLM calls)
ocr review --preview
ocr review -c abc123 -p

# Review workspace changes with default settings
ocr review

# Review branch diff with higher concurrency
ocr review --from main --to my-feature --concurrency 4

# Review a specific commit with verbose JSON output
ocr review --commit abc123 --format json --audience agent

# Use custom review rules
ocr review --rule /path/to/my-rules.json

# Preview which rule applies to a file
ocr rules check src/main/java/com/example/Foo.java
ocr rules check --rule custom.json src/main/resources/mapper/UserMapper.xml

# Start production web console
ocr serve
ocr serve --addr :3030
```

### Web Console Security

The web console serves session JSONL contents (LLLM request messages and responses) over HTTP. It enforces a Host-header allowlist on every request: loopback names (`localhost`, `127.0.0.0/8`, `::1`) and the concrete bind host are always allowed. Wildcard binds (`--addr :3030`, `--addr 0.0.0.0:3030`) and other non-loopback Hostnames must be added via the `OCR_VIEWER_ALLOWED_HOSTS` environment variable (comma-separated):

```bash
OCR_VIEWER_ALLOWED_HOSTS=review.internal,ocr.lan ocr serve --addr :3030
```

This blocks DNS-rebinding attacks against the local web console.

## Review Rules

OCR resolves review rules using a four-layer priority chain. Each layer uses first-match-wins: if a file path matches a pattern, that rule is used; otherwise it falls through to the next layer.

| Priority | Source | Path | Description |
|----------|--------|------|-------------|
| 1 (highest) | `--rule` flag | User-specified path | CLI explicit override |
| 2 | Project config | `<repoDir>/.opencodereview/rule.json` | Per-project rules, can be committed to git |
| 3 | Global config | `~/.opencodereview/rule.json` | User-wide personal preferences |
| 4 (lowest) | System default | Embedded `system_rules.json` | Built-in rules covering common languages and file types |

### Rule File Format

Layers 1–3 share the same JSON format:

```json
{
  "rules": [
    {
      "path": "force-api/**/*.java",
      "rule": "All new methods must validate required parameters for null values"
    },
    {
      "path": "**/*mapper*.xml",
      "rule": "Check SQL for injection risks, parameter errors, and missing closing tags"
    }
  ]
}
```

- `path` supports `**` recursive matching and `{java,kt}` brace expansion.
- Within each layer, rules are evaluated in declaration order — the first match wins.
- If a rule file does not exist, it is silently skipped.

## Configuration Reference

Config file: `~/.opencodereview/config.json`

| Key | Type | Example |
|-----|------|---------|
| `llm.url` | string | `https://api.openai.com/v1/chat/completions` |
| `llm.auth_token` | string | `sk-xxxxxxx` |
| `llm.model` | string | `claude-opus-4-6` |
| `llm.use_anthropic` | boolean | `true` \| `false` |
| `language` | string | `English` \| `Chinese` (default: Chinese) |
| `telemetry.enabled` | boolean | `true` \| `false` |
| `telemetry.exporter` | string | `console` \| `otlp` |
| `telemetry.otlp_endpoint` | string | OTLP collector address |
| `telemetry.content_logging` | boolean | Include prompts in telemetry |

Environment variables take precedence over the config file.

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `OCR_LLM_URL` | LLM API endpoint URL |
| `OCR_LLM_TOKEN` | API key / auth token |
| `OCR_LLM_MODEL` | Model name |
| `OCR_USE_ANTHROPIC` | `true` = Anthropic, `false` = OpenAI |


## Telemetry

OpenTelemetry integration for observability (spans, metrics). Disabled by default.

```bash
ocr config set telemetry.enabled true
ocr config set telemetry.exporter otlp
ocr config set telemetry.otlp_endpoint localhost:4317
```

Set `telemetry.content_logging` to include LLM prompts and responses in exported data.

## Contributing

See CONTRIBUTING.md for development setup and guidelines.

## License

Apache-2.0 — Copyright 2026 TheoEquity. Derived from [Open Code Review](https://github.com/alibaba/open-code-review) (Copyright 2026 Alibaba).
