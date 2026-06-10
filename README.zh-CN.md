<p align="center">
  <img src="imgs/logo.svg" alt="OpenCodeReview logo" width="240" height="240">
</p>
<p align="center">The open source AI code review agent.</p>
<p align="center">
  <a href="https://github.com/TheoEquity/Code-Review/blob/main/LICENSE"><img alt="License" src="https://img.shields.io/github/license/TheoEquity/Code-Review?style=flat-square" /></a>
</p>
<p align="center">
  <a href="README.md">English</a> | 简体中文 | <a href="README.ja-JP.md">日本語</a>
</p>

---

## TheoEquity Code Review

本项目是 [Open Code Review](https://github.com/alibaba/open-code-review) 的独立维护分支（Apache 2.0 License）。

本分支增强了企业级功能：
- **Web 管理控制台** - 仓库管理、任务清单、问题跟踪、审计报告
- **全量仓库审计模式** - 完整代码库扫描，智能过滤非源码文件
- **多 LLM Fallback** - 配置主用和备用 LLM 提供商
- **问题清单与审计报告** - 结构化缺陷跟踪和综合诊断

**安装：请始终使用本仓库的 `main` 分支。NPM 包和上游 Release 不包含这些增强功能。**

## 快速安装（生产模式）

推荐用于生产服务器部署。

### 方式一：一键安装（推荐）

```bash
curl -sSL https://raw.githubusercontent.com/TheoEquity/Code-Review/main/install.sh | bash
```

### 方式二：手动下载

从 [GitHub Releases](https://github.com/TheoEquity/Code-Review/releases) 下载：

```bash
# Linux x86_64
wget https://github.com/TheoEquity/Code-Review/releases/latest/download/opencodereview-linux-amd64 -O ocr
chmod +x ocr && sudo mv ocr /usr/local/bin/

# Linux ARM64
wget https://github.com/TheoEquity/Code-Review/releases/latest/download/opencodereview-linux-arm64 -O ocr
chmod +x ocr && sudo mv ocr /usr/local/bin/

# macOS Apple Silicon
wget https://github.com/TheoEquity/Code-Review/releases/latest/download/opencodereview-darwin-arm64 -O ocr
chmod +x ocr && sudo mv ocr /usr/local/bin/
```

### 启动 Web 控制台

```bash
ocr serve --addr :3030
```

打开浏览器访问 `http://localhost:3030`。

---

## 安装选项

### 生产模式（服务器）

**推荐：快速安装** - 见上方 [快速安装](#快速安装生产模式) 章节。

**备选：源码构建**

```bash
git clone https://github.com/TheoEquity/Code-Review.git
cd Code-Review
./build.sh
cp ./dist/opencodereview /usr/local/bin/ocr
ocr serve --addr :3030
```

### 开发模式（本地开发）

开发模式下前端使用 webpack dev server，支持热更新。后端独立端口运行。

```bash
git clone https://github.com/TheoEquity/Code-Review.git
cd Code-Review

# 终端 1：后端（端口 5483）
./build.sh
./dist/opencodereview serve --addr :5483 &

# 终端 2：前端（端口 3030）
cd pages
npm run dev
```

打开浏览器访问 `http://localhost:3030`。前端会自动 proxy `/api/*` 请求到后端 5483。

---

## 配置 LLM

**在审查代码之前必须先配置 LLM。**

方式 A：命令行配置
```bash
ocr config set llm.url https://your-llm-endpoint
ocr config set llm.auth_token your-api-key
ocr config set llm.model claude-opus-4-6
ocr config set llm.use_anthropic true
```

方式 B：环境变量（最高优先级）
```bash
export OCR_LLM_URL=https://your-llm-endpoint
export OCR_LLM_TOKEN=your-api-key
export OCR_LLM_MODEL=claude-opus-4-6
export OCR_USE_ANTHROPIC=true
```

配置存储在 `~/.opencodereview/config.json`。

兼容 Claude Code 环境变量（`ANTHROPIC_BASE_URL`、`ANTHROPIC_AUTH_TOKEN`、`ANTHROPIC_MODEL`），并解析 `~/.zshrc` / `~/.bashrc` 中的相关导出。

> **CC-Switch 用户提示**：如果使用 [CC-Switch](https://github.com/farion1231/cc-switch) 并开启了 [路由服务](https://www.ccswitch.io/zh/docs?section=proxy&item=service)，可以将 `llm.url` 配置为 CC-Switch 代理地址，无需额外配置：
> - **Claude** 供应商：设置 `llm.url` 为 `http://127.0.0.1:15721`
> - **CodeX** 供应商：设置 `llm.url` 为 `http://127.0.0.1:15721/v1`
> - `llm.model` 根据供应商设置配置
> - `llm.auth_token` 可以设置任意值
> - `extra_body` 设置依然生效

### 测试连通性

```bash
ocr llm test
```

### 开始审查

```bash
cd your-project

# 工作区模式 - 审查所有暂存、未暂存和未跟踪的变更
ocr review

# 分支范围 - 比较两个引用
ocr review --from main --to feature-branch

# 单个提交
ocr review --commit abc123
```

---

## 命令参考

| 命令 | 别名 | 描述 |
|------|------|------|
| `ocr review` | `ocr r` | 开始代码审查 |
| `ocr rules check <file>` | — | 预览哪个审查规则适用于某个文件路径 |
| `ocr config set <key> <value>` | — | 设置配置值 |
| `ocr llm test` | — | 测试 LLM 连通性 |
| `ocr serve` | — | 启动生产模式 Web 控制台，默认 `localhost:3030` |
| `ocr version` | — | 显示版本信息 |

### `ocr review` 参数

| 参数 | 缩写 | 默认值 | 描述 |
|------|------|--------|------|
| `--repo` | — | 当前目录 | Git 仓库根目录 |
| `--from` | — | — | 源引用（如 `main`） |
| `--to` | — | — | 目标引用（如 `feature-branch`） |
| `--commit` | `-c` | — | 审查单个提交 |
| `--preview` | `-p` | `false` | 预览将被审查的文件列表，不调用 LLM |
| `--format` | `-f` | `text` | 输出格式：`text` 或 `json` |
| `--concurrency` | — | `8` | 最大并发文件审查数 |
| `--timeout` | — | `10` | 并发任务超时时间（分钟） |
| `--audience` | — | `human` | `human`（显示进度）或 `agent`（仅摘要） |
| `--rule` | — | — | 自定义 JSON 审查规则路径 |
| `--max-tools` | — | 内置默认 | 每个文件的最大工具调用轮次 |
| `--max-git-procs` | — | 内置默认 | 最大并发 git 子进程数 |
| `--tools` | — | — | 自定义 JSON 工具配置路径 |

## 示例

```bash
# 预览将被审查的文件（不调用 LLM）
ocr review --preview
ocr review -c abc123 -p

# 使用默认设置审查工作区变更
ocr review

# 以更高并发审查分支差异
ocr review --from main --to my-feature --concurrency 4

# 审查特定提交并以 JSON 格式输出详细信息
ocr review --commit abc123 --format json --audience agent

# 使用自定义审查规则
ocr review --rule /path/to/my-rules.json

# 预览哪个规则适用于某个文件
ocr rules check src/main/java/com/example/Foo.java
ocr rules check --rule custom.json src/main/resources/mapper/UserMapper.xml

# 启动生产模式 Web 控制台
ocr serve
ocr serve --addr :3030
```

### Web 控制台安全

Web 控制台通过 HTTP 提供会话 JSONL 内容（LLM 请求消息和响应）。每个请求都强制执行 Host 头白名单：回环名称（`localhost`、`127.0.0.0/8`、`::1`）和具体绑定主机始终允许。通配符绑定（`--addr :3030`、`--addr 0.0.0.0:3030`）和其他非回环主机名必须通过 `OCR_VIEWER_ALLOWED_HOSTS` 环境变量（逗号分隔）明确添加：

```bash
OCR_VIEWER_ALLOWED_HOSTS=review.internal,ocr.lan ocr serve --addr :3030
```

这可防止针对本地 Web 控制台的 DNS 重绑定攻击。

## 评审规则

OCR 通过四层优先级链解析评审规则。每层采用首次匹配原则：如果文件路径匹配到某个模式，则使用该规则；否则穿透到下一层。

| 优先级 | 来源 | 路径 | 描述 |
|--------|------|------|------|
| 1（最高） | `--rule` 参数 | 用户指定路径 | CLI 显式覆盖 |
| 2 | 项目配置 | `<repoDir>/.opencodereview/rule.json` | 项目级规则，可提交到 git |
| 3 | 全局配置 | `~/.opencodereview/rule.json` | 用户级个人偏好 |
| 4（最低） | 系统默认 | 内嵌 `system_rules.json` | 内置规则覆盖常见语言和文件类型 |

### 规则文件格式

第 1–3 层使用相同的 JSON 格式：

```json
{
  "rules": [
    {
      "path": "force-api/**/*.java",
      "rule": "所有新方法必须对必填参数进行空值校验"
    },
    {
      "path": "**/*mapper*.xml",
      "rule": "检查 SQL 注入风险、参数错误和缺少闭合标签"
    }
  ]
}
```

- `path` 支持 `**` 递归匹配和 `{java,kt}` 大括号展开。
- 在每一层内，规则按声明顺序评估 —— 首次匹配生效。
- 如果规则文件不存在，将被静默跳过。

## 配置参考

配置文件：`~/.opencodereview/config.json`

| 键 | 类型 | 示例 |
|-----|------|------|
| `llm.url` | string | `https://api.openai.com/v1/chat/completions` |
| `llm.auth_token` | string | `sk-xxxxxxx` |
| `llm.model` | string | `claude-opus-4-6` |
| `llm.use_anthropic` | boolean | `true` \| `false` |
| `language` | string | `English` \| `Chinese`（默认：Chinese） |
| `telemetry.enabled` | boolean | `true` \| `false` |
| `telemetry.exporter` | string | `console` \| `otlp` |
| `telemetry.otlp_endpoint` | string | OTLP 采集器地址 |
| `telemetry.content_logging` | boolean | 在遥测数据中包含提示词 |

环境变量优先级高于配置文件。

### 环境变量

| 变量 | 用途 |
|------|------|
| `OCR_LLM_URL` | LLM API 端点 URL |
| `OCR_LLM_TOKEN` | API 密钥 / 认证令牌 |
| `OCR_LLM_MODEL` | 模型名称 |
| `OCR_USE_ANTHROPIC` | `true` = Anthropic，`false` = OpenAI |

## 遥测

OpenTelemetry 集成用于可观测性（spans、metrics）。默认关闭。

```bash
ocr config set telemetry.enabled true
ocr config set telemetry.exporter otlp
ocr config set telemetry.otlp_endpoint localhost:4317
```

设置 `telemetry.content_logging` 可在导出数据中包含 LLM 提示词和响应。

## 贡献

参见 CONTRIBUTING.md 了解开发环境搭建和编码规范。

## 许可证

Apache-2.0 — Copyright 2026 TheoEquity. Derived from [Open Code Review](https://github.com/alibaba/open-code-review) (Copyright 2026 Alibaba).
