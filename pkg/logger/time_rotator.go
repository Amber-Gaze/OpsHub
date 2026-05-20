package logger

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// rotationSpec 定义一种轮转精度的参数。
type rotationSpec struct {
	timeFormat    string        // Go time 格式化字符串
	suffixLen     int           // 文件名后缀长度
	checkInterval time.Duration // 后台轮转检查间隔
}

// rotationConfigs 支持三种轮转精度。
var rotationConfigs = map[string]rotationSpec{
	"minute": {"2006-01-02_15_04", 16, 10 * time.Second},
	"hour":   {"2006-01-02_15", 13, 30 * time.Second},
	"day":    {"2006-01-02", 10, time.Minute},
}

// TimeRotator 按配置精度轮转日志文件。
//   - 文件名格式: {basename}.YYYY-MM-DD_HH (hour) / YYYY-MM-DD_HH_MM (minute) / YYYY-MM-DD (day)
//   - 软链接:     {basename} -> 当前正在写入的文件
//   - 压缩:       开启后轮转出的历史文件自动 gzip
//   - 清理:       支持按保留天数 (maxAge) 和最大文件数 (maxBackups) 清理
type TimeRotator struct {
	dir        string
	basename   string
	spec       rotationSpec
	maxAge     time.Duration // 保留时长（0 表示不按时间清理）
	maxBackups int           // 最多保留的文件数（0 表示不限制）
	compress   bool

	mu          sync.Mutex
	current     *os.File
	currentHour string
	stopCh      chan struct{}
}

// NewTimeRotator 创建 TimeRotator 并立即打开当前时间段的日志文件。
// rotation: "minute" | "hour" | "day"（默认 "hour"）
func NewTimeRotator(dir, basename, rotation string, maxAge time.Duration, maxBackups int, compress bool) (*TimeRotator, error) {
	spec, ok := rotationConfigs[rotation]
	if !ok {
		spec = rotationConfigs["hour"]
	}

	tr := &TimeRotator{
		dir:        dir,
		basename:   basename,
		spec:       spec,
		maxAge:     maxAge,
		maxBackups: maxBackups,
		compress:   compress,
		stopCh:     make(chan struct{}),
	}
	if err := tr.rotate(); err != nil {
		return nil, err
	}
	go tr.loop()
	return tr, nil
}

// Write implements io.Writer.
func (tr *TimeRotator) Write(p []byte) (n int, err error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return tr.current.Write(p)
}

// Sync 刷新当前日志文件到磁盘。
func (tr *TimeRotator) Sync() error {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if tr.current != nil {
		return tr.current.Sync()
	}
	return nil
}

// Close 关闭 TimeRotator，停止后台轮转协程。
func (tr *TimeRotator) Close() error {
	close(tr.stopCh)
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if tr.current != nil {
		return tr.current.Close()
	}
	return nil
}

// hourSuffix 返回当前时间按 spec.timeFormat 格式化的后缀。
func (tr *TimeRotator) hourSuffix(t time.Time) string {
	return t.Format(tr.spec.timeFormat)
}

// rotate 执行文件轮转。
// 先（不持锁）创建新文件 → 快速交换指针（持锁，~ns 级）→ 关闭旧文件/压缩/更新软链接（不持锁）。
func (tr *TimeRotator) rotate() error {
	now := time.Now()
	suffix := tr.hourSuffix(now)
	logFilePath := filepath.Join(tr.dir, tr.basename+"."+suffix)

	// 先创建新文件（不持锁，避免 IO 阻塞写入）
	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", logFilePath, err)
	}

	// 快速交换指针（持锁临界区）
	tr.mu.Lock()
	old := tr.current
	tr.current = f
	tr.currentHour = suffix
	tr.mu.Unlock()

	// 关闭旧文件（不持锁）
	if old != nil {
		oldName := old.Name()
		old.Close()

		// 异步压缩旧文件
		if tr.compress {
			go compressFile(oldName)
		}
	}

	// 更新软链接（不持锁）
	symlinkPath := filepath.Join(tr.dir, tr.basename)
	os.Remove(symlinkPath)
	if err := os.Symlink(tr.basename+"."+suffix, symlinkPath); err != nil {
		fmt.Fprintf(os.Stderr, "time_rotator: create symlink %s -> %s failed: %v\n",
			symlinkPath, tr.basename+"."+suffix, err)
	}

	// 清理旧文件（后台异步执行）
	go tr.cleanup()
	return nil
}

// loop 按 spec.checkInterval 定时检查是否需要轮转。
func (tr *TimeRotator) loop() {
	ticker := time.NewTicker(tr.spec.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			suffix := tr.hourSuffix(time.Now())
			tr.mu.Lock()
			need := suffix != tr.currentHour
			tr.mu.Unlock()
			if need {
				// rotate 内部自己加锁，这里不需要再包一层
				if err := tr.rotate(); err != nil {
					fmt.Fprintf(os.Stderr, "time_rotator: rotate failed: %v\n", err)
				}
			}
		case <-tr.stopCh:
			return
		}
	}
}

// listLogFiles 返回匹配的日志文件列表（包括 .gz 压缩文件），按名称降序排列。
func (tr *TimeRotator) listLogFiles() []string {
	pattern := filepath.Join(tr.dir, tr.basename+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}

	var files []string
	for _, m := range matches {
		stem := strings.TrimPrefix(filepath.Base(m), tr.basename+".")

		// 去掉 .gz 后缀再解析时间
		timeStr := strings.TrimSuffix(stem, ".gz")

		// 用 spec.timeFormat 解析，能成功的才是目标日志文件
		if _, err := time.Parse(tr.spec.timeFormat, timeStr); err == nil {
			files = append(files, m)
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return files
}

// cleanup 删除超出保留策略的旧日志文件（包括 .gz 文件）。
func (tr *TimeRotator) cleanup() {
	files := tr.listLogFiles()

	// 按 maxAge 清理
	if tr.maxAge > 0 {
		deadline := time.Now().Add(-tr.maxAge)
		for _, f := range files {
			stem := strings.TrimPrefix(filepath.Base(f), tr.basename+".")
			timeStr := strings.TrimSuffix(stem, ".gz")
			t, err := time.Parse(tr.spec.timeFormat, timeStr)
			if err != nil {
				continue
			}
			if t.Before(deadline) {
				os.Remove(f)
			}
		}
	}

	// 按 maxBackups 清理
	if tr.maxBackups > 0 {
		files = tr.listLogFiles()
		if len(files) > tr.maxBackups {
			for _, f := range files[tr.maxBackups:] {
				os.Remove(f)
			}
		}
	}
}

// compressFile 将 src 文件压缩为 src.gz，成功后删除原文件。
func compressFile(src string) {
	dst := src + ".gz"

	in, err := os.Open(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "time_rotator: compress open %s failed: %v\n", src, err)
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "time_rotator: compress create %s failed: %v\n", dst, err)
		return
	}

	gw := gzip.NewWriter(out)
	if _, err := io.Copy(gw, in); err != nil {
		gw.Close()
		out.Close()
		os.Remove(dst)
		fmt.Fprintf(os.Stderr, "time_rotator: compress write %s failed: %v\n", dst, err)
		return
	}
	gw.Close()
	out.Close()
	in.Close()

	if err := os.Remove(src); err != nil {
		fmt.Fprintf(os.Stderr, "time_rotator: compress remove original %s failed: %v\n", src, err)
	}
}
