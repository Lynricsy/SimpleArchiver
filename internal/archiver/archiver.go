// Package archiver 提供文件压缩和解压功能
package archiver

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/pgzip"
	"github.com/pierrec/lz4/v4"
	"github.com/ulikunitz/xz"
)

// ProgressCallback 进度回调函数类型
type ProgressCallback func(current, total int, currentFile string)

// CompressStats 压缩统计信息
type CompressStats struct {
	TotalFiles      int
	ProcessedFiles  int
	TotalSize       int64
	CompressedSize  int64
	ExcludedFiles   int
	CurrentFile     string
	CompressionRate float64
}

// CompressOptions 压缩选项
type CompressOptions struct {
	Source     string
	Output     string
	Format     string
	Excludes   []string
	OnProgress ProgressCallback
	OnStats    func(stats CompressStats)
}

// shouldExclude 检查文件是否应该被排除
func shouldExclude(path string, excludes []string) bool {
	name := filepath.Base(path)

	for _, pattern := range excludes {
		// 检查是否是通配符模式
		if strings.Contains(pattern, "*") {
			matched, _ := filepath.Match(pattern, name)
			if matched {
				return true
			}
		} else {
			// 精确匹配目录或文件名
			if name == pattern {
				return true
			}
			// 检查路径中是否包含排除的目录
			if strings.Contains(path, string(filepath.Separator)+pattern+string(filepath.Separator)) ||
				strings.HasSuffix(path, string(filepath.Separator)+pattern) {
				return true
			}
		}
	}
	return false
}

// collectFiles 收集需要压缩的文件
func collectFiles(source string, excludes []string) ([]string, int64, int, error) {
	var files []string
	var totalSize int64
	excludedCount := 0

	sourceInfo, err := os.Stat(source)
	if err != nil {
		return nil, 0, 0, err
	}

	// 如果是单个文件
	if !sourceInfo.IsDir() {
		return []string{source}, sourceInfo.Size(), 0, nil
	}

	err = filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径用于排除检查
		relPath, err := filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			relPath = path
		}

		// 检查是否应该排除
		if shouldExclude(relPath, excludes) || shouldExclude(path, excludes) {
			if d.IsDir() {
				excludedCount++
				return filepath.SkipDir
			}
			excludedCount++
			return nil
		}

		if !d.IsDir() {
			files = append(files, path)
			info, err := d.Info()
			if err == nil {
				totalSize += info.Size()
			}
		}
		return nil
	})

	return files, totalSize, excludedCount, err
}

// Compress 执行压缩操作
func Compress(ctx context.Context, opts CompressOptions) (*CompressStats, error) {
	stats := &CompressStats{}

	// 检查源文件/目录是否存在
	_, err := os.Stat(opts.Source)
	if err != nil {
		return nil, fmt.Errorf("源路径不存在: %w", err)
	}

	files, totalSize, excludedCount, err := collectFiles(opts.Source, opts.Excludes)
	if err != nil {
		return nil, fmt.Errorf("收集文件失败: %w", err)
	}

	stats.TotalFiles = len(files)
	stats.TotalSize = totalSize
	stats.ExcludedFiles = excludedCount

	if stats.TotalFiles == 0 {
		return nil, fmt.Errorf("没有可压缩的文件")
	}

	// 根据格式选择压缩方式
	switch opts.Format {
	case ".zip":
		err = compressZip(ctx, files, opts, stats)
	case ".tar.gz":
		err = compressTarGz(ctx, files, opts, stats)
	case ".tar.bz2":
		err = compressTarBz2(ctx, files, opts, stats)
	case ".tar.xz":
		err = compressTarXz(ctx, files, opts, stats)
	case ".tar.zst":
		err = compressTarZstd(ctx, files, opts, stats)
	case ".tar.lz4":
		err = compressTarLz4(ctx, files, opts, stats)
	default:
		return nil, fmt.Errorf("不支持的压缩格式: %s", opts.Format)
	}

	if err != nil {
		return nil, err
	}

	// 获取压缩后文件大小
	outInfo, err := os.Stat(opts.Output)
	if err == nil {
		stats.CompressedSize = outInfo.Size()
		if stats.TotalSize > 0 {
			stats.CompressionRate = float64(stats.TotalSize-stats.CompressedSize) / float64(stats.TotalSize) * 100
		}
	}

	return stats, nil
}

