# Godscan

<h4 align="center">你的下一台扫描器，又何必是扫描器</h4>

<p align="center">
  <img src="https://img.shields.io/badge/language-Go-blue.svg" alt="Language">
  <a href="https://goreportcard.com/report/github.com/godspeedcurry/godscan">
    <img src="https://goreportcard.com/badge/github.com/godspeedcurry/godscan" alt="Go Report Card">
  </a>
  <a href="https://opensource.org/licenses/MIT">
    <img src="https://img.shields.io/badge/license-MIT-red.svg" alt="License">
  </a>
  <a href="https://github.com/godspeedcurry/godscan/releases">
    <img src="https://img.shields.io/github/downloads/godspeedcurry/godscan/total" alt="Downloads">
  </a>
</p>

## 简介 (Introduction)

**Godscan** 是一款现代化、高并发的企业级资产发现与安全评估工具。专为红队作业与安全建设设计，集成了端口扫描、指纹识别、API 敏感信息提取、弱口令检测以及基于 LLM 的智能分析功能。

## 核心特性 (Features)

- **全方位资产发现**:
  - **深度爬虫 (Spider)**: 基于 DFS 算法，深度提取 HTML/JS 中的 API 接口、敏感凭证 (AK/SK)、CDN 节点及 SourceMap 文件。
  - **指纹识别**: 内置丰富的指纹库，支持 favicon hash (Murmur3/MD5)、关键词及 HTTP 头识别。
  - **首页快照**: 自动捕获并存储首页 HTML 与响应头，支持离线 grep 检索。

- **AI 智能分析 (LLM)**:
  - **多模型支持**: 原生支持 Google Gemini (2.5/3.0)，兼容 OpenAI 协议 (GPT-4, Claude, DeepSeek)。
  - **智能摘要**: 自动分析扫描报告，识别潜在风险并生成修复建议。
  - **Profile 管理**: 支持加密存储多套 API 配置，灵活切换环境。

- **高性能引擎**:
  - **端口扫描**: 基于 Golang 协程池，支持自定义 TCP/JDWP 探针，吞吐量大且稳定。
  - **弱口令爆破**: 支持基础字典与动态规则变异（阴历、身份证、手机号组合），支持在线/离线模式。

- **专业报表**:
  - 生成交互式离线 HTML 报告，支持分页、搜索与分类筛选。
  - 标准化输出目录结构 (`output/YYYY-MM-DD/target/`)，便于 CI/CD 集成。

## 快速开始 (Quick Start)

### 1. 资产爬取与分析
```bash
# 针对单一目标进行爬取 (sp 为 spider 简写)
./godscan sp -u https://example.com

# 针对 URL 列表批量爬取
./godscan sp -f urls.txt
```

### 2. 生成智能报告
```bash
# 生成基础 HTML 报告
./godscan report

# 使用 Gemini/OpenAI 生成 AI 摘要 (推荐)
./godscan report --llm-key "sk-xxxx..." --llm-model "gemini-2.5-flash"

# 或者使用预设的 Profile
./godscan report --llm-profile local
```

### 3. 端口与服务扫描
```bash
# 扫描指定网段的 Top 端口
./godscan port -i "192.168.1.0/24" --top 1000

# 自定义端口范围
./godscan port -i "192.168.1.0/24" -p "80,443,8000-8080"
```

### 4. 弱口令生成
```bash
# 基于关键词生成个性化弱口令字典
./godscan weak -k "baidu,admin,123456" --full > pass.txt
```

### 5. 实用工具箱 (Utilities)
```bash
# 目录爆破 (DirBrute)
./godscan dir -u https://example.com -t 20

# 图标 Hash 计算 (Icon)
./godscan icon --url https://example.com/favicon.ico
```

## LLM 配置指南 (AI Integration)

Godscan 支持灵活的 LLM 配置方式，适应不同的网络环境与模型需求。

### 方式一：命令行参数 (CI/CD 友好)
直接在命令中指定 Key 和 BaseURL（适用于 OneAPI/DeepSeek 等 OpenAI 兼容接口）：
```bash
./godscan report \
  --llm-key "sk-your-key" \
  --llm-base-url "https://api.deepseek.com/v1" \
  --llm-model "deepseek-coder"
```

### 方式二：Profile 配置文件 (推荐)
通过交互式命令保存配置（支持加密存储）：
```bash
# 创建/更新 Profile
./godscan llm -i
# 按提示输入 Name (如: dev), Provider, Key 等信息

# 使用 Profile
./godscan report --llm-profile dev
```

## 编译与安装 (Installation)

### 源码编译
```bash
# 编译当前平台版本
go build -ldflags="-s -w" -trimpath -o godscan

# 交叉编译 (使用内置 Skill)
# 详情见 .agent/workflows/build_release.md
```

### 输出结构
所有扫描结果默认存储在 `output/` 目录下，按日期和目标自动归档：
```text
output/
├── 2026-01-13/
│   └── example.com_80/
│       └── spider/
│           └── spider.log
├── result.log          (端口扫描聚合结果)
└── report-2026-01-13.html
```

## 免责声明 (Disclaimer)

本工具仅面向合法授权的企业安全建设行为。在使用本工具进行检测时，您应确保该行为符合当地法律法规，并已取得足够的授权。如您在使用本工具的过程中存在任何非法行为，您需自行承担相应后果，开发者将不承担任何法律及连带责任。

