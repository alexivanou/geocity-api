package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/alexivanou/geocity-api/internal/config"
	"github.com/alexivanou/geocity-api/internal/database"
	"github.com/alexivanou/geocity-api/internal/stats"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	db, err := database.Connect(context.Background(), cfg.DB)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}

	logger.Info("Collecting statistics...", zap.String("db_type", string(cfg.DB.Type)))

	collector := stats.NewCollector(db, cfg.DB)

	ctx := context.Background()
	statistics, err := collector.Collect(ctx)
	if err != nil {
		logger.Fatal("Failed to collect statistics", zap.Error(err))
	}

	outputFormat := os.Getenv("OUTPUT_FORMAT")
	if outputFormat == "" {
		outputFormat = "json"
	}

	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(statistics); err != nil {
			logger.Fatal("Failed to encode statistics", zap.Error(err))
		}
	case "text", "human":
		printHumanReadable(statistics)
	default:
		logger.Fatal("Unknown output format", zap.String("format", outputFormat))
	}
}

func printHumanReadable(s *stats.Stats) {
	fmt.Println("=== Application Statistics ===")
	fmt.Printf("Timestamp: %s\n", s.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Println()

	fmt.Println("--- Memory Statistics ---")
	fmt.Printf("Allocated:        %s\n", formatBytes(s.Memory.Alloc))
	fmt.Printf("Total Allocated:  %s\n", formatBytes(s.Memory.TotalAlloc))
	fmt.Println()

	fmt.Println("--- Database Statistics ---")
	fmt.Printf("Type:            %s\n", s.Database.Type)
	fmt.Printf("Total Records:   %d\n", s.Database.TotalRecords)
	fmt.Println()
	fmt.Println("Table Statistics:")
	for _, ts := range s.Database.TableStats {
		fmt.Printf("  %-25s: %10d rows", ts.Name, ts.RowCount)
		if ts.SizeBytes > 0 {
			fmt.Printf(" (%s)", formatBytes(uint64(ts.SizeBytes)))
		}
		fmt.Println()
	}
	fmt.Println()

	fmt.Println("--- Runtime Statistics ---")
	fmt.Printf("Goroutines:      %d\n", s.Runtime.NumGoroutines)
	fmt.Printf("Uptime:          %ds\n", s.Runtime.UptimeSeconds)
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
