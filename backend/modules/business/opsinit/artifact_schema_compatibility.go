package opsinit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/observability"
	"gorm.io/gorm"
)

var ErrO4Start = errors.New("observability.supported_start_refused")

type O4BuildReceipt struct {
	Kind                     string    `json:"kind"`
	DescriptorSHA256         string    `json:"descriptorSha256"`
	CandidateIdentity        string    `json:"candidateIdentity"`
	ExecutableSHA256         string    `json:"executableSha256"`
	Capabilities             []string  `json:"capabilities"`
	ManagedProcessNames      []string  `json:"managedProcessNames"`
	KnownOldExecutableSHA256 []string  `json:"knownOldExecutableSha256"`
	BuiltAt                  time.Time `json:"builtAt"`
}
type O4ArtifactPins struct {
	DescriptorPath     string
	DescriptorSHA256   string
	BuildReceiptPath   string
	BuildReceiptSHA256 string
	ExecutablePath     string
	ExecutableSHA256   string
}
type O4Process struct {
	PID            uint32 `json:"pid"`
	Name           string `json:"name"`
	ExecutablePath string `json:"executablePath"`
}
type O4StartReceipt struct {
	Kind                string    `json:"kind"`
	Database            string    `json:"database"`
	SchemaState         string    `json:"schemaState"`
	DescriptorSHA256    string    `json:"descriptorSha256"`
	BuildReceiptSHA256  string    `json:"buildReceiptSha256"`
	ExecutableSHA256    string    `json:"executableSha256"`
	ZeroOldProcesses    bool      `json:"zeroOldProcesses"`
	ManagedProcesses    []uint32  `json:"managedProcesses"`
	ManagedProcessNames []string  `json:"managedProcessNames"`
	CheckedAt           time.Time `json:"checkedAt"`
}

