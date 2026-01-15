// SimpleArchiver - 智能终端文件压缩/解压工具
// 一个美观、功能丰富的 TUI 压缩器，支持多种压缩格式
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Lynricsy/SimpleArchiver/internal/archiver"
	"github.com/Lynricsy/SimpleArchiver/internal/config"
)

// 版本信息
const (
	AppName    = "SimpleArchiver"
	AppVersion = "1.4.0"
)

// 操作模式
type opMode int

const (
	modeCompress opMode = iota
	modeExtract
)

// 颜色定义
var (
	primaryColor    = lipgloss.Color("#7C3AED")
	secondaryColor  = lipgloss.Color("#06B6D4")
	successColor    = lipgloss.Color("#10B981")
	warningColor    = lipgloss.Color("#F59E0B")
	errorColor      = lipgloss.Color("#EF4444")
	mutedColor      = lipgloss.Color("#6B7280")
	foregroundColor = lipgloss.Color("#F9FAFB")
	borderColor     = lipgloss.Color("#374151")
	archiveColor    = lipgloss.Color("#EC4899") // 粉色用于压缩文件
)

// 样式定义
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(1, 2)

	highlightBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Foreground(foregroundColor).
			Background(primaryColor).
			Bold(true).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Foreground(foregroundColor).
			Padding(0, 1)

	disabledStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Strikethrough(true).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	infoStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	statLabelStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Width(16)

	statValueStyle = lipgloss.NewStyle().
			Foreground(foregroundColor).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	folderIconStyle = lipgloss.NewStyle().
			Foreground(warningColor)

	fileIconStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	archiveIconStyle = lipgloss.NewStyle().
			Foreground(archiveColor)

	// Zellij 风格状态栏样式
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(foregroundColor).
			Padding(0, 1)

	statusKeyStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true).
			Padding(0, 1)

	statusDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")).
			Padding(0, 1)

	statusSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4B5563"))
)

// AppState 应用状态
type appState int

const (
	stateSelectMode appState = iota
	stateSelectFile
	stateSelectFormat
	stateSelectExcludes
	stateInputPassword
	stateConfirm
	stateCompressing
	stateExtracting
	stateDone
	stateError
)

// FileEntry 文件条目
type fileEntry struct {
	name      string
	path      string
	isDir     bool
	isArchive bool
	size      int64
}

// Model 主应用模型
type model struct {
	state             appState
	mode              opMode
	modeCursor        int
	entries           []fileEntry
	cursor            int
	cwd               string
	width             int
	height            int

	formatCursor      int
	formats           []config.ArchiveFormat
	excludeCategories []config.ExcludeCategory
	excludeCursor     int

	selectedPath      string
	selectedFormat    config.ArchiveFormat
	outputPath        string
	password          string
	passwordInput     string
	usePassword       bool
	passwordCursor    int // 0: 不使用密码, 1: 使用密码

	progress          progress.Model
	spinner           spinner.Model
	compressStats     archiver.CompressStats
	extractStats      archiver.ExtractStats

	// 速度统计
	speedHistory      []float64  // 速度历史记录
	lastBytes         int64      // 上次记录的字节数
	lastTime          time.Time  // 上次记录时间
	currentSpeed      float64    // 当前速度 (bytes/s)
	avgSpeed          float64    // 平均速度
	startTime         time.Time  // 开始时间
	errorMsg          string

	operationCtx      context.Context
	operationCancel   context.CancelFunc
}

// CompressProgressMsg 压缩进度消息
type compressProgressMsg struct {
	current     int
	total       int
	currentFile string
	stats       archiver.CompressStats
}

// ExtractProgressMsg 解压进度消息
type extractProgressMsg struct {
	current     int
	total       int
	currentFile string
	stats       archiver.ExtractStats
}

// CompressDoneMsg 压缩完成消息
type compressDoneMsg struct {
	stats *archiver.CompressStats
	err   error
}

// ExtractDoneMsg 解压完成消息
type extractDoneMsg struct {
	stats *archiver.ExtractStats
	err   error
}

// tickMsg 定时器消息
type tickMsg time.Time

// newModel 创建新的应用模型
func newModel() model {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
		progress.WithoutPercentage(),
	)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	m := model{
		state:             stateSelectMode,
		mode:              modeCompress,
		cwd:               cwd,
		formats:           config.GetArchiveFormats(),
		excludeCategories: config.GetExcludeCategories(),
		progress:          p,
		spinner:           s,
		width:             80,
		height:            24,
	}
	return m
}

// loadEntries 加载当前目录的文件列表
func (m *model) loadEntries() {
	m.entries = []fileEntry{}

	entries, err := os.ReadDir(m.cwd)
	if err != nil {
		return
	}

	// 分离目录、压缩文件和普通文件
	var dirs, archives, files []fileEntry

	for _, entry := range entries {
		// 跳过隐藏文件
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		fe := fileEntry{
			name:  entry.Name(),
			path:  filepath.Join(m.cwd, entry.Name()),
			isDir: entry.IsDir(),
		}

		if !entry.IsDir() {
			fe.size = info.Size()
			fe.isArchive = archiver.IsArchiveFile(entry.Name())
		}

		if entry.IsDir() {
			dirs = append(dirs, fe)
		} else if fe.isArchive {
			archives = append(archives, fe)
		} else {
			files = append(files, fe)
		}
	}

	// 根据模式排序
	if m.mode == modeExtract {
		// 解压模式：压缩文件在前
		m.entries = append(archives, dirs...)
		m.entries = append(m.entries, files...)
	} else {
		// 压缩模式：目录在前
		m.entries = append(dirs, archives...)
		m.entries = append(m.entries, files...)
	}
	m.cursor = 0
}

