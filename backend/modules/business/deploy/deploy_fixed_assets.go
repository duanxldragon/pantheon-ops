package deploy

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// fixedTemplateAssetDirEnv lets CI point the check at a packaged asset folder.
const fixedTemplateAssetDirEnv = "PANTHEON_DEPLOY_ASSET_DIR"

// FixedTemplateAsset records which preset archive a fixed template consumes and
// whether the repository ships one under files/amd64.
type FixedTemplateAsset struct {
	TemplateCode    string
	AssetPrefixes   []string
	NeedsOwnPackage bool
}

// FixedTemplateAssets is the single machine-readable catalog behind the
// install-asset consistency check. A fixed template either names the preset
// archive prefixes it consumes or is explicitly marked as needing an
// operator-supplied package; a new template must do one of the two or
// TestFixedTemplateAssetCoverage fails.
func FixedTemplateAssets() []FixedTemplateAsset {
	return []FixedTemplateAsset{
		{TemplateCode: TemplateCodeNginxSystemd, AssetPrefixes: []string{"nginx-"}},
		{TemplateCode: TemplateCodeRedisSystemd, AssetPrefixes: []string{"redis-"}},
		{TemplateCode: TemplateCodeJDKSystemd, AssetPrefixes: []string{"jdk-", "bisheng-jdk-"}},
		{TemplateCode: TemplateCodePrometheusSystemd, AssetPrefixes: []string{"prometheus-"}},
		{TemplateCode: TemplateCodeMySQLSystemd, NeedsOwnPackage: true},
		{TemplateCode: TemplateCodeMinIOSystemd, NeedsOwnPackage: true},
		{TemplateCode: TemplateCodeHarborOffline, NeedsOwnPackage: true},
	}
}

// FixedTemplateAssetDir resolves the preset package directory. The default is
// the repository copy; PANTHEON_DEPLOY_ASSET_DIR overrides it for packaging
// and CI layouts that keep files/ elsewhere.
func FixedTemplateAssetDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv(fixedTemplateAssetDirEnv)); custom != "" {
		return custom, nil
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("deploy.fixedAsset.callerdepth")
	}
	// backend/modules/business/deploy -> repository root
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "files", "amd64"), nil
}

// UnbackedFixedTemplates lists templates that declare a preset package no
// archive in the asset directory satisfies. Templates explicitly marked as
// needing an operator-supplied package are never reported.
func UnbackedFixedTemplates(catalog []FixedTemplateAsset, assetNames []string) []string {
	missing := make([]string, 0, len(catalog))
	for _, item := range catalog {
		if item.NeedsOwnPackage || len(item.AssetPrefixes) == 0 {
			continue
		}
		backed := false
		for _, name := range assetNames {
			for _, prefix := range item.AssetPrefixes {
				if strings.HasPrefix(name, prefix) {
					backed = true
					break
				}
			}
			if backed {
				break
			}
		}
		if !backed {
			missing = append(missing, item.TemplateCode)
		}
	}
	sort.Strings(missing)
	return missing
}

func listFixedAssetNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}
