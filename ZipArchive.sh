#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[1;37m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m' # No Color

# 光标控制
SAVE_CURSOR='\033[s'
RESTORE_CURSOR='\033[u'
CLEAR_LINE='\033[2K'
HIDE_CURSOR='\033[?25l'
SHOW_CURSOR='\033[?25h'

# 默认排除模式列表
DEFAULT_EXCLUDES=(
    # Python
    "venv/*"
    ".venv/*"
    "__pycache__/*"
    "*.pyc"
    "*.pyo"
    ".pytest_cache/*"
    ".mypy_cache/*"
    "*.egg-info/*"
    ".eggs/*"
    # Node.js
    "node_modules/*"
    ".npm/*"
    ".pnpm-store/*"
    # IDE/Editor
    ".idea/*"
    ".vscode/*"
    "*.swp"
    "*.swo"
    "*~"
    # Git
    ".git/*"
    # 构建产物
    "dist/*"
    "build/*"
    "target/*"
    "out/*"
    # 系统文件
    ".DS_Store"
    "Thumbs.db"
    "desktop.ini"
    # 日志和缓存
    "*.log"
    ".cache/*"
    ".temp/*"
    ".tmp/*"
    # 环境配置（可选，默认不排除敏感文件由用户决定）
    # ".env"
    # ".env.local"
    # Go
    "vendor/*"
    # Rust
    "target/*"
    # Java/Maven/Gradle
    ".gradle/*"
    ".m2/*"
)

# 用户选择的排除模式
EXCLUDE_PATTERNS=()

# 清理函数
cleanup() {
    echo -e "${SHOW_CURSOR}"
    if [ -n "$TEMP_ZIP" ] && [ -f "$TEMP_ZIP" ]; then
        rm -f "$TEMP_ZIP"
    fi
}
trap cleanup EXIT INT TERM

