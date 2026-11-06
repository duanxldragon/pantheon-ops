package dynamicmodule

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 生成包的编译校验（AUD-001）。
//
// 为什么需要它：注册前置原本只校验"生成目录存在"，于是 dynamicmodule 会把 registration
// 推进到 pending_activation，而重启后 `go build` 才失败——用户看到"已接入"，服务起不来。
// 校验必须发生在写盘之后、registration 状态推进之前，且只编译生成包本身，
// 避免把全仓历史构建债务带进注册路径。

const (
	// goBinaryEnvKey 与 scaffold 的 PANTHEON_NODE_BIN 同形：部署环境不保证 PATH 里有 go。
	goBinaryEnvKey = "PANTHEON_GO_BIN"

	// 首次编译生成包需要构建其依赖，给一个明确上限而不是无限等待。
	generatedModuleBuildTimeout = 180 * time.Second

	// 编译失败与"工具链缺失"共用同一个用户可见错误：
	// 两种情况对调用方的结论相同——这份生成源码现在不能接入。
	msgSourceUncompilable = "module.register.source_uncompilable"
)

// generatedModuleBuilder 编译工作区内的一个生成包。
//
// 抽成函数类型是为了给测试留缝：单元测试的临时工作区没有真实 Go 模块，
// 用它替换真实编译，才能覆盖"编译失败 → 拒绝注册"这条负向路径。
type generatedModuleBuilder func(workspaceRoot, packagePath string) error

// runGoBuildGeneratedModule 在 <workspace>/backend 下执行 `go build ./modules/<scope>/<name>`。
//
// packagePath 只指向生成包本身，因此退出码只反映这份生成代码是否可编译。
func runGoBuildGeneratedModule(workspaceRoot, packagePath string) error {
	goBinary, err := resolveGoBinary()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), generatedModuleBuildTimeout)
	defer cancel()

	// goBinary comes from PANTHEON_GO_BIN (absolute path enforced by resolveGoBinary)
	// or exec.LookPath; packagePath is the generated module import path produced by
	// the naming gate, not request input. Neither is attacker-controlled.
	//nolint:gosec // G204: both arguments are validated before reaching exec.
	cmd := exec.CommandContext(ctx, goBinary, "build", packagePath)
	cmd.Dir = filepath.Join(workspaceRoot, "backend")
	output, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail == "" {
			detail = err.Error()
		}
		return fmt.Errorf("%s: %w: %s", msgSourceUncompilable, err, detail)
	}
	return nil
}

func resolveGoBinary() (string, error) {
	if configured := strings.TrimSpace(os.Getenv(goBinaryEnvKey)); configured != "" {
		if !filepath.IsAbs(configured) {
			// 不新增 i18n 键：这是运维配置错误，走通用失败文案 + 服务端日志即可
			// （与 scaffold.resolveNodeBinary 的 module.generate.invalid_node_binary 不同，
			// 那个键从未登记进语言包，属既有幽灵键，不在本包复制）。
			return "", fmt.Errorf("%s: %s must be an absolute path, got %q", msgSourceUncompilable, goBinaryEnvKey, configured)
		}
		return configured, nil
	}
	resolved, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("%s: go toolchain not found in PATH: %w", msgSourceUncompilable, err)
	}
	return resolved, nil
}