func o4StableBytes(path, expected string, maxBytes int64) ([]byte, error) {
	if len(expected) != 64 {
		return nil, ErrO4Start
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, ErrO4Start
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() || before.Size() > maxBytes {
		return nil, ErrO4Start
	}
	bytes, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil || int64(len(bytes)) > maxBytes {
		return nil, ErrO4Start
	}
	after, err := file.Stat()
	if err != nil || before.Size() != after.Size() || before.ModTime() != after.ModTime() {
		return nil, ErrO4Start
	}
	sum := sha256.Sum256(bytes)
	if hex.EncodeToString(sum[:]) != expected {
		return nil, ErrO4Start
	}
	return bytes, nil
}
func o4FileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", ErrO4Start
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil || !before.Mode().IsRegular() {
		return "", ErrO4Start
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", ErrO4Start
	}
	after, err := file.Stat()
	if err != nil || before.Size() != after.Size() || before.ModTime() != after.ModTime() {
		return "", ErrO4Start
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
func validateO4Artifact(pins O4ArtifactPins) (O4BuildReceipt, bool, error) {
	var receipt O4BuildReceipt
	descriptorBytes, err := o4StableBytes(pins.DescriptorPath, pins.DescriptorSHA256, 8<<20)
	if err != nil {
		return receipt, false, err
	}
	var descriptor struct {
		Kind              string `json:"kind"`
		CandidateIdentity string `json:"candidateIdentity"`
	}
	if json.Unmarshal(descriptorBytes, &descriptor) != nil || descriptor.Kind != "pantheon-base-candidate" || descriptor.CandidateIdentity == "" {
		return receipt, false, ErrO4Start
	}
	receiptBytes, err := o4StableBytes(pins.BuildReceiptPath, pins.BuildReceiptSHA256, 64<<10)
	if err != nil {
		return receipt, false, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(receiptBytes)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&receipt) != nil || receipt.Kind != "pantheon-o4-build-v1" || receipt.DescriptorSHA256 != pins.DescriptorSHA256 || receipt.CandidateIdentity != descriptor.CandidateIdentity ||
		receipt.ExecutableSHA256 != pins.ExecutableSHA256 || len(receipt.ManagedProcessNames) == 0 || len(receipt.ManagedProcessNames) > 16 || len(receipt.KnownOldExecutableSHA256) > 16 {
		return receipt, false, ErrO4Start
	}
	actual, err := o4FileSHA256(pins.ExecutablePath)
	if err != nil || actual != pins.ExecutableSHA256 {
		return receipt, false, ErrO4Start
	}
	capable := false
	for _, capability := range receipt.Capabilities {
		if capability != observability.QueryWorkspaceCapability || capable {
			return receipt, false, ErrO4Start
		}
		capable = true
	}
	for _, name := range receipt.ManagedProcessNames {
		if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, "/\\\r\n") {
			return receipt, false, ErrO4Start
		}
	}
	for _, hash := range receipt.KnownOldExecutableSHA256 {
		if len(hash) != 64 {
			return receipt, false, ErrO4Start
		}
	}
	return receipt, capable, nil
}
func ScanO4ManagedProcesses(ctx context.Context) ([]O4Process, error) {
	if runtime.GOOS != "windows" {
		return nil, ErrO4Start
	}
	// Read-only process inventory; no command lines, credentials or mutations.
	command := exec.CommandContext(ctx, "pwsh", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command",
		"@(Get-CimInstance Win32_Process | Select-Object @{n='pid';e={$_.ProcessId}}, @{n='name';e={$_.Name}}, @{n='executablePath';e={$_.ExecutablePath}}) | ConvertTo-Json -Compress")
	bytes, err := command.Output()
	if err != nil || len(bytes) > 2<<20 {
		return nil, ErrO4Start
	}
	var processes []O4Process
	if json.Unmarshal(bytes, &processes) != nil {
		return nil, ErrO4Start
	}
	return processes, nil
}

// CheckO4SupportedStart binds reviewed source -> build -> exact executable,
// inspects real DB shape/dirty state, and checks live managed process hashes.
// Unmanaged external launches are outside this supported entry's protection.
func CheckO4SupportedStart(ctx context.Context, db *gorm.DB, pins O4ArtifactPins, scan func(context.Context) ([]O4Process, error)) (O4StartReceipt, error) {
	var result O4StartReceipt
	build, capable, err := validateO4Artifact(pins)
	if err != nil {
		return result, err
	}
	state, err := observability.QueryWorkspaceSchemaState(db)
	if err != nil {
		return result, ErrO4Start
	}
	if !capable && state == "o4" {
		return result, ErrO4Start
	}
	if db.Migrator().HasTable("schema_migrations") {
		var version uint64
		if err := db.Raw("SELECT version FROM schema_migrations WHERE dirty=false").Scan(&version).Error; err != nil || (state != "o4" && version > 36) {
			return result, ErrO4Start
		}
	}
	if scan == nil {
		return result, ErrO4Start
	}
	processes, err := scan(ctx)
	if err != nil {
		return result, err
	}
	var managed []uint32
	for _, process := range processes {
		selected := false
		for _, name := range build.ManagedProcessNames {
			selected = selected || strings.EqualFold(process.Name, name)
		}
		if !selected {
			continue
		}
		hash, err := o4FileSHA256(process.ExecutablePath)
		if err != nil {
			return result, ErrO4Start
		}
		knownOld := false
		for _, oldHash := range build.KnownOldExecutableSHA256 {
			knownOld = knownOld || hash == oldHash
		}
		// Transition cannot coexist with any old or unknown managed process.
		if hash != build.ExecutableSHA256 || knownOld {
			return result, ErrO4Start
		}
		managed = append(managed, process.PID)
	}
	var databaseName string
	if err := db.Raw("SELECT DATABASE()").Scan(&databaseName).Error; err != nil || databaseName == "" {
		return result, ErrO4Start
	}
	sort.Slice(managed, func(i, j int) bool { return managed[i] < managed[j] })
	result = O4StartReceipt{Kind: "pantheon-o4-supported-start-v1", Database: databaseName, SchemaState: state, DescriptorSHA256: pins.DescriptorSHA256, BuildReceiptSHA256: pins.BuildReceiptSHA256,
		ExecutableSHA256: pins.ExecutableSHA256, ZeroOldProcesses: true, ManagedProcesses: managed, ManagedProcessNames: append([]string(nil), build.ManagedProcessNames...), CheckedAt: time.Now().UTC()}
	return result, nil
}

func CheckO4RecoveryQuiescence(ctx context.Context, ownership observability.O4RecoveryOwnership, path string) error {
	bytes, err := o4StableBytes(path, ownership.QuiescenceReceiptSHA256, 64<<10)
	if err != nil {
		return err
	}
	var receipt O4StartReceipt
	if json.Unmarshal(bytes, &receipt) != nil || receipt.Kind != "pantheon-o4-supported-start-v1" || receipt.Database != ownership.Database || !receipt.ZeroOldProcesses || len(receipt.ManagedProcesses) != 0 ||
		len(receipt.ManagedProcessNames) == 0 || time.Since(receipt.CheckedAt) < 0 || time.Since(receipt.CheckedAt) > 5*time.Minute {
		return ErrO4Start
	}
	processes, err := ScanO4ManagedProcesses(ctx)
	if err != nil {
		return err
	}
	for _, process := range processes {
		for _, name := range receipt.ManagedProcessNames {
			if strings.EqualFold(name, process.Name) {
				return ErrO4Start
			}
		}
	}
	return nil
}
