// Package i18n 提供国际化支持
// 支持中文(zh)和英文(en)，默认英文
package i18n

import (
	"os"
	"strings"
)

// Language 语言类型
type Language string

const (
	LangEN Language = "en" // 英语
	LangZH Language = "zh" // 中文
)

// 当前语言
var currentLang Language = LangEN

// Messages 消息定义
type Messages struct {
	// 应用标题
	AppTitle string

	// 模式
	ModeCompress string
	ModeExtract  string

	// 状态栏提示
	HintUp       string
	HintDown     string
	HintEnter    string
	HintSelect   string
	HintBack     string
	HintQuit     string
	HintToggle   string
	HintSelectAll string
	HintClear    string
	HintConfirm  string
	HintCancel   string
	HintPassword string
	HintInput    string
	HintExit     string

	// 模式选择
	SelectModeTitle       string
	CompressOption        string
	CompressOptionDesc    string
	ExtractOption         string
	ExtractOptionDesc     string

	// 文件选择
	SelectFileCompress    string
	SelectFileExtract     string
	EmptyDir              string
	ShowRange             string

	// 格式选择
	SelectFormat          string
	SelectedFile          string

	// 排除规则
	SelectExcludes        string
	ExcludeFormat         string
	ToggleHint            string

	// 密码输入
	PasswordTitle         string
	PasswordExtract       string
	PasswordHint          string
	PasswordEmpty         string
	PasswordProtection    string
	NoPassword            string
	NoPasswordDesc        string
	SetPassword           string
	SetPasswordDesc       string
	InputPassword         string
	InputPasswordHint     string

	// 确认
	ConfirmCompress       string
	ConfirmExtract        string
	SourceFile            string
	OutputFile            string
	ExtractTo             string
	ExtractPassword       string
	PasswordSet           string
	PasswordNone          string
	CompressFormat        string
	PasswordProtect       string
	AESEncrypted          string
	ExcludeRules          string
	PatternsCount         string
	ConfirmStart          string
	ConfirmStartExtract   string

	// 压缩中/解压中
	Compressing           string
	Extracting            string
	Preparing             string
	Speed                 string
	Current               string
	Average               string
	Progress              string
	FilesProgress         string
	Excluded              string
	FilesAndDirs          string
	ElapsedTime           string

	// 完成
	CompressDone          string
	ExtractDone           string
	OutputFileLabel       string
	ExtractToLabel        string
	ExtractedFiles        string
	ExtractedSize         string
	CompressedFiles       string
	OriginalSize          string
	CompressedSize        string
	CompressionRate       string
	ExcludedFiles         string

	// 错误
	CompressFailed        string
	ExtractFailed         string
	ErrorMessage          string
}