// Init 初始化
func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
	)
}

// Update 更新
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = m.width - 20
		if m.progress.Width < 20 {
			m.progress.Width = 20
		}

	case tea.KeyMsg:
		// 全局退出
		if key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))) {
			if m.operationCancel != nil {
				m.operationCancel()
			}
			return m, tea.Quit
		}

		switch m.state {
		case stateSelectMode:
			return m.updateSelectMode(msg)
		case stateSelectFile:
			return m.updateSelectFile(msg)
		case stateSelectFormat:
			return m.updateSelectFormat(msg)
		case stateSelectExcludes:
			return m.updateSelectExcludes(msg)
		case stateInputPassword:
			return m.updateInputPassword(msg)
		case stateConfirm:
			return m.updateConfirm(msg)
		case stateDone, stateError:
			if key.Matches(msg, key.NewBinding(key.WithKeys("q", "esc", "enter"))) {
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)

	case compressProgressMsg:
		if msg.total > 0 {
			percent := float64(msg.current) / float64(msg.total)
			cmds = append(cmds, m.progress.SetPercent(percent))
		}
		m.compressStats = msg.stats

	case extractProgressMsg:
		if msg.total > 0 {
			percent := float64(msg.current) / float64(msg.total)
			cmds = append(cmds, m.progress.SetPercent(percent))
		}
		m.extractStats = msg.stats

	case compressDoneMsg:
		if msg.err != nil {
			m.state = stateError
			m.errorMsg = msg.err.Error()
		} else {
			m.state = stateDone
			if msg.stats != nil {
				m.compressStats = *msg.stats
			}
		}

	case extractDoneMsg:
		if msg.err != nil {
			m.state = stateError
			m.errorMsg = msg.err.Error()
		} else {
			m.state = stateDone
			if msg.stats != nil {
				m.extractStats = *msg.stats
			}
		}

	case tickMsg:
		if m.state == stateCompressing || m.state == stateExtracting {
			// 计算速度
			m.updateSpeed()
			cmds = append(cmds, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
				return tickMsg(t)
			}))
		}
	}

	return m, tea.Batch(cmds...)
}

// updateSelectMode 更新模式选择状态
func (m model) updateSelectMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, tea.Quit

	case "up", "k":
		if m.modeCursor > 0 {
			m.modeCursor--
		}

	case "down", "j":
		if m.modeCursor < 1 {
			m.modeCursor++
		}

	case "enter", " ":
		if m.modeCursor == 0 {
			m.mode = modeCompress
		} else {
			m.mode = modeExtract
		}
		m.state = stateSelectFile
		m.loadEntries()
	}

	return m, nil
}

// updateSelectFile 更新文件选择状态
func (m model) updateSelectFile(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = stateSelectMode

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}

	case "enter", "l":
		if len(m.entries) > 0 && m.entries[m.cursor].isDir {
			m.cwd = m.entries[m.cursor].path
			m.loadEntries()
		}

	case "backspace", "h":
		parent := filepath.Dir(m.cwd)
		if parent != m.cwd {
			m.cwd = parent
			m.loadEntries()
		}

	case " ":
		if len(m.entries) > 0 {
			entry := m.entries[m.cursor]
			m.selectedPath = entry.path

			if m.mode == modeExtract {
				// 解压模式：只能选择压缩文件
				if entry.isArchive {
					// 自动生成解压目录名
					baseName := filepath.Base(entry.path)
					// 移除所有扩展名
					for {
						ext := filepath.Ext(baseName)
						if ext == "" || (!strings.HasPrefix(ext, ".tar") && ext != ".zip" && ext != ".gz" && ext != ".bz2" && ext != ".xz" && ext != ".zst" && ext != ".lz4" && ext != ".tgz" && ext != ".tbz2" && ext != ".txz" && ext != ".7z") {
							break
						}
						baseName = strings.TrimSuffix(baseName, ext)
					}
					m.outputPath = filepath.Join(filepath.Dir(entry.path), baseName)
					
					// 检测是否是支持密码的格式（ZIP或7z）
					format := archiver.DetectArchiveFormat(entry.path)
					if format == ".zip" || format == ".7z" {
						// 进入密码输入界面
						m.state = stateInputPassword
						m.passwordCursor = 0
						m.passwordInput = ""
					} else {
						m.state = stateConfirm
					}
				}
			} else {
				// 压缩模式
				m.state = stateSelectFormat
			}
		}
	}

	return m, nil
}

// updateSelectFormat 更新格式选择状态
func (m model) updateSelectFormat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = stateSelectFile

	case "up", "k":
		if m.formatCursor > 0 {
			m.formatCursor--
		}

	case "down", "j":
		if m.formatCursor < len(m.formats)-1 {
			m.formatCursor++
		}

	case "enter", " ":
		m.selectedFormat = m.formats[m.formatCursor]
		m.outputPath = m.selectedPath + m.selectedFormat.Extension
		m.state = stateSelectExcludes
	}

	return m, nil
}