# 打印标题
print_header() {
    clear
    echo -e "${CYAN}╔════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║${WHITE}${BOLD}           📦 智能文件压缩工具 📦                     ${CYAN}║${NC}"
    echo -e "${CYAN}╚════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# 生成随机后缀
generate_random_suffix() {
    cat /dev/urandom | tr -dc 'A-Za-z0-9' | fold -w 4 | head -n 1
}

# 格式化文件大小
format_size() {
    local size=$1
    if [ $size -lt 1024 ]; then
        echo "${size}B"
    elif [ $size -lt 1048576 ]; then
        echo "$(awk "BEGIN {printf \"%.2f\", $size/1024}")KB"
    elif [ $size -lt 1073741824 ]; then
        echo "$(awk "BEGIN {printf \"%.2f\", $size/1048576}")MB"
    else
        echo "$(awk "BEGIN {printf \"%.2f\", $size/1073741824}")GB"
    fi
}

# 格式化时间
format_time() {
    local timestamp=$1
    date -d "@$timestamp" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || date -r "$timestamp" "+%Y-%m-%d %H:%M:%S"
}

# 选择文件或文件夹
select_target() {
    print_header
    echo -e "${YELLOW}📂 当前目录下的文件和文件夹：${NC}\n"
    
    local items=()
    local index=1
    
    # 列出所有文件和文件夹（排除隐藏文件和当前脚本）
    while IFS= read -r item; do
        if [ -d "$item" ]; then
            echo -e "${BLUE}  [$index]${NC} 📁 ${GREEN}$item/${NC}"
        else
            echo -e "${BLUE}  [$index]${NC} 📄 ${WHITE}$item${NC}"
        fi
        items+=("$item")
        ((index++))
    done < <(ls -1A | grep -v "^$(basename "$0")$")
    
    if [ ${#items[@]} -eq 0 ]; then
        echo -e "${RED}❌ 当前目录下没有可压缩的文件或文件夹！${NC}"
        exit 1
    fi
    
    echo ""
    echo -e "${RED}  [0]${NC} 🚪 ${DIM}退出程序${NC}"
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    while true; do
        echo -ne "${YELLOW}请输入序号选择要压缩的目标 [0-${#items[@]}] (0=退出): ${NC}"
        read selection
        
        # 退出选项
        if [[ "$selection" == "0" ]] || [[ "$selection" == "q" ]] || [[ "$selection" == "Q" ]]; then
            echo -e "\n${GREEN}👋 感谢使用，再见！${NC}\n"
            exit 0
        fi
        
        if [[ "$selection" =~ ^[0-9]+$ ]] && [ "$selection" -ge 1 ] && [ "$selection" -le ${#items[@]} ]; then
            TARGET="${items[$((selection-1))]}"
            break
        else
            echo -e "${RED}❌ 无效的选择，请重新输入！${NC}"
        fi
    done
}

# 检测目标中存在的可排除项
detect_excludable_items() {
    local target=$1
    local found_items=()
    
    if [ -d "$target" ]; then
        # 检测常见的可排除目录/文件
        local check_dirs=("node_modules" "venv" ".venv" "__pycache__" ".git" ".idea" ".vscode" "dist" "build" "target" ".cache" "vendor" ".gradle" ".pytest_cache" ".mypy_cache")
        
        for dir in "${check_dirs[@]}"; do
            if [ -d "$target/$dir" ] || find "$target" -type d -name "$dir" -print -quit 2>/dev/null | grep -q .; then
                found_items+=("$dir")
            fi
        done
        
        # 检测 .DS_Store
        if find "$target" -name ".DS_Store" -print -quit 2>/dev/null | grep -q .; then
            found_items+=(".DS_Store")
        fi
        
        # 检测 *.pyc 文件
        if find "$target" -name "*.pyc" -print -quit 2>/dev/null | grep -q .; then
            found_items+=("*.pyc")
        fi
        
        # 检测 *.log 文件
        if find "$target" -name "*.log" -print -quit 2>/dev/null | grep -q .; then
            found_items+=("*.log")
        fi
    fi
    
    echo "${found_items[@]}"
}

# 选择排除模式
select_exclude_patterns() {
    local target=$1
    
    # 检测目标中存在哪些可排除项
    local detected_items
    detected_items=$(detect_excludable_items "$target")
    
    print_header
    echo -e "${YELLOW}🔧 排除文件设置${NC}"
    echo -e "${DIM}选择要从压缩包中排除的文件/目录类型${NC}\n"
    
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  [1]${NC} 🚫 ${WHITE}使用默认排除规则${NC} ${DIM}(推荐)${NC}"
    echo -e "      ${DIM}排除: node_modules, venv, __pycache__, .git, .idea, dist, build 等${NC}"
    echo ""
    echo -e "${BLUE}  [2]${NC} 📦 ${WHITE}不排除任何文件${NC} ${DIM}(完整压缩)${NC}"
    echo ""
    echo -e "${BLUE}  [3]${NC} ⚙️  ${WHITE}自定义排除规则${NC}"
    echo ""
    echo -e "${RED}  [0]${NC} 🚪 ${DIM}退出程序${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # 如果检测到了可排除项，显示提示
    if [ -n "$detected_items" ]; then
        echo ""
        echo -e "${YELLOW}💡 检测到目标中包含以下可排除项：${NC}"
        echo -e "   ${MAGENTA}$detected_items${NC}"
    fi
    
    echo ""
    while true; do
        echo -ne "${YELLOW}请选择排除模式 [0-3] (默认=1): ${NC}"
        read choice
        
        # 默认选择1
        if [ -z "$choice" ]; then
            choice=1
        fi
        
        case "$choice" in
            0|q|Q)
                echo -e "\n${GREEN}👋 感谢使用，再见！${NC}\n"
                exit 0
                ;;
            1)
                EXCLUDE_PATTERNS=("${DEFAULT_EXCLUDES[@]}")
                echo -e "${GREEN}✓ 将使用默认排除规则${NC}"
                break
                ;;
            2)
                EXCLUDE_PATTERNS=()
                echo -e "${GREEN}✓ 将压缩所有文件（不排除）${NC}"
                break
                ;;
            3)
                custom_exclude_selection
                break
                ;;
            *)
                echo -e "${RED}❌ 无效的选择，请重新输入！${NC}"
                ;;
        esac
    done
}