// 英文消息
var messagesEN = Messages{
	AppTitle: "SimpleArchiver",

	ModeCompress: "Compress",
	ModeExtract:  "Extract",

	HintUp:        "Up",
	HintDown:      "Down",
	HintEnter:     "Enter",
	HintSelect:    "Select",
	HintBack:      "Back",
	HintQuit:      "Quit",
	HintToggle:    "Toggle",
	HintSelectAll: "All",
	HintClear:     "Clear",
	HintConfirm:   "Confirm",
	HintCancel:    "Cancel",
	HintPassword:  "Password",
	HintInput:     "Input",
	HintExit:      "Exit",

	SelectModeTitle:    "🎯 Select Operation Mode",
	CompressOption:     "Compress File/Folder",
	CompressOptionDesc: "Compress files or folders into an archive",
	ExtractOption:      "Extract Archive",
	ExtractOptionDesc:  "Extract archive to a directory",

	SelectFileCompress: "📂 Select File or Folder to Compress",
	SelectFileExtract:  "📂 Select Archive to Extract",
	EmptyDir:           "(empty directory)",
	ShowRange:          "Showing %d-%d / %d",

	SelectFormat: "📦 Select Compression Format",
	SelectedFile: "Selected: ",

	SelectExcludes: "🚫 Select Exclude Rules",
	ExcludeFormat:  "Format: ",
	ToggleHint:     " | Space to toggle",

	PasswordTitle:       "🔐 Password Protection",
	PasswordExtract:     "🔐 Enter Extraction Password",
	PasswordHint:        "If the archive is password protected, enter the password",
	PasswordEmpty:       "(empty=no password, press Enter to confirm)",
	PasswordProtection:  "ZIP supports AES-256 encryption",
	NoPassword:          "No Password",
	NoPasswordDesc:      "Create a normal ZIP file",
	SetPassword:         "Set Password",
	SetPasswordDesc:     "Use AES-256 encryption",
	InputPassword:       "Enter password:",
	InputPasswordHint:   "(enter password and press Enter)",

	ConfirmCompress:     "✅ Confirm Compression",
	ConfirmExtract:      "✅ Confirm Extraction",
	SourceFile:          "Source:",
	OutputFile:          "Output:",
	ExtractTo:           "Extract to:",
	ExtractPassword:     "Password:",
	PasswordSet:         "🔑 Set",
	PasswordNone:        "🔓 None",
	CompressFormat:      "Format:",
	PasswordProtect:     "Protection:",
	AESEncrypted:        "🔒 AES-256 Encrypted",
	ExcludeRules:        "Excludes:",
	PatternsCount:       "%d patterns",
	ConfirmStart:        "Press Y/Enter to start compression, N/Esc to go back",
	ConfirmStartExtract: "Press Y/Enter to start extraction, N/Esc to go back",

	Compressing:   "🚀 Compressing...",
	Extracting:    "📂 Extracting...",
	Preparing:     "Preparing...",
	Speed:         "Speed:",
	Current:       "Current:",
	Average:       "Average:",
	Progress:      "Progress:",
	FilesProgress: "%d / %d files",
	Excluded:      "Excluded:",
	FilesAndDirs:  "%d files/dirs",
	ElapsedTime:   "Elapsed:",

	CompressDone:    "🎉 Compression Complete!",
	ExtractDone:     "🎉 Extraction Complete!",
	OutputFileLabel: "Output:",
	ExtractToLabel:  "Extracted to:",
	ExtractedFiles:  "Files:",
	ExtractedSize:   "Size:",
	CompressedFiles: "Files:",
	OriginalSize:    "Original:",
	CompressedSize:  "Compressed:",
	CompressionRate: "Ratio:",
	ExcludedFiles:   "Excluded:",

	CompressFailed: "❌ Compression Failed",
	ExtractFailed:  "❌ Extraction Failed",
	ErrorMessage:   "Error:",
}