// updateSelectExcludes 更新排除规则选择状态
func (m model) updateSelectExcludes(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.state = stateSelectFormat

	case "up", "k":
		if m.excludeCursor > 0 {
			m.excludeCursor--
		}

	case "down", "j":
		if m.excludeCursor < len(m.excludeCategories)-1 {
			m.excludeCursor++
		}

	case " ":
		m.excludeCategories[m.excludeCursor].Selected = !m.excludeCategories[m.excludeCursor].Selected

	case "a":
		for i := range m.excludeCategories {
			m.excludeCategories[i].Selected = true
		}

	case "n":
		for i := range m.excludeCategories {
			m.excludeCategories[i].Selected = false
		}

	case "enter":
		// 如果是ZIP格式，询问是否加密
		if m.selectedFormat.Extension == ".zip" {
			m.state = stateInputPassword
			m.passwordCursor = 0
		} else {
			m.state = stateConfirm
		}
	}

	return m, nil
}

// updateInputPassword 更新密码输入状态
func (m model) updateInputPassword(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// 解压模式：简化的密码输入（只有输入密码选项）
	if m.mode == modeExtract {
		switch msg.String() {
		case "q", "esc":
			m.state = stateSelectFile
			m.passwordInput = ""
			m.password = ""

		case "enter":
			// 确认密码（可以为空，表示尝试无密码解压）
			m.password = m.passwordInput
			m.state = stateConfirm

		case "backspace":
			if len(m.passwordInput) > 0 {
				m.passwordInput = m.passwordInput[:len(m.passwordInput)-1]
			}

		default:
			// 记录输入
			if len(msg.String()) == 1 {
				m.passwordInput += msg.String()
			}
		}
		return m, nil
	}

	// 压缩模式：选择是否使用密码
	switch msg.String() {
	case "q", "esc":
		m.state = stateSelectExcludes
		m.passwordInput = ""
		m.usePassword = false

	case "up", "k":
		if m.passwordCursor > 0 {
			m.passwordCursor--
		}

	case "down", "j":
		if m.passwordCursor < 1 {
			m.passwordCursor++
		}

	case "enter":
		if m.passwordCursor == 0 {
			// 不使用密码
			m.usePassword = false
			m.password = ""
			m.state = stateConfirm
		} else {
			// 使用密码 - 如果还没输入密码，等待输入
			if m.passwordInput == "" {
				// 密码输入提示已显示，等待输入
				return m, nil
			}
			m.usePassword = true
			m.password = m.passwordInput
			m.state = stateConfirm
		}

	case "backspace":
		if m.passwordCursor == 1 && len(m.passwordInput) > 0 {
			m.passwordInput = m.passwordInput[:len(m.passwordInput)-1]
		}

	default:
		// 如果选择了使用密码，记录输入
		if m.passwordCursor == 1 && len(msg.String()) == 1 {
			m.passwordInput += msg.String()
		}
	}

	return m, nil
}

// updateConfirm 更新确认状态
func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "n":
		if m.mode == modeExtract {
			// 检测是否是支持密码的格式
			format := archiver.DetectArchiveFormat(m.selectedPath)
			if format == ".zip" || format == ".7z" {
				m.state = stateInputPassword
			} else {
				m.state = stateSelectFile
			}
		} else if m.selectedFormat.Extension == ".zip" {
			m.state = stateInputPassword
		} else {
			m.state = stateSelectExcludes
		}

	case "y", "enter":
		// 初始化速度统计
		m.speedHistory = make([]float64, 0, 30)
		m.lastBytes = 0
		m.lastTime = time.Now()
		m.startTime = time.Now()
		m.currentSpeed = 0
		m.avgSpeed = 0

		if m.mode == modeExtract {
			m.state = stateExtracting
			return m, tea.Batch(
				m.startExtract(),
				tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
					return tickMsg(t)
				}),
			)
		} else {
			m.state = stateCompressing
			return m, tea.Batch(
				m.startCompress(),
				tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
					return tickMsg(t)
				}),
			)
		}
	}

	return m, nil
}

// startCompress 开始压缩
func (m *model) startCompress() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.operationCtx = ctx
		m.operationCancel = cancel

		// 收集排除模式
		var excludes []string
		for _, cat := range m.excludeCategories {
			if cat.Selected {
				excludes = append(excludes, cat.Patterns...)
			}
		}

		opts := archiver.CompressOptions{
			Source:   m.selectedPath,
			Output:   m.outputPath,
			Format:   m.selectedFormat.Extension,
			Excludes: excludes,
			Password: m.password,
		}

		stats, err := archiver.Compress(ctx, opts)
		if err != nil {
			return compressDoneMsg{stats: nil, err: err}
		}

		return compressDoneMsg{stats: stats, err: nil}
	}
}

// startExtract 开始解压
func (m *model) startExtract() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.operationCtx = ctx
		m.operationCancel = cancel

		opts := archiver.ExtractOptions{
			Source:   m.selectedPath,
			Output:   m.outputPath,
			Password: m.password,
		}

		stats, err := archiver.Extract(ctx, opts)
		if err != nil {
			return extractDoneMsg{stats: nil, err: err}
		}

		return extractDoneMsg{stats: stats, err: nil}
	}
}