// compressZip 使用 ZIP 格式压缩
func compressZip(ctx context.Context, files []string, opts CompressOptions, stats *CompressStats) error {
	outFile, err := os.Create(opts.Output)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	baseDir := filepath.Dir(opts.Source)

	for i, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		relPath, err := filepath.Rel(baseDir, file)
		if err != nil {
			relPath = filepath.Base(file)
		}

		// 更新进度
		stats.ProcessedFiles = i + 1
		stats.CurrentFile = relPath
		if opts.OnProgress != nil {
			opts.OnProgress(i+1, len(files), relPath)
		}
		if opts.OnStats != nil {
			opts.OnStats(*stats)
		}

		// 添加文件到 zip
		err = addFileToZip(zipWriter, file, relPath)
		if err != nil {
			return fmt.Errorf("添加文件失败 %s: %w", relPath, err)
		}
	}

	return nil
}

// addFileToZip 添加文件到 zip 归档
func addFileToZip(zw *zip.Writer, filePath, archivePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = archivePath
	header.Method = zip.Deflate

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// compressTarGz 使用 TAR.GZ 格式压缩
func compressTarGz(ctx context.Context, files []string, opts CompressOptions, stats *CompressStats) error {
	outFile, err := os.Create(opts.Output)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outFile.Close()

	gzWriter := pgzip.NewWriter(outFile)
	defer gzWriter.Close()

	return compressTar(ctx, files, gzWriter, opts, stats)
}

// compressTarBz2 使用 TAR.BZ2 格式压缩
func compressTarBz2(ctx context.Context, files []string, opts CompressOptions, stats *CompressStats) error {
	outFile, err := os.Create(opts.Output)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outFile.Close()

	bz2Writer, err := bzip2.NewWriter(outFile, &bzip2.WriterConfig{Level: bzip2.DefaultCompression})
	if err != nil {
		return fmt.Errorf("创建 Bzip2 写入器失败: %w", err)
	}
	defer bz2Writer.Close()

	return compressTar(ctx, files, bz2Writer, opts, stats)
}

// compressTarXz 使用 TAR.XZ 格式压缩
func compressTarXz(ctx context.Context, files []string, opts CompressOptions, stats *CompressStats) error {
	outFile, err := os.Create(opts.Output)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outFile.Close()

	xzWriter, err := xz.NewWriter(outFile)
	if err != nil {
		return fmt.Errorf("创建 XZ 写入器失败: %w", err)
	}
	defer xzWriter.Close()

	return compressTar(ctx, files, xzWriter, opts, stats)
}

// compressTarZstd 使用 TAR.ZSTD 格式压缩
func compressTarZstd(ctx context.Context, files []string, opts CompressOptions, stats *CompressStats) error {
	outFile, err := os.Create(opts.Output)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outFile.Close()

	zstdWriter, err := zstd.NewWriter(outFile, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		return fmt.Errorf("创建 Zstd 写入器失败: %w", err)
	}
	defer zstdWriter.Close()

	return compressTar(ctx, files, zstdWriter, opts, stats)
}

// compressTarLz4 使用 TAR.LZ4 格式压缩
func compressTarLz4(ctx context.Context, files []string, opts CompressOptions, stats *CompressStats) error {
	outFile, err := os.Create(opts.Output)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outFile.Close()

	lz4Writer := lz4.NewWriter(outFile)
	defer lz4Writer.Close()

	return compressTar(ctx, files, lz4Writer, opts, stats)
}

// compressTar TAR 压缩通用函数
func compressTar(ctx context.Context, files []string, writer io.Writer, opts CompressOptions, stats *CompressStats) error {
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	baseDir := filepath.Dir(opts.Source)

	for i, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		relPath, err := filepath.Rel(baseDir, file)
		if err != nil {
			relPath = filepath.Base(file)
		}

		// 更新进度
		stats.ProcessedFiles = i + 1
		stats.CurrentFile = relPath
		if opts.OnProgress != nil {
			opts.OnProgress(i+1, len(files), relPath)
		}
		if opts.OnStats != nil {
			opts.OnStats(*stats)
		}

		// 添加文件到 tar
		err = addFileToTar(tarWriter, file, relPath)
		if err != nil {
			return fmt.Errorf("添加文件失败 %s: %w", relPath, err)
		}
	}

	return nil
}

