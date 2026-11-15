package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/observability"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/opsinit"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	ownershipPath := flag.String("exclusive-ownership", "", "test-owned stopped-writer receipt")
	expected := flag.String("ownership-sha256", "", "trusted receipt digest")
	receiptPath := flag.String("receipt", "", "exclusive new recovery receipt")
	quiescencePath := flag.String("quiescence-receipt", "", "verified zero-managed-writer receipt")
	flag.Parse()
	fail := func() { _, _ = os.Stderr.WriteString("observability.schema_unavailable\n"); os.Exit(1) }
	bytes, err := os.ReadFile(*ownershipPath)
	sum := sha256.Sum256(bytes)
	if err != nil || len(bytes) > 64<<10 || hex.EncodeToString(sum[:]) != *expected || *receiptPath == "" {
		fail()
	}
	var ownership observability.O4RecoveryOwnership
	if json.Unmarshal(bytes, &ownership) != nil {
		fail()
	}
	dsn := os.Getenv("PANTHEON_DSN")
	if dsn == "" {
		fail()
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fail()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := opsinit.CheckO4RecoveryQuiescence(ctx, ownership, *quiescencePath); err != nil {
		fail()
	}
	receipt, err := observability.RecoverO4Migration37(ctx, db, ownership)
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
}