// updateSpeed 更新速度统计
func (m *model) updateSpeed() {
	now := time.Now()
	elapsed := now.Sub(m.lastTime).Seconds()
	if elapsed < 0.1 {
		return // 避免除以太小的数
	}

	var currentBytes int64
	if m.state == stateCompressing {
		currentBytes = m.compressStats.CompressedSize
	} else {
		currentBytes = m.extractStats.ExtractedSize
	}

	// 计算当前速度
	bytesDiff := currentBytes - m.lastBytes
	if bytesDiff >= 0 {
		m.currentSpeed = float64(bytesDiff) / elapsed
	}

	// 更新历史记录
	m.speedHistory = append(m.speedHistory, m.currentSpeed)
	if len(m.speedHistory) > 30 {
		m.speedHistory = m.speedHistory[1:]
	}

	// 计算平均速度
	totalElapsed := now.Sub(m.startTime).Seconds()
	if totalElapsed > 0 {
		m.avgSpeed = float64(currentBytes) / totalElapsed
	}

	m.lastBytes = currentBytes
	m.lastTime = now
}

// renderSparkline 渲染速度图表
func (m model) renderSparkline() string {
	if len(m.speedHistory) == 0 {
		return ""
	}

	// Unicode sparkline 字符
	sparkChars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// 找到最大值用于归一化
	var maxSpeed float64
	for _, s := range m.speedHistory {
		if s > maxSpeed {
			maxSpeed = s
		}
	}

	if maxSpeed == 0 {
		return strings.Repeat(string(sparkChars[0]), len(m.speedHistory))
	}

	var sb strings.Builder
	for _, s := range m.speedHistory {
		idx := int((s / maxSpeed) * float64(len(sparkChars)-1))
		if idx >= len(sparkChars) {
			idx = len(sparkChars) - 1
		}
		if idx < 0 {
			idx = 0
		}
		sb.WriteRune(sparkChars[idx])
	}

	return sb.String()
}

// formatSpeed 格式化速度显示
func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	} else if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	} else if bytesPerSec < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB/s", bytesPerSec/(1024*1024*1024))
}

// formatDuration 格式化时间显示
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	} else if d < time.Hour {
		mins := int(d.Minutes())
		secs := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// renderStatusBar 渲染 Zellij 风格的底部快捷键提示栏
func (m model) renderStatusBar() string {
	type keyHint struct {
		key  string
		desc string
	}

	var hints []keyHint

	switch m.state {
	case stateSelectMode:
		hints = []keyHint{
			{"↑/k", "上移"},
			{"↓/j", "下移"},
			{"Enter", "选择"},
			{"q", "退出"},
		}
	case stateSelectFile:
		hints = []keyHint{
			{"↑/k", "上移"},
			{"↓/j", "下移"},
			{"Enter/l", "进入"},
			{"h/BS", "返回"},
			{"Space", "选择"},
			{"Esc", "返回"},
		}
	case stateSelectFormat:
		hints = []keyHint{
			{"↑/k", "上移"},
			{"↓/j", "下移"},
			{"Enter", "确认"},
			{"Esc", "返回"},
		}
	case stateSelectExcludes:
		hints = []keyHint{
			{"↑/k", "上移"},
			{"↓/j", "下移"},
			{"Space", "切换"},
			{"a", "全选"},
			{"n", "清除"},
			{"Enter", "确认"},
			{"Esc", "返回"},
		}
	case stateInputPassword:
		hints = []keyHint{
			{"输入", "密码"},
			{"Enter", "确认"},
			{"Esc", "返回"},
		}
	case stateConfirm:
		hints = []keyHint{
			{"y/Enter", "确认"},
			{"n/Esc", "返回"},
		}
	case stateCompressing, stateExtracting:
		hints = []keyHint{
			{"Ctrl+C", "取消"},
		}
	case stateDone, stateError:
		hints = []keyHint{
			{"Enter/q", "退出"},
		}
	}

	var parts []string
	for _, h := range hints {
		key := statusKeyStyle.Render(h.key)
		desc := statusDescStyle.Render(h.desc)
		parts = append(parts, key+desc)
	}

	sep := statusSepStyle.Render(" │ ")
	content := strings.Join(parts, sep)

	// 创建全宽状态栏
	bar := statusBarStyle.Width(m.width).Render(content)
	return bar
}

// View 渲染视图
func (m model) View() string {
	var sb strings.Builder

	// 标题
	modeStr := "压缩"
	if m.mode == modeExtract {
		modeStr = "解压"
	}
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(foregroundColor).
		Background(primaryColor).
		Padding(0, 2).
		MarginBottom(1).
		Render(fmt.Sprintf("📦 %s v%s - %s模式", AppName, AppVersion, modeStr))
	sb.WriteString(header)
	sb.WriteString("\n\n")

	// 主内容区域
	var content string
	switch m.state {
	case stateSelectMode:
		content = m.viewSelectMode()
	case stateSelectFile:
		content = m.viewSelectFile()
	case stateSelectFormat:
		content = m.viewSelectFormat()
	case stateSelectExcludes:
		content = m.viewSelectExcludes()
	case stateInputPassword:
		content = m.viewInputPassword()
	case stateConfirm:
		content = m.viewConfirm()
	case stateCompressing:
		content = m.viewCompressing()
	case stateExtracting:
		content = m.viewExtracting()
	case stateDone:
		content = m.viewDone()
	case stateError:
		content = m.viewError()
	}
	sb.WriteString(content)

	// 计算需要填充的空行数，使状态栏固定在底部
	contentLines := strings.Count(sb.String(), "\n") + 1
	statusBarHeight := 1
	headerHeight := 3 // 标题区域高度
	availableHeight := m.height - statusBarHeight - headerHeight

	if contentLines < availableHeight {
		for i := 0; i < availableHeight-contentLines; i++ {
			sb.WriteString("\n")
		}
	}

	// 添加底部状态栏
	sb.WriteString("\n")
	sb.WriteString(m.renderStatusBar())

	return sb.String()
}

// viewSelectMode 渲染模式选择视图
func (m model) viewSelectMode() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("🎯 选择操作模式"))
	sb.WriteString("\n\n")

	modes := []struct {
		icon string
		name string
		desc string
	}{
		{"🗜️", "压缩文件/文件夹", "将文件或文件夹压缩为归档文件"},
		{"📂", "解压归档文件", "将压缩包解压到指定目录"},
	}

	for i, mode := range modes {
		cursor := "  "
		if i == m.modeCursor {
			cursor = "▸ "
		}

		icon := mode.icon
		var name string
		if i == m.modeCursor {
			name = selectedStyle.Render(mode.name)
		} else {
			name = normalStyle.Render(mode.name)
		}

		desc := subtitleStyle.Render(" - " + mode.desc)
		sb.WriteString(fmt.Sprintf("%s%s %s%s\n", cursor, icon, name, desc))
	}

	return borderStyle.Render(sb.String())
}

