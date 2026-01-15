// Package config 提供压缩工具的配置管理
package config

// DefaultExcludes 默认排除模式列表
var DefaultExcludes = []string{
	// Python
	"venv", ".venv", "__pycache__", "*.pyc", "*.pyo",
	".pytest_cache", ".mypy_cache", "*.egg-info", ".eggs",

	// Node.js
	"node_modules", ".npm", ".pnpm-store",

	// IDE/Editor
	".idea", ".vscode", "*.swp", "*.swo", "*~",

	// Git
	".git",

	// 构建产物
	"dist", "build", "target", "out",

	// 系统文件
	".DS_Store", "Thumbs.db", "desktop.ini",

	// 日志和缓存
	"*.log", ".cache", ".temp", ".tmp",

	// Go
	"vendor",

	// Rust
	"target",

	// Java/Gradle
	".gradle", ".m2",
}

// ExcludeCategory 排除类别
type ExcludeCategory struct {
	Name     string
	Patterns []string
	Selected bool
}

// GetExcludeCategories 获取排除类别列表
func GetExcludeCategories() []ExcludeCategory {
	return []ExcludeCategory{
		{
			Name:     "Python 相关",
			Patterns: []string{"venv", ".venv", "__pycache__", "*.pyc", "*.pyo", ".pytest_cache", ".mypy_cache", "*.egg-info", ".eggs"},
			Selected: true,
		},
		{
			Name:     "Node.js 相关",
			Patterns: []string{"node_modules", ".npm", ".pnpm-store"},
			Selected: true,
		},
		{
			Name:     "IDE/编辑器配置",
			Patterns: []string{".idea", ".vscode", "*.swp", "*.swo", "*~"},
			Selected: true,
		},
		{
			Name:     "Git 版本控制",
			Patterns: []string{".git"},
			Selected: true,
		},
		{
			Name:     "构建产物",
			Patterns: []string{"dist", "build", "target", "out"},
			Selected: true,
		},
		{
			Name:     "系统文件",
			Patterns: []string{".DS_Store", "Thumbs.db", "desktop.ini"},
			Selected: true,
		},
		{
			Name:     "日志和缓存",
			Patterns: []string{"*.log", ".cache", ".temp", ".tmp"},
			Selected: true,
		},
		{
			Name:     "Go 依赖",
			Patterns: []string{"vendor"},
			Selected: true,
		},
		{
			Name:     "Java/Gradle 相关",
			Patterns: []string{".gradle", ".m2"},
			Selected: true,
		},
	}
}

// ArchiveFormat 压缩格式
type ArchiveFormat struct {
	Name        string
	Extension   string
	Description string
}

// GetArchiveFormats 获取支持的压缩格式列表
func GetArchiveFormats() []ArchiveFormat {
	return []ArchiveFormat{
		{Name: "ZIP", Extension: ".zip", Description: "通用压缩格式，兼容性最好"},
		{Name: "TAR.GZ", Extension: ".tar.gz", Description: "Linux 常用格式，压缩率中等"},
		{Name: "TAR.BZ2", Extension: ".tar.bz2", Description: "压缩率较高，速度较慢"},
		{Name: "TAR.XZ", Extension: ".tar.xz", Description: "压缩率最高，速度最慢"},
		{Name: "TAR.ZST", Extension: ".tar.zst", Description: "Zstandard 压缩，速度和压缩率平衡"},
		{Name: "TAR.LZ4", Extension: ".tar.lz4", Description: "LZ4 压缩，速度最快"},
	}
}
