package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/opsinit"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	var pins opsinit.O4ArtifactPins
	flag.StringVar(&pins.DescriptorPath, "descriptor", "", "reviewed descriptor")
	flag.StringVar(&pins.DescriptorSHA256, "descriptor-sha256", "", "trusted descriptor hash")
	flag.StringVar(&pins.BuildReceiptPath, "build-receipt", "", "sealed build receipt")
	flag.StringVar(&pins.BuildReceiptSHA256, "build-receipt-sha256", "", "trusted build receipt hash")
	flag.StringVar(&pins.ExecutablePath, "executable", "", "exact executable")
	flag.StringVar(&pins.ExecutableSHA256, "executable-sha256", "", "trusted executable hash")
	receiptPath := flag.String("receipt", "", "exclusive new start receipt")
	checkOnly := flag.Bool("check-only", false, "verify without launching")
	flag.Parse()
	fail := func() { _, _ = os.Stderr.WriteString("observability.supported_start_refused\n"); os.Exit(1) }
	dsn := os.Getenv("PANTHEON_DSN")
	if dsn == "" || *receiptPath == "" {
		fail()
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fail()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	receipt, err := opsinit.CheckO4SupportedStart(ctx, db, pins, opsinit.ScanO4ManagedProcesses)
	if err != nil {
		fail()
	}
	file, err := os.OpenFile(*receiptPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		fail()
	}
	if json.NewEncoder(file).Encode(receipt) != nil {
		_ = file.Close()
		fail()
	}
	if file.Close() != nil {
		fail()
	}
	if *checkOnly {
		return
	}
	// The child uses the exact privately inspected DSN via inherited environment.
	command := exec.Command(pins.ExecutablePath, flag.Args()...)
	command.Env = os.Environ()
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		fail()
	}
}