// viewSelectFile 渲染文件选择视图
func (m model) viewSelectFile() string {
	var sb strings.Builder

	if m.mode == modeExtract {
		sb.WriteString(titleStyle.Render("📂 选择要解压的归档文件"))
	} else {
		sb.WriteString(titleStyle.Render("📂 选择要压缩的文件或文件夹"))
	}
	sb.WriteString("\n")

	// 当前路径
	pathStyle := lipgloss.NewStyle().
		Foreground(secondaryColor).
		Bold(true)
	sb.WriteString(pathStyle.Render("📍 " + m.cwd))
	sb.WriteString("\n\n")

	// 文件列表
	visibleHeight := m.height - 15
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	start := 0
	if m.cursor >= visibleHeight {
		start = m.cursor - visibleHeight + 1
	}

	end := start + visibleHeight
	if end > len(m.entries) {
		end = len(m.entries)
	}

	if len(m.entries) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render("  (空目录)"))
		sb.WriteString("\n")
	}

	for i := start; i < end; i++ {
		entry := m.entries[i]
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}

		var line string
		if entry.isDir {
			icon := folderIconStyle.Render("📁")
			name := entry.name + "/"
			if i == m.cursor {
				name = selectedStyle.Render(name)
			} else {
				name = normalStyle.Render(name)
			}
			line = fmt.Sprintf("%s%s %s", cursor, icon, name)
		} else if entry.isArchive {
			icon := archiveIconStyle.Render("📦")
			name := entry.name
			size := formatFileSize(entry.size)
			if i == m.cursor {
				name = selectedStyle.Render(name)
			} else {
				name = normalStyle.Render(name)
			}
			sizeStr := lipgloss.NewStyle().Foreground(mutedColor).Render("(" + size + ")")
			line = fmt.Sprintf("%s%s %s %s", cursor, icon, name, sizeStr)
		} else {
			icon := fileIconStyle.Render("📄")
			name := entry.name
			size := formatFileSize(entry.size)
			if i == m.cursor {
				name = selectedStyle.Render(name)
			} else {
				name = normalStyle.Render(name)
			}
			sizeStr := lipgloss.NewStyle().Foreground(mutedColor).Render("(" + size + ")")
			line = fmt.Sprintf("%s%s %s %s", cursor, icon, name, sizeStr)
		}

		sb.WriteString(line)
		sb.WriteString("\n")
	}

	// 滚动指示器
	if len(m.entries) > visibleHeight {
		scrollInfo := lipgloss.NewStyle().Foreground(mutedColor).Render(
			fmt.Sprintf("\n  显示 %d-%d / %d", start+1, end, len(m.entries)),
		)
		sb.WriteString(scrollInfo)
	}

	return borderStyle.Render(sb.String())
}

// viewSelectFormat 渲染格式选择视图
func (m model) viewSelectFormat() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("📦 选择压缩格式"))
	sb.WriteString("\n")
	sb.WriteString(subtitleStyle.Render("已选择: " + filepath.Base(m.selectedPath)))
	sb.WriteString("\n\n")

	for i, format := range m.formats {
		cursor := "  "
		if i == m.formatCursor {
			cursor = "▸ "
		}

		var name string
		if i == m.formatCursor {
			name = selectedStyle.Render(format.Name)
		} else {
			name = normalStyle.Render(format.Name)
		}

		desc := subtitleStyle.Render(" - " + format.Description)
		sb.WriteString(fmt.Sprintf("%s%s%s\n", cursor, name, desc))
	}

	return borderStyle.Render(sb.String())
}

