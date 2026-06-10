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

本分支针对企业使用场景增强了以下功能：
- **Web 管理控制台** - 仓库管理、任务清单、问题跟踪、审计报告
- **全量仓库审计模式** - 完整代码库扫描，智能过滤非源码文件
- **多 LLM Fallback** - 配置主用和备用 LLM 提供商
- **问题清单与审计报告** - 结构化缺陷跟踪和综合诊断

**安装：请始终使用本仓库的 `main` 分支。NPM 包和上游 Release 不包含这些增强功能。**

## 安装与使用

请参阅英文版 [README.md](README.md) 获取完整的安装和使用说明。

主要内容：
- **生产模式**：`./build.sh` 一键构建，`ocr serve --addr :3030` 启动
- **开发模式**：分离启动前端和后端，支持热更新
- **配置模型**：`ocr config set` 或环境变量
- **开始审查**：`ocr review` / `ocr review --full`

## 贡献

参见 CONTRIBUTING.md 了解开发环境搭建和编码规范。

## 许可证

Apache-2.0 — Copyright 2026 TheoEquity. Derived from [Open Code Review](https://github.com/alibaba/open-code-review) (Copyright 2026 Alibaba).
