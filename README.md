# 📦 SimpleArchiver

一个美观、功能丰富的终端用户界面（TUI）文件压缩/解压工具，使用 Go 语言和 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 框架构建。

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/License-MIT-green)

## ✨ 特性

- 🎨 **美观的 TUI 界面** - 使用 Bubble Tea 和 Lip Gloss 构建的现代终端界面
- 📁 **交互式文件选择器** - 轻松浏览和选择文件/文件夹
- 🗜️ **压缩功能** - 支持多种压缩格式
- 📂 **解压功能** - 支持多种归档格式的解压
- 🔄 **模式切换** - 启动时选择压缩或解压模式
- 🔐 **密码保护** - ZIP格式支持AES-256加密
- 🗜️ **多种压缩格式支持**
  - ZIP（通用格式，兼容性最好）
  - TAR.GZ（Linux 常用格式）
  - TAR.BZ2（高压缩率）
  - TAR.XZ（最高压缩率）
  - TAR.ZST（Zstandard，速度与压缩率平衡）
  - TAR.LZ4（最快速度）
- 🚫 **智能排除规则** - 预设常见开发目录排除模式
  - Python: `venv`, `__pycache__`, `.pyc` 等
  - Node.js: `node_modules`, `.npm` 等
  - IDE: `.idea`, `.vscode` 等
  - Git: `.git`
  - 构建产物: `dist`, `build`, `target` 等
- 📊 **实时进度显示** - 动画进度条和当前文件显示
- 📈 **压缩统计** - 显示压缩率、文件数量、大小等信息
- ⌨️ **Vim 风格快捷键** - `j/k` 导航，`h/l` 进入/返回

## 🚀 安装

### 从源码编译

```bash
# 克隆仓库
git clone https://github.com/Lynricsy/SimpleArchiver.git
cd SimpleArchiver

# 编译
go build -o simple-archiver .

# 运行
./simple-archiver
```

### 使用 Go Install

```bash
go install github.com/Lynricsy/SimpleArchiver@latest
```

## 📖 使用方法

直接运行程序即可启动交互式界面：

```bash
./simple-archiver
```

### 操作流程

#### 压缩模式
1. **选择压缩模式** - 启动后选择"压缩文件/文件夹"
2. **选择文件/文件夹** - 使用方向键或 `j/k` 浏览，`Space` 选择
3. **选择压缩格式** - 选择需要的压缩格式（ZIP, TAR.GZ 等）
4. **配置排除规则** - 选择要排除的文件类型（可自定义）
5. **确认并压缩** - 确认设置后开始压缩

#### 解压模式
1. **选择解压模式** - 启动后选择"解压归档文件"
2. **选择归档文件** - 浏览并选择要解压的压缩包（📦 图标标识）
3. **确认并解压** - 确认后开始解压到同名目录

#### 支持的归档格式
| 格式 | 压缩 | 解压 | 密码支持 |
|------|------|------|----------|
| ZIP | ✅ | ✅ | ✅ AES-256 |
| 7z | ❌ | ✅ | ✅ |
| TAR.GZ | ✅ | ✅ | ❌ |
| TAR.BZ2 | ✅ | ✅ | ❌ |
| TAR.XZ | ✅ | ✅ | ❌ |
| TAR.ZST | ✅ | ✅ | ❌ |
| TAR.LZ4 | ✅ | ✅ | ❌ |

### 快捷键

| 按键 | 功能 |
|------|------|
| `↑` / `k` | 上移光标 |
| `↓` / `j` | 下移光标 |
| `Enter` / `l` | 进入目录 |
| `Backspace` / `h` | 返回上级目录 |
| `Space` | 选择/切换 |
| `a` | 全选排除规则 |
| `n` | 取消全选 |
| `y` / `Enter` | 确认 |
| `Esc` / `q` | 返回/退出 |
| `Ctrl+C` | 强制退出 |

## 🎯 示例

### 压缩项目目录（排除依赖）

```
选择: my-project/
格式: TAR.GZ
排除: ☑ Node.js 相关 (node_modules)
      ☑ Python 相关 (venv, __pycache__)
      ☑ Git 版本控制 (.git)
输出: my-project.tar.gz
```

### 完整备份

```
选择: important-files/
格式: ZIP
排除: (全部取消)
输出: important-files.zip
```

## 🏗️ 技术栈

- **语言**: Go 1.25+
- **TUI 框架**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **样式**: [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- **组件**: [Bubbles](https://github.com/charmbracelet/bubbles) (进度条, Spinner)
- **压缩库**:
  - `pgzip` - 并行 Gzip
  - `dsnet/compress/bzip2` - Bzip2
  - `ulikunitz/xz` - XZ
  - `klauspost/compress/zstd` - Zstandard
  - `pierrec/lz4` - LZ4

## 📁 项目结构

```
SimpleArchiver/
├── main.go                    # 主程序入口
├── internal/
│   ├── archiver/             # 压缩核心逻辑
│   │   └── archiver.go
│   ├── config/               # 配置和排除规则
│   │   └── config.go
│   └── ui/                   # TUI 组件
│       ├── app.go
│       ├── filepicker.go
│       └── styles.go
├── go.mod
├── go.sum
└── README.md
```

## 📝 开发计划

- [x] ~~解压缩功能~~ ✅ 已完成
- [x] ~~密码保护压缩~~ ✅ 已完成 (ZIP AES-256)
- [x] ~~7z格式支持~~ ✅ 已完成 (解压)
- [ ] 命令行参数支持（非交互模式）
- [ ] 分卷压缩
- [ ] 压缩预览
- [ ] 配置文件支持

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

- [Charm](https://charm.sh/) - 提供优秀的 TUI 工具链
- 原始 Shell 脚本 `ZipArchive.sh` 的灵感

---

Made with ❤️ by [Lynricsy](https://github.com/Lynricsy)