// viewSelectExcludes 渲染排除规则选择视图
func (m model) viewSelectExcludes() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("🚫 选择排除规则"))
	sb.WriteString("\n")
	sb.WriteString(subtitleStyle.Render("格式: " + m.selectedFormat.Name + " | 空格切换选中状态"))
	sb.WriteString("\n\n")

	for i, cat := range m.excludeCategories {
		cursor := "  "
		if i == m.excludeCursor {
			cursor = "▸ "
		}

		checkbox := "☐"
		checkStyle := lipgloss.NewStyle().Foreground(mutedColor)
		if cat.Selected {
			checkbox = "☑"
			checkStyle = lipgloss.NewStyle().Foreground(successColor)
		}

		var name string
		if i == m.excludeCursor {
			name = selectedStyle.Render(cat.Name)
		} else if cat.Selected {
			name = normalStyle.Render(cat.Name)
		} else {
			name = disabledStyle.Render(cat.Name)
		}

		// 显示部分模式
		patterns := cat.Patterns
		if len(patterns) > 3 {
			patterns = patterns[:3]
		}
		patternsStr := subtitleStyle.Render(" (" + strings.Join(patterns, ", ") + "...)")

		sb.WriteString(fmt.Sprintf("%s%s %s%s\n", cursor, checkStyle.Render(checkbox), name, patternsStr))
	}

	return borderStyle.Render(sb.String())
}

// viewInputPassword 渲染密码输入视图
func (m model) viewInputPassword() string {
	var sb strings.Builder

	// 解压模式：直接输入密码
	if m.mode == modeExtract {
		sb.WriteString(titleStyle.Render("🔐 输入解压密码"))
		sb.WriteString("\n")
		sb.WriteString(subtitleStyle.Render("如果归档文件有密码保护，请输入密码"))
		sb.WriteString("\n\n")

		sb.WriteString(statLabelStyle.Render("文件:"))
		sb.WriteString(statValueStyle.Render(filepath.Base(m.selectedPath)))
		sb.WriteString("\n\n")

		sb.WriteString(statLabelStyle.Render("密码:"))
		passwordDisplay := strings.Repeat("●", len(m.passwordInput))
		if passwordDisplay == "" {
			passwordDisplay = lipgloss.NewStyle().Foreground(mutedColor).Render("(留空=无密码，直接Enter确认)")
		} else {
			passwordDisplay = infoStyle.Render(passwordDisplay)
		}
		sb.WriteString(passwordDisplay)
		sb.WriteString("\n")

		return borderStyle.Render(sb.String())
	}

	// 压缩模式：选择是否使用密码
	sb.WriteString(titleStyle.Render("🔐 密码保护设置"))
	sb.WriteString("\n")
	sb.WriteString(subtitleStyle.Render("ZIP格式支持 AES-256 加密保护"))
	sb.WriteString("\n\n")

	options := []struct {
		icon string
		name string
		desc string
	}{
		{"🔓", "不使用密码", "生成普通ZIP文件"},
		{"🔒", "设置密码", "使用 AES-256 加密"},
	}

	for i, opt := range options {
		cursor := "  "
		if i == m.passwordCursor {
			cursor = "▸ "
		}

		icon := opt.icon
		var name string
		if i == m.passwordCursor {
			name = selectedStyle.Render(opt.name)
		} else {
			name = normalStyle.Render(opt.name)
		}

		desc := subtitleStyle.Render(" - " + opt.desc)
		sb.WriteString(fmt.Sprintf("%s%s %s%s\n", cursor, icon, name, desc))
	}

	// 如果选择了使用密码，显示密码输入框
	if m.passwordCursor == 1 {
		sb.WriteString("\n")
		sb.WriteString(statLabelStyle.Render("输入密码:"))
		passwordDisplay := strings.Repeat("●", len(m.passwordInput))
		if passwordDisplay == "" {
			passwordDisplay = lipgloss.NewStyle().Foreground(mutedColor).Render("(输入密码后按Enter确认)")
		} else {
			passwordDisplay = infoStyle.Render(passwordDisplay)
		}
		sb.WriteString(passwordDisplay)
		sb.WriteString("\n")
	}

	return borderStyle.Render(sb.String())
}