# 自定义排除选择
custom_exclude_selection() {
    EXCLUDE_PATTERNS=()
    
    print_header
    echo -e "${YELLOW}⚙️  自定义排除规则${NC}"
    echo -e "${DIM}输入序号切换排除状态，输入 'done' 完成选择${NC}\n"
    
    # 定义排除类别
    declare -A categories
    categories=(
        ["1,Python 相关"]="venv/* .venv/* __pycache__/* *.pyc *.pyo .pytest_cache/* .mypy_cache/* *.egg-info/* .eggs/*"
        ["2,Node.js 相关"]="node_modules/* .npm/* .pnpm-store/*"
        ["3,IDE/编辑器配置"]="'.idea/*' '.vscode/*' '*.swp' '*.swo' '*~'"
        ["4,Git 版本控制"]=".git/*"
        ["5,构建产物"]="dist/* build/* target/* out/*"
        ["6,系统文件"]=".DS_Store Thumbs.db desktop.ini"
        ["7,日志和缓存"]="*.log .cache/* .temp/* .tmp/*"
        ["8,Go 依赖"]="vendor/*"
        ["9,Java/Gradle 相关"]=".gradle/* .m2/*"
    )
    
    declare -A selected
    # 默认全选
    for key in "${!categories[@]}"; do
        selected["$key"]=1
    done
    
    while true; do
        print_header
        echo -e "${YELLOW}⚙️  自定义排除规则${NC}"
        echo -e "${DIM}输入序号切换排除状态，输入 'done' 或 'd' 完成选择，输入 '0' 退出${NC}\n"
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        
        for key in $(echo "${!categories[@]}" | tr ' ' '\n' | sort); do
            local num="${key%%,*}"
            local name="${key#*,}"
            local patterns="${categories[$key]}"
            
            if [ "${selected[$key]}" -eq 1 ]; then
                echo -e "${BLUE}  [$num]${NC} ${GREEN}✓${NC} ${WHITE}$name${NC}"
            else
                echo -e "${BLUE}  [$num]${NC} ${RED}✗${NC} ${DIM}$name${NC}"
            fi
            echo -e "      ${DIM}$patterns${NC}"
        done
        
        echo ""
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${DIM}  [a] 全选  [n] 全不选  [d/done] 完成  [0] 退出${NC}"
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        
        echo -ne "\n${YELLOW}请输入选项: ${NC}"
        read input
        
        case "$input" in
            0|q|Q)
                echo -e "\n${GREEN}👋 感谢使用，再见！${NC}\n"
                exit 0
                ;;
            done|d|D)
                break
                ;;
            a|A)
                for key in "${!categories[@]}"; do
                    selected["$key"]=1
                done
                ;;
            n|N)
                for key in "${!categories[@]}"; do
                    selected["$key"]=0
                done
                ;;
            [1-9])
                for key in "${!categories[@]}"; do
                    if [[ "$key" == "$input,"* ]]; then
                        if [ "${selected[$key]}" -eq 1 ]; then
                            selected["$key"]=0
                        else
                            selected["$key"]=1
                        fi
                        break
                    fi
                done
                ;;
            *)
                echo -e "${RED}❌ 无效输入${NC}"
                sleep 0.5
                ;;
        esac
    done
    
    # 构建最终的排除模式列表
    for key in "${!categories[@]}"; do
        if [ "${selected[$key]}" -eq 1 ]; then
            # 将空格分隔的模式添加到数组
            for pattern in ${categories[$key]}; do
                # 移除可能的引号
                pattern="${pattern//\'/}"
                EXCLUDE_PATTERNS+=("$pattern")
            done
        fi
    done
    
    echo -e "${GREEN}✓ 自定义排除规则已设置${NC}"
}

# 检查并处理重名
handle_duplicate() {
    local zip_name=$1
    
    if [ -f "$zip_name" ]; then
        local size=$(stat -f%z "$zip_name" 2>/dev/null || stat -c%s "$zip_name" 2>/dev/null)
        local mtime=$(stat -f%m "$zip_name" 2>/dev/null || stat -c%Y "$zip_name" 2>/dev/null)
        
        echo ""
        echo -e "${YELLOW}⚠️  检测到同名压缩包已存在！${NC}"
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${WHITE}文件名：${NC}${MAGENTA}$zip_name${NC}"
        echo -e "${WHITE}大  小：${NC}${GREEN}$(format_size $size)${NC}"
        echo -e "${WHITE}修改时间：${NC}${BLUE}$(format_time $mtime)${NC}"
        echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
        
        while true; do
            echo -ne "${YELLOW}是否替换现有文件？[y/N]: ${NC}"
            read -r response
            case "$response" in
                [yY][eE][sS]|[yY])
                    rm -f "$zip_name"
                    echo "$zip_name"
                    return
                    ;;
                [nN][oO]|[nN]|"")
                    local base="${zip_name%.zip}"
                    local suffix=$(generate_random_suffix)
                    local new_name="${base}_${suffix}.zip"
                    echo -e "${GREEN}✓ 新文件将命名为：${MAGENTA}$new_name${NC}"
                    echo "$new_name"
                    return
                    ;;
                *)
                    echo -e "${RED}❌ 请输入 y 或 n${NC}"
                    ;;
            esac
        done
    else
        echo "$zip_name"
    fi
}

# 构建排除参数
build_exclude_args() {
    local exclude_args=""
    for pattern in "${EXCLUDE_PATTERNS[@]}"; do
        exclude_args="$exclude_args -x '$pattern'"
    done
    echo "$exclude_args"
}

