package system

import (
	"context"
	"io"
	"os"

	"github.com/MinimaxFlora/Linux-BBR-v3/internal/execx"
)

// appendFile 以追加模式写入文件（root 运行，直接写）。
func appendFile(ctx context.Context, log execx.Logger, path, content string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}

// writeFile 覆盖写入文件。
func writeFile(ctx context.Context, path, content string, mode os.FileMode) error {
	return os.WriteFile(path, []byte(content), mode)
}

// copyFile 复制 src 到 dst：先写临时文件再 rename 原子替换，
// 目标正在运行（如 /usr/local/bin/bbr）时也安全。
func copyFile(ctx context.Context, src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// removeFile 删除文件，不存在不报错。
func removeFile(ctx context.Context, path string) error {
	return os.Remove(path)
}

// ensureFileExists 确保文件存在（touch）。
func ensureFileExists(ctx context.Context, path string) {
	_ = execx.RunOK(ctx, "touch", path)
}