// viewConfirm 渲染确认视图
func (m model) viewConfirm() string {
	var sb strings.Builder

	if m.mode == modeExtract {
		sb.WriteString(titleStyle.Render("✅ 确认解压"))
	} else {
		sb.WriteString(titleStyle.Render("✅ 确认压缩"))
	}
	sb.WriteString("\n\n")

	// 源文件
	sb.WriteString(statLabelStyle.Render("源文件:"))
	sb.WriteString(statValueStyle.Render(filepath.Base(m.selectedPath)))
	sb.WriteString("\n")

	// 输出
	if m.mode == modeExtract {
		sb.WriteString(statLabelStyle.Render("解压到:"))
		sb.WriteString(statValueStyle.Render(filepath.Base(m.outputPath) + "/"))
		sb.WriteString("\n")

		// 显示密码状态（解压模式）
		format := archiver.DetectArchiveFormat(m.selectedPath)
		if format == ".zip" || format == ".7z" {
			sb.WriteString(statLabelStyle.Render("解压密码:"))
			if m.password != "" {
				sb.WriteString(infoStyle.Render("🔑 已设置"))
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render("🔓 无"))
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString(statLabelStyle.Render("输出文件:"))
		sb.WriteString(statValueStyle.Render(filepath.Base(m.outputPath)))
	}
	sb.WriteString("\n")

	if m.mode == modeCompress {
		// 压缩格式
		sb.WriteString(statLabelStyle.Render("压缩格式:"))
		sb.WriteString(infoStyle.Render(m.selectedFormat.Name))
		sb.WriteString("\n")

		// 密码保护
		if m.selectedFormat.Extension == ".zip" {
			sb.WriteString(statLabelStyle.Render("密码保护:"))
			if m.usePassword {
				sb.WriteString(successStyle.Render("🔒 AES-256 加密"))
			} else {
				sb.WriteString(lipgloss.NewStyle().Foreground(mutedColor).Render("🔓 无"))
			}
			sb.WriteString("\n")
		}

		// 排除规则数量
		excludeCount := 0
		for _, cat := range m.excludeCategories {
			if cat.Selected {
				excludeCount += len(cat.Patterns)
			}
		}
		sb.WriteString(statLabelStyle.Render("排除规则:"))
		sb.WriteString(warningStyle.Render(fmt.Sprintf("%d 个模式", excludeCount)))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	if m.mode == modeExtract {
		sb.WriteString(successStyle.Render("按 Y/Enter 开始解压，N/Esc 返回修改"))
	} else {
		sb.WriteString(successStyle.Render("按 Y/Enter 开始压缩，N/Esc 返回修改"))
	}

	return highlightBorderStyle.Render(sb.String())
}

// viewCompressing 渲染压缩中视图
func (m model) viewCompressing() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("🚀 正在压缩..."))
	sb.WriteString("\n\n")

	// Spinner
	sb.WriteString(m.spinner.View())
	sb.WriteString(" ")

	// 当前文件
	if m.compressStats.CurrentFile != "" {
		currentFile := m.compressStats.CurrentFile
		if len(currentFile) > 50 {
			currentFile = "..." + currentFile[len(currentFile)-47:]
		}
		sb.WriteString(infoStyle.Render(currentFile))
	} else {
		sb.WriteString(subtitleStyle.Render("准备中..."))
	}
	sb.WriteString("\n\n")

	// 进度条
	percent := 0.0
	if m.compressStats.TotalFiles > 0 {
		percent = float64(m.compressStats.ProcessedFiles) / float64(m.compressStats.TotalFiles)
	}
	sb.WriteString(m.progress.ViewAs(percent))
	sb.WriteString("\n\n")

	// 速度图表
	sparkline := m.renderSparkline()
	if sparkline != "" {
		speedColor := lipgloss.Color("#00D4FF")
		sparkStyle := lipgloss.NewStyle().Foreground(speedColor)
		sb.WriteString(statLabelStyle.Render("速度:"))
		sb.WriteString(sparkStyle.Render(sparkline))
		sb.WriteString("\n")
		
		// 当前速度和平均速度
		sb.WriteString(statLabelStyle.Render("当前:"))
		sb.WriteString(infoStyle.Render(formatSpeed(m.currentSpeed)))
		sb.WriteString("  ")
		sb.WriteString(statLabelStyle.Render("平均:"))
		sb.WriteString(infoStyle.Render(formatSpeed(m.avgSpeed)))
		sb.WriteString("\n")
	}

	// 统计信息
	sb.WriteString(statLabelStyle.Render("处理进度:"))
	sb.WriteString(statValueStyle.Render(fmt.Sprintf("%d / %d 文件", m.compressStats.ProcessedFiles, m.compressStats.TotalFiles)))
	sb.WriteString("\n")

	if m.compressStats.ExcludedFiles > 0 {
		sb.WriteString(statLabelStyle.Render("已排除:"))
		sb.WriteString(warningStyle.Render(fmt.Sprintf("%d 个文件/目录", m.compressStats.ExcludedFiles)))
		sb.WriteString("\n")
	}

	// 已用时间
	if !m.startTime.IsZero() {
		elapsed := time.Since(m.startTime)
		sb.WriteString(statLabelStyle.Render("已用时间:"))
		sb.WriteString(statValueStyle.Render(formatDuration(elapsed)))
		sb.WriteString("\n")
	}

	return highlightBorderStyle.Render(sb.String())
}