// 中文消息
var messagesZH = Messages{
	AppTitle: "SimpleArchiver",

	ModeCompress: "压缩",
	ModeExtract:  "解压",

	HintUp:        "上移",
	HintDown:      "下移",
	HintEnter:     "进入",
	HintSelect:    "选择",
	HintBack:      "返回",
	HintQuit:      "退出",
	HintToggle:    "切换",
	HintSelectAll: "全选",
	HintClear:     "清除",
	HintConfirm:   "确认",
	HintCancel:    "取消",
	HintPassword:  "密码",
	HintInput:     "输入",
	HintExit:      "退出",

	SelectModeTitle:    "🎯 选择操作模式",
	CompressOption:     "压缩文件/文件夹",
	CompressOptionDesc: "将文件或文件夹压缩为归档文件",
	ExtractOption:      "解压归档文件",
	ExtractOptionDesc:  "将压缩包解压到指定目录",

	SelectFileCompress: "📂 选择要压缩的文件或文件夹",
	SelectFileExtract:  "📂 选择要解压的归档文件",
	EmptyDir:           "(空目录)",
	ShowRange:          "显示 %d-%d / %d",

	SelectFormat: "📦 选择压缩格式",
	SelectedFile: "已选择: ",

	SelectExcludes: "🚫 选择排除规则",
	ExcludeFormat:  "格式: ",
	ToggleHint:     " | 空格切换选中状态",

	PasswordTitle:       "🔐 密码保护设置",
	PasswordExtract:     "🔐 输入解压密码",
	PasswordHint:        "如果归档文件有密码保护，请输入密码",
	PasswordEmpty:       "(留空=无密码，直接Enter确认)",
	PasswordProtection:  "ZIP格式支持 AES-256 加密保护",
	NoPassword:          "不使用密码",
	NoPasswordDesc:      "生成普通ZIP文件",
	SetPassword:         "设置密码",
	SetPasswordDesc:     "使用 AES-256 加密",
	InputPassword:       "输入密码:",
	InputPasswordHint:   "(输入密码后按Enter确认)",

	ConfirmCompress:     "✅ 确认压缩",
	ConfirmExtract:      "✅ 确认解压",
	SourceFile:          "源文件:",
	OutputFile:          "输出文件:",
	ExtractTo:           "解压到:",
	ExtractPassword:     "解压密码:",
	PasswordSet:         "🔑 已设置",
	PasswordNone:        "🔓 无",
	CompressFormat:      "压缩格式:",
	PasswordProtect:     "密码保护:",
	AESEncrypted:        "🔒 AES-256 加密",
	ExcludeRules:        "排除规则:",
	PatternsCount:       "%d 个模式",
	ConfirmStart:        "按 Y/Enter 开始压缩，N/Esc 返回修改",
	ConfirmStartExtract: "按 Y/Enter 开始解压，N/Esc 返回修改",

	Compressing:   "🚀 正在压缩...",
	Extracting:    "📂 正在解压...",
	Preparing:     "准备中...",
	Speed:         "速度:",
	Current:       "当前:",
	Average:       "平均:",
	Progress:      "处理进度:",
	FilesProgress: "%d / %d 文件",
	Excluded:      "已排除:",
	FilesAndDirs:  "%d 个文件/目录",
	ElapsedTime:   "已用时间:",

	CompressDone:    "🎉 压缩完成！",
	ExtractDone:     "🎉 解压完成！",
	OutputFileLabel: "输出文件:",
	ExtractToLabel:  "解压到:",
	ExtractedFiles:  "解压文件:",
	ExtractedSize:   "解压大小:",
	CompressedFiles: "压缩文件:",
	OriginalSize:    "原始大小:",
	CompressedSize:  "压缩后大小:",
	CompressionRate: "压缩率:",
	ExcludedFiles:   "排除文件:",

	CompressFailed: "❌ 压缩失败",
	ExtractFailed:  "❌ 解压失败",
	ErrorMessage:   "错误信息:",
}

// Init 初始化语言设置，根据系统locale自动检测
func Init() {
	// 检测系统语言
	lang := detectSystemLanguage()
	SetLanguage(lang)
}

// detectSystemLanguage 检测系统语言
func detectSystemLanguage() Language {
	// 按优先级检查环境变量
	envVars := []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"}

	for _, env := range envVars {
		if val := os.Getenv(env); val != "" {
			// 检查是否包含中文标识
			lowerVal := strings.ToLower(val)
			if strings.HasPrefix(lowerVal, "zh") ||
				strings.Contains(lowerVal, "chinese") ||
				strings.Contains(lowerVal, "china") {
				return LangZH
			}
		}
	}

	// 默认英语
	return LangEN
}

// SetLanguage 设置当前语言
func SetLanguage(lang Language) {
	currentLang = lang
}

// GetLanguage 获取当前语言
func GetLanguage() Language {
	return currentLang
}

// T 返回当前语言的消息
func T() *Messages {
	switch currentLang {
	case LangZH:
		return &messagesZH
	default:
		return &messagesEN
	}
}

// IsZH 检查当前是否为中文
func IsZH() bool {
	return currentLang == LangZH
}
