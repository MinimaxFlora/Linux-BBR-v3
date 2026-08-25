package system

import (
	"context"
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

// removeFile 删除文件，不存在不报错。
func removeFile(ctx context.Context, path string) error {
	return os.Remove(path)
}

// ensureFileExists 确保文件存在（touch）。
func ensureFileExists(ctx context.Context, path string) {
	_ = execx.RunOK(ctx, "touch", path)
}