// viewExtracting 渲染解压中视图
func (m model) viewExtracting() string {
	var sb strings.Builder

	sb.WriteString(titleStyle.Render("📂 正在解压..."))
	sb.WriteString("\n\n")

	// Spinner
	sb.WriteString(m.spinner.View())
	sb.WriteString(" ")

	// 当前文件
	if m.extractStats.CurrentFile != "" {
		currentFile := m.extractStats.CurrentFile
		if len(currentFile) > 50 {
			currentFile = "..." + currentFile[len(currentFile)-47:]
		}
		sb.WriteString(infoStyle.Render(currentFile))
	} else {
		sb.WriteString(subtitleStyle.Render("准备中..."))
	}
	sb.WriteString("\n\n")

	// 进度条
	percent := 0.0
	if m.extractStats.TotalFiles > 0 {
		percent = float64(m.extractStats.ProcessedFiles) / float64(m.extractStats.TotalFiles)
	}
	sb.WriteString(m.progress.ViewAs(percent))
	sb.WriteString("\n\n")

	// 速度图表
	sparkline := m.renderSparkline()
	if sparkline != "" {
		speedColor := lipgloss.Color("#00D4FF")
		sparkStyle := lipgloss.NewStyle().Foreground(speedColor)
		sb.WriteString(statLabelStyle.Render("速度:"))
		sb.WriteString(sparkStyle.Render(sparkline))
		sb.WriteString("\n")
		
		// 当前速度和平均速度
		sb.WriteString(statLabelStyle.Render("当前:"))
		sb.WriteString(infoStyle.Render(formatSpeed(m.currentSpeed)))
		sb.WriteString("  ")
		sb.WriteString(statLabelStyle.Render("平均:"))
		sb.WriteString(infoStyle.Render(formatSpeed(m.avgSpeed)))
		sb.WriteString("\n")
	}

	// 统计信息
	sb.WriteString(statLabelStyle.Render("处理进度:"))
	if m.extractStats.TotalFiles > 0 {
		sb.WriteString(statValueStyle.Render(fmt.Sprintf("%d / %d 文件", m.extractStats.ProcessedFiles, m.extractStats.TotalFiles)))
	} else {
		sb.WriteString(statValueStyle.Render(fmt.Sprintf("%d 文件", m.extractStats.ProcessedFiles)))
	}
	sb.WriteString("\n")

	// 已用时间
	if !m.startTime.IsZero() {
		elapsed := time.Since(m.startTime)
		sb.WriteString(statLabelStyle.Render("已用时间:"))
		sb.WriteString(statValueStyle.Render(formatDuration(elapsed)))
		sb.WriteString("\n")
	}

	return highlightBorderStyle.Render(sb.String())
}

// viewDone 渲染完成视图
func (m model) viewDone() string {
	var sb strings.Builder

	if m.mode == modeExtract {
		sb.WriteString(successStyle.Render("🎉 解压完成！"))
		sb.WriteString("\n\n")

		// 输出目录
		sb.WriteString(statLabelStyle.Render("解压到:"))
		sb.WriteString(statValueStyle.Render(filepath.Base(m.outputPath) + "/"))
		sb.WriteString("\n")

		// 解压文件数
		sb.WriteString(statLabelStyle.Render("解压文件:"))
		sb.WriteString(statValueStyle.Render(fmt.Sprintf("%d 个", m.extractStats.TotalFiles)))
		sb.WriteString("\n")

		// 解压大小
		sb.WriteString(statLabelStyle.Render("解压大小:"))
		sb.WriteString(successStyle.Render(formatFileSize(m.extractStats.ExtractedSize)))
		sb.WriteString("\n")
	} else {
		sb.WriteString(successStyle.Render("🎉 压缩完成！"))
		sb.WriteString("\n\n")

		// 输出文件
		sb.WriteString(statLabelStyle.Render("输出文件:"))
		sb.WriteString(statValueStyle.Render(filepath.Base(m.outputPath)))
		sb.WriteString("\n")

		// 压缩文件数
		sb.WriteString(statLabelStyle.Render("压缩文件:"))
		sb.WriteString(statValueStyle.Render(fmt.Sprintf("%d 个", m.compressStats.TotalFiles)))
		sb.WriteString("\n")

		// 原始大小
		sb.WriteString(statLabelStyle.Render("原始大小:"))
		sb.WriteString(infoStyle.Render(formatFileSize(m.compressStats.TotalSize)))
		sb.WriteString("\n")

		// 压缩后大小
		sb.WriteString(statLabelStyle.Render("压缩后大小:"))
		sb.WriteString(successStyle.Render(formatFileSize(m.compressStats.CompressedSize)))
		sb.WriteString("\n")

		// 压缩率
		sb.WriteString(statLabelStyle.Render("压缩率:"))
		sb.WriteString(successStyle.Render(fmt.Sprintf("%.1f%%", m.compressStats.CompressionRate)))
		sb.WriteString("\n")

		// 排除文件数
		if m.compressStats.ExcludedFiles > 0 {
			sb.WriteString(statLabelStyle.Render("排除文件:"))
			sb.WriteString(warningStyle.Render(fmt.Sprintf("%d 个", m.compressStats.ExcludedFiles)))
			sb.WriteString("\n")
		}
	}

	return highlightBorderStyle.Render(sb.String())
}

// viewError 渲染错误视图
func (m model) viewError() string {
	var sb strings.Builder

	if m.mode == modeExtract {
		sb.WriteString(errorStyle.Render("❌ 解压失败"))
	} else {
		sb.WriteString(errorStyle.Render("❌ 压缩失败"))
	}
	sb.WriteString("\n\n")

	sb.WriteString(statLabelStyle.Render("错误信息:"))
	sb.WriteString("\n")
	sb.WriteString(errorStyle.Render(m.errorMsg))

	return borderStyle.Render(sb.String())
}

// formatFileSize 格式化文件大小
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size < KB:
		return fmt.Sprintf("%d B", size)
	case size < MB:
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	case size < GB:
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	default:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	}
}

func main() {
	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("启动失败: %v\n", err)
		os.Exit(1)
	}
}