// addFileToTar 添加文件到 tar 归档
func addFileToTar(tw *tar.Writer, filePath, archivePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}

	header.Name = archivePath

	err = tw.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(tw, file)
	return err
}

// ExtractStats 解压统计信息
type ExtractStats struct {
	TotalFiles     int
	ProcessedFiles int
	TotalSize      int64
	ExtractedSize  int64
	CurrentFile    string
}

// ExtractOptions 解压选项
type ExtractOptions struct {
	Source     string
	Output     string
	OnProgress ProgressCallback
	OnStats    func(stats ExtractStats)
}

// DetectArchiveFormat 检测归档格式
func DetectArchiveFormat(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return ".zip"
	case strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz"):
		return ".tar.gz"
	case strings.HasSuffix(lower, ".tar.bz2") || strings.HasSuffix(lower, ".tbz2"):
		return ".tar.bz2"
	case strings.HasSuffix(lower, ".tar.xz") || strings.HasSuffix(lower, ".txz"):
		return ".tar.xz"
	case strings.HasSuffix(lower, ".tar.zst") || strings.HasSuffix(lower, ".tzst"):
		return ".tar.zst"
	case strings.HasSuffix(lower, ".tar.lz4"):
		return ".tar.lz4"
	case strings.HasSuffix(lower, ".tar"):
		return ".tar"
	default:
		return ""
	}
}

// IsArchiveFile 检查是否是支持的归档文件
func IsArchiveFile(filename string) bool {
	return DetectArchiveFormat(filename) != ""
}

// Extract 执行解压操作
func Extract(ctx context.Context, opts ExtractOptions) (*ExtractStats, error) {
	stats := &ExtractStats{}

	// 检查源文件是否存在
	sourceInfo, err := os.Stat(opts.Source)
	if err != nil {
		return nil, fmt.Errorf("源文件不存在: %w", err)
	}
	stats.TotalSize = sourceInfo.Size()

	// 检测格式
	format := DetectArchiveFormat(opts.Source)
	if format == "" {
		return nil, fmt.Errorf("不支持的归档格式")
	}

	// 创建输出目录
	if err := os.MkdirAll(opts.Output, 0755); err != nil {
		return nil, fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 根据格式选择解压方式
	switch format {
	case ".zip":
		err = extractZip(ctx, opts, stats)
	case ".tar.gz":
		err = extractTarGz(ctx, opts, stats)
	case ".tar.bz2":
		err = extractTarBz2(ctx, opts, stats)
	case ".tar.xz":
		err = extractTarXz(ctx, opts, stats)
	case ".tar.zst":
		err = extractTarZstd(ctx, opts, stats)
	case ".tar.lz4":
		err = extractTarLz4(ctx, opts, stats)
	case ".tar":
		err = extractTar(ctx, opts, stats)
	default:
		return nil, fmt.Errorf("不支持的归档格式: %s", format)
	}

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// extractZip 解压 ZIP 文件
func extractZip(ctx context.Context, opts ExtractOptions, stats *ExtractStats) error {
	reader, err := zip.OpenReader(opts.Source)
	if err != nil {
		return fmt.Errorf("打开 ZIP 文件失败: %w", err)
	}
	defer reader.Close()

	stats.TotalFiles = len(reader.File)

	for i, file := range reader.File {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 更新进度
		stats.ProcessedFiles = i + 1
		stats.CurrentFile = file.Name
		if opts.OnProgress != nil {
			opts.OnProgress(i+1, len(reader.File), file.Name)
		}
		if opts.OnStats != nil {
			opts.OnStats(*stats)
		}

		// 构建目标路径
		targetPath := filepath.Join(opts.Output, file.Name)

		// 安全检查：防止路径遍历攻击
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(opts.Output)) {
			return fmt.Errorf("非法的文件路径: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, file.Mode()); err != nil {
				return fmt.Errorf("创建目录失败 %s: %w", file.Name, err)
			}
			continue
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("创建父目录失败: %w", err)
		}

		// 解压文件
		if err := extractZipFile(file, targetPath); err != nil {
			return fmt.Errorf("解压文件失败 %s: %w", file.Name, err)
		}

		stats.ExtractedSize += int64(file.UncompressedSize64)
	}

	return nil
}

// extractZipFile 解压单个 ZIP 文件
func extractZipFile(file *zip.File, targetPath string) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	writer, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	return err
}

