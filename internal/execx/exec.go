// Package execx 提供命令执行封装：流式输出转发到 Logger、上下文取消、
// 静默检查、合并输出捕获。TUI 层通过实现 Logger 把每行输出推送到界面。
package execx

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Logger 接收命令执行过程中的文本输出。
type Logger interface {
	Log(line string)
	Logf(format string, args ...any)
}

// Options 控制命令执行行为。
type Options struct {
	// Env 附加环境变量（KEY=VALUE）。
	Env []string
	// Dir 设置工作目录。
	Dir string
	// Stdin 可选标准输入。
	Stdin io.Reader
}

// environ 返回当前进程环境（Windows 上 exec 的 Env 处理不同，统一走 os.Environ）。
func environ() []string { return os.Environ() }

// Run 执行命令，stdout/stderr 逐行转发到 log（stderr 行加 [stderr] 前缀），
// 返回合并输出；命令非零退出时返回 *ExitError 包装的错误。
func Run(ctx context.Context, log Logger, name string, args ...string) (string, error) {
	return RunOpt(ctx, log, Options{}, name, args...)
}

// RunOpt 同 Run，带自定义选项。
func RunOpt(ctx context.Context, log Logger, opts Options, name string, args ...string) (string, error) {
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = opts.Dir
	cmd.Env = append(environ(), opts.Env...)
	cmd.Stdin = opts.Stdin

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		return "", err
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			buf.WriteString(line)
			buf.WriteByte('\n')
			if log != nil {
				log.Log(line)
			}
		}
	}()

	err := cmd.Wait()
	pw.Close()
	<-done

	out := strings.TrimSuffix(buf.String(), "\n")
	if err != nil {
		return out, err
	}
	return out, nil
}

// Output 静默执行命令（不输出日志），返回 stdout。
func Output(ctx context.Context, name string, args ...string) (string, error) {
	return RunOpt(ctx, nil, Options{}, name, args...)
}

// TryOutput 静默执行命令，任何错误返回空字符串。
func TryOutput(ctx context.Context, name string, args ...string) string {
	out, _ := Output(ctx, name, args...)
	return out
}

// RunOK 静默执行命令，仅返回是否成功（退出码 0）。
func RunOK(ctx context.Context, name string, args ...string) bool {
	err := exec.CommandContext(ctx, name, args...).Run()
	return err == nil
}

// HasCommand 判断命令是否存在于 PATH（等价 shell 的 command -v，但直接用 LookPath，
// 避免 "command" 是 shell 内建、没有对应可执行文件的问题）。
func HasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Which 返回命令的绝对路径，不存在返回空字符串。
func Which(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

// ExitError 是 exec.ExitError 的别名判断辅助。
func IsExitError(err error) bool {
	var ee *exec.ExitError
	return errors.As(err, &ee)
}
