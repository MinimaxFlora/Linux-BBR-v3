package app

// 版本信息（构建时可通过 ldflags 注入，见 release-cli.yml）。
var (
	// Version 程序版本。
	Version = "v1.0.0"
	// Commit git commit SHA（-X ...app.Commit=...）。
	Commit = "dev"
	// kernelVersion 运行环境的内核版本（main 注入，用于界面展示）。
	kernelVersion = ""
)

// SetKernelVersion 注入当前内核版本（main 调用）。
func SetKernelVersion(k string) { kernelVersion = k }