# 压缩文件
compress_with_progress() {
    local target=$1
    local output=$2
    
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}${BOLD}🚀 开始压缩...${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    # 显示排除信息
    if [ ${#EXCLUDE_PATTERNS[@]} -gt 0 ]; then
        echo -e "${YELLOW}🚫 排除模式数量：${WHITE}${#EXCLUDE_PATTERNS[@]}${NC}"
    else
        echo -e "${YELLOW}📦 完整压缩模式（不排除任何文件）${NC}"
    fi
    
    # 计算总文件数（考虑排除模式）
    local total_files
    if [ -d "$target" ]; then
        if [ ${#EXCLUDE_PATTERNS[@]} -gt 0 ]; then
            # 使用find并排除匹配的模式
            local find_excludes=""
            for pattern in "${EXCLUDE_PATTERNS[@]}"; do
                # 将zip排除模式转换为find的排除参数
                local clean_pattern="${pattern//\*/}"
                clean_pattern="${clean_pattern//\//}"
                if [ -n "$clean_pattern" ]; then
                    find_excludes="$find_excludes -not -path '*/$clean_pattern/*' -not -path '*/$clean_pattern' -not -name '$clean_pattern'"
                fi
            done
            # 简化：直接统计文件数（排除常见目录）
            total_files=$(find "$target" -type f \
                -not -path "*/node_modules/*" \
                -not -path "*/.git/*" \
                -not -path "*/venv/*" \
                -not -path "*/.venv/*" \
                -not -path "*/__pycache__/*" \
                -not -path "*/.idea/*" \
                -not -path "*/.vscode/*" \
                -not -path "*/dist/*" \
                -not -path "*/build/*" \
                -not -path "*/target/*" \
                -not -path "*/.cache/*" \
                -not -name "*.pyc" \
                -not -name ".DS_Store" \
                2>/dev/null | wc -l)
        else
            total_files=$(find "$target" -type f 2>/dev/null | wc -l)
        fi
    else
        total_files=1
    fi
    
    echo -e "${WHITE}📊 预计压缩文件数：${YELLOW}$total_files${NC}"
    echo ""
    
    # 隐藏光标
    echo -e "${HIDE_CURSOR}"
    
    # 保存进度条位置
    local progress_line=$(($(tput lines) - 10))
    
    # 创建临时文件用于存储zip输出
    TEMP_ZIP="${output}.tmp"
    local current_file=0
    local last_percentage=-1
    
    # 构建zip命令
    local zip_cmd
    if [ ${#EXCLUDE_PATTERNS[@]} -gt 0 ]; then
        # 构建排除参数数组
        local exclude_args=()
        for pattern in "${EXCLUDE_PATTERNS[@]}"; do
            exclude_args+=("-x" "$pattern")
        done
        
        # 使用zip命令并捕获输出（带排除）
        (
            if [ -d "$target" ]; then
                zip -r "$TEMP_ZIP" "$target" "${exclude_args[@]}" 2>&1
            else
                zip "$TEMP_ZIP" "$target" "${exclude_args[@]}" 2>&1
            fi
        ) | while IFS= read -r line; do
            if [[ $line =~ adding:\ (.+)\ \(.*\)$ ]]; then
                local file="${BASH_REMATCH[1]}"
                ((current_file++))
                
                # 计算百分比（防止除零）
                local percentage=0
                if [ $total_files -gt 0 ]; then
                    percentage=$((current_file * 100 / total_files))
                    if [ $percentage -gt 100 ]; then
                        percentage=100
                    fi
                fi
                
                # 只在百分比变化时更新进度条
                if [ $percentage -ne $last_percentage ]; then
                    last_percentage=$percentage
                    
                    # 绘制进度条
                    local bar_width=50
                    local filled=$((percentage * bar_width / 100))
                    local empty=$((bar_width - filled))
                    
                    # 保存当前位置，移动到进度条位置
                    tput sc
                    tput cup $progress_line 0
                    
                    # 清空进度条区域
                    echo -e "${CLEAR_LINE}"
                    
                    # 绘制进度条
                    echo -ne "${WHITE}进度: [${GREEN}"
                    printf '%*s' "$filled" '' | tr ' ' '█'
                    echo -ne "${DIM}"
                    printf '%*s' "$empty" '' | tr ' ' '░'
                    echo -ne "${NC}${WHITE}] ${YELLOW}${percentage}%${NC} ${CYAN}(${current_file})${NC}"
                    
                    # 恢复光标位置
                    tput rc
                fi
                
                # 显示当前文件（限制长度）
                local display_file="$file"
                if [ ${#display_file} -gt 60 ]; then
                    display_file="...${display_file: -57}"
                fi
                echo -e "${CLEAR_LINE}${DIM}${CYAN}📄 正在压缩:${NC} ${WHITE}$display_file${NC}"
            fi
        done
    else
        # 使用zip命令并捕获输出（不排除）
        (
            if [ -d "$target" ]; then
                zip -r "$TEMP_ZIP" "$target" 2>&1
            else
                zip "$TEMP_ZIP" "$target" 2>&1
            fi
        ) | while IFS= read -r line; do
            if [[ $line =~ adding:\ (.+)\ \(.*\)$ ]]; then
                local file="${BASH_REMATCH[1]}"
                ((current_file++))
                
                # 计算百分比
                local percentage=$((current_file * 100 / total_files))
                
                # 只在百分比变化时更新进度条
                if [ $percentage -ne $last_percentage ]; then
                    last_percentage=$percentage
                    
                    # 绘制进度条
                    local bar_width=50
                    local filled=$((percentage * bar_width / 100))
                    local empty=$((bar_width - filled))
                    
                    # 保存当前位置，移动到进度条位置
                    tput sc
                    tput cup $progress_line 0
                    
                    # 清空进度条区域
                    echo -e "${CLEAR_LINE}"
                    
                    # 绘制进度条
                    echo -ne "${WHITE}进度: [${GREEN}"
                    printf '%*s' "$filled" '' | tr ' ' '█'
                    echo -ne "${DIM}"
                    printf '%*s' "$empty" '' | tr ' ' '░'
                    echo -ne "${NC}${WHITE}] ${YELLOW}${percentage}%${NC} ${CYAN}(${current_file}/${total_files})${NC}"
                    
                    # 恢复光标位置
                    tput rc
                fi
                
                # 显示当前文件（限制长度）
                local display_file="$file"
                if [ ${#display_file} -gt 60 ]; then
                    display_file="...${display_file: -57}"
                fi
                echo -e "${CLEAR_LINE}${DIM}${CYAN}📄 正在压缩:${NC} ${WHITE}$display_file${NC}"
            fi
        done
    fi
    
    # 确保进度条显示100%
    tput cup $progress_line 0
    echo -e "${CLEAR_LINE}"
    echo -e "${WHITE}进度: [${GREEN}$(printf '%*s' 50 '' | tr ' ' '█')${NC}${WHITE}] ${YELLOW}100%${NC} ${GREEN}完成${NC}"
    
    # 移动临时文件到最终位置
    if [ -f "$TEMP_ZIP" ]; then
        mv "$TEMP_ZIP" "$output"
    fi
    
    # 显示光标
    echo -e "${SHOW_CURSOR}"
    
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    if [ -f "$output" ]; then
        local final_size=$(stat -f%z "$output" 2>/dev/null || stat -c%s "$output" 2>/dev/null)
        # 获取压缩包内的实际文件数
        local actual_files=$(unzip -l "$output" 2>/dev/null | tail -1 | awk '{print $2}')
        
        echo -e "${GREEN}${BOLD}✅ 压缩完成！${NC}"
        echo ""
        echo -e "${WHITE}📦 输出文件：${NC}${MAGENTA}$output${NC}"
        echo -e "${WHITE}📊 文件大小：${NC}${GREEN}$(format_size $final_size)${NC}"
        echo -e "${WHITE}✨ 压缩文件数：${NC}${YELLOW}${actual_files:-N/A}${NC}"
        if [ ${#EXCLUDE_PATTERNS[@]} -gt 0 ]; then
            echo -e "${WHITE}🚫 已排除模式：${NC}${DIM}${#EXCLUDE_PATTERNS[@]} 个${NC}"
        fi
    else
        echo -e "${RED}❌ 压缩失败！${NC}"
        exit 1
    fi
    
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# 主程序
main() {
    # 检查zip命令
    if ! command -v zip &> /dev/null; then
        echo -e "${RED}❌ 错误：未找到 zip 命令，请先安装！${NC}"
        exit 1
    fi
    
    # 选择目标
    select_target
    
    # 如果是目录，询问排除选项
    if [ -d "$TARGET" ]; then
        select_exclude_patterns "$TARGET"
    else
        # 单个文件不需要排除
        EXCLUDE_PATTERNS=()
    fi
    
    # 生成压缩包名称
    local zip_name="${TARGET}.zip"
    
    # 处理重名
    zip_name=$(handle_duplicate "$zip_name")
    
    # 执行压缩
    compress_with_progress "$TARGET" "$zip_name"
    
    echo ""
    echo -e "${GREEN}${BOLD}🎉 所有操作完成！${NC}"
    echo ""
}

# 运行主程序
main