// extractTarGz 解压 TAR.GZ 文件
func extractTarGz(ctx context.Context, opts ExtractOptions, stats *ExtractStats) error {
	file, err := os.Open(opts.Source)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	gzReader, err := pgzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("创建 Gzip 读取器失败: %w", err)
	}
	defer gzReader.Close()

	return extractTarReader(ctx, gzReader, opts, stats)
}

// extractTarBz2 解压 TAR.BZ2 文件
func extractTarBz2(ctx context.Context, opts ExtractOptions, stats *ExtractStats) error {
	file, err := os.Open(opts.Source)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	bz2Reader, err := bzip2.NewReader(file, nil)
	if err != nil {
		return fmt.Errorf("创建 Bzip2 读取器失败: %w", err)
	}
	defer bz2Reader.Close()

	return extractTarReader(ctx, bz2Reader, opts, stats)
}

// extractTarXz 解压 TAR.XZ 文件
func extractTarXz(ctx context.Context, opts ExtractOptions, stats *ExtractStats) error {
	file, err := os.Open(opts.Source)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	xzReader, err := xz.NewReader(file)
	if err != nil {
		return fmt.Errorf("创建 XZ 读取器失败: %w", err)
	}

	return extractTarReader(ctx, xzReader, opts, stats)
}

// extractTarZstd 解压 TAR.ZSTD 文件
func extractTarZstd(ctx context.Context, opts ExtractOptions, stats *ExtractStats) error {
	file, err := os.Open(opts.Source)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	zstdReader, err := zstd.NewReader(file)
	if err != nil {
		return fmt.Errorf("创建 Zstd 读取器失败: %w", err)
	}
	defer zstdReader.Close()

	return extractTarReader(ctx, zstdReader, opts, stats)
}

// extractTarLz4 解压 TAR.LZ4 文件
func extractTarLz4(ctx context.Context, opts ExtractOptions, stats *ExtractStats) error {
	file, err := os.Open(opts.Source)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	lz4Reader := lz4.NewReader(file)

	return extractTarReader(ctx, lz4Reader, opts, stats)
}

// extractTar 解压 TAR 文件
func extractTar(ctx context.Context, opts ExtractOptions, stats *ExtractStats) error {
	file, err := os.Open(opts.Source)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	return extractTarReader(ctx, bufio.NewReader(file), opts, stats)
}

// extractTarReader TAR 解压通用函数
func extractTarReader(ctx context.Context, reader io.Reader, opts ExtractOptions, stats *ExtractStats) error {
	tarReader := tar.NewReader(reader)
	fileCount := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取 TAR 头失败: %w", err)
		}

		fileCount++
		stats.ProcessedFiles = fileCount
		stats.CurrentFile = header.Name
		if opts.OnProgress != nil {
			opts.OnProgress(fileCount, 0, header.Name) // TAR 不知道总文件数
		}
		if opts.OnStats != nil {
			opts.OnStats(*stats)
		}

		// 构建目标路径
		targetPath := filepath.Join(opts.Output, header.Name)

		// 安全检查：防止路径遍历攻击
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(opts.Output)) {
			return fmt.Errorf("非法的文件路径: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("创建目录失败 %s: %w", header.Name, err)
			}

		case tar.TypeReg:
			// 确保父目录存在
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("创建父目录失败: %w", err)
			}

			// 写入文件
			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("创建文件失败 %s: %w", header.Name, err)
			}

			written, err := io.Copy(outFile, tarReader)
			outFile.Close()
			if err != nil {
				return fmt.Errorf("写入文件失败 %s: %w", header.Name, err)
			}

			stats.ExtractedSize += written

		case tar.TypeSymlink:
			// 创建符号链接
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("创建父目录失败: %w", err)
			}
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				// 忽略符号链接错误（Windows 可能不支持）
				continue
			}
		}
	}

	stats.TotalFiles = fileCount
	return nil
}
