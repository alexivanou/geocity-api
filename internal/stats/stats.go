package stats

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/alexivanou/geocity-api/internal/config"
	"github.com/jmoiron/sqlx"
)

type Stats struct {
	Timestamp time.Time     `json:"timestamp"`
	Memory    MemoryStats   `json:"memory"`
	Database  DatabaseStats `json:"database"`
	Runtime   RuntimeStats  `json:"runtime"`
}

type MemoryStats struct {
	Alloc        uint64 `json:"alloc"`
	TotalAlloc   uint64 `json:"total_alloc"`
	Sys          uint64 `json:"sys"`
	NumGC        uint32 `json:"num_gc"`
	HeapAlloc    uint64 `json:"heap_alloc"`
	HeapSys      uint64 `json:"heap_sys"`
	HeapInuse    uint64 `json:"heap_inuse"`
	HeapReleased uint64 `json:"heap_released"`
}

type DatabaseStats struct {
	Type               string      `json:"type"`
	TotalRecords       int64       `json:"total_records"`
	SizeBytes          int64       `json:"size_bytes"`
	TableStats         []TableStat `json:"table_stats"`
	AvailableLanguages int         `json:"available_languages"`
}

type TableStat struct {
	Name      string `json:"name"`
	RowCount  int64  `json:"row_count"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

type RuntimeStats struct {
	NumGoroutines int   `json:"num_goroutines"`
	NumCPU        int   `json:"num_cpu"`
	UptimeSeconds int64 `json:"uptime_seconds"`
}

type Collector struct {
	db         *sqlx.DB
	config     config.DBConfig
	startTime  time.Time
	cachedMem  *MemoryStats
	cacheTime  time.Time
	cacheMutex sync.RWMutex
}

var (
	memStatsCacheDuration = 5 * time.Second
)

func NewCollector(db *sqlx.DB, cfg config.DBConfig) *Collector {
	return &Collector{
		db:        db,
		config:    cfg,
		startTime: time.Now(),
	}
}

func (c *Collector) Collect(ctx context.Context) (*Stats, error) {
	stats := &Stats{
		Timestamp: time.Now(),
	}

	stats.Memory = c.collectMemoryStats()

	dbStats, err := c.collectDatabaseStats(ctx)
	if err != nil {
		return nil, err
	}
	stats.Database = *dbStats
	stats.Runtime = c.collectRuntimeStats()

	return stats, nil
}

func (c *Collector) collectMemoryStats() MemoryStats {
	c.cacheMutex.RLock()
	if c.cachedMem != nil && time.Since(c.cacheTime) < memStatsCacheDuration {
		mem := *c.cachedMem
		c.cacheMutex.RUnlock()
		return mem
	}
	c.cacheMutex.RUnlock()

	c.cacheMutex.Lock()
	defer c.cacheMutex.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mem := MemoryStats{
		Alloc:        m.Alloc,
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		NumGC:        m.NumGC,
		HeapAlloc:    m.HeapAlloc,
		HeapSys:      m.HeapSys,
		HeapInuse:    m.HeapInuse,
		HeapReleased: m.HeapReleased,
	}

	c.cachedMem = &mem
	c.cacheTime = time.Now()

	return mem
}

func (c *Collector) collectDatabaseStats(ctx context.Context) (*DatabaseStats, error) {
	stats := &DatabaseStats{
		Type: string(c.config.Type),
	}

	if totalSize, err := c.getDatabaseSize(ctx); err == nil {
		stats.SizeBytes = totalSize
	}

	tableStats, err := c.getTableStats(ctx)
	if err != nil {
		return nil, err
	}
	stats.TableStats = tableStats

	var totalRecords int64
	for _, ts := range tableStats {
		totalRecords += ts.RowCount
	}
	stats.TotalRecords = totalRecords

	languagesCount, err := c.getAvailableLanguagesCount(ctx)
	if err != nil {
		stats.AvailableLanguages = 0
	} else {
		stats.AvailableLanguages = languagesCount
	}

	return stats, nil
}

func (c *Collector) getDatabaseSize(ctx context.Context) (int64, error) {
	var size int64
	var err error

	if c.config.Type == config.DBTypePostgreSQL {
		err = c.db.GetContext(ctx, &size, "SELECT pg_database_size(current_database())")
	} else {
		err = c.db.GetContext(ctx, &size, "SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()")
	}

	if err != nil {
		return 0, err
	}
	return size, nil
}

func (c *Collector) getAvailableLanguagesCount(ctx context.Context) (int, error) {
	var querySQL string
	if c.config.Type == config.DBTypePostgreSQL {
		querySQL = `
			SELECT COUNT(DISTINCT lang) FROM (
				SELECT lang FROM city_translations
				UNION
				SELECT lang FROM country_translations
			) AS all_langs
		`
	} else {
		querySQL = `
			SELECT COUNT(DISTINCT lang) FROM (
				SELECT lang FROM city_translations
				UNION
				SELECT lang FROM country_translations
			)
		`
	}

	var count int
	err := c.db.GetContext(ctx, &count, querySQL)
	if err != nil {
		return 0, fmt.Errorf("failed to get available languages count: %w", err)
	}

	return count, nil
}

func (c *Collector) getTableStats(ctx context.Context) ([]TableStat, error) {
	var stats []TableStat

	tables := []string{"countries", "cities", "city_translations", "country_translations"}

	for _, table := range tables {
		stat, err := c.getTableStat(ctx, table)
		if err != nil {
			continue
		}
		stats = append(stats, *stat)
	}

	return stats, nil
}

func (c *Collector) getTableStat(ctx context.Context, tableName string) (*TableStat, error) {
	stat := &TableStat{Name: tableName}

	countQuery := "SELECT COUNT(*) FROM " + tableName
	var count int64
	err := c.db.GetContext(ctx, &count, countQuery)
	if err != nil {
		return nil, err
	}
	stat.RowCount = count

	if c.config.Type == config.DBTypePostgreSQL {
		sizeQuery := `SELECT COALESCE(pg_total_relation_size($1::regclass), 0)`
		var size int64
		err = c.db.GetContext(ctx, &size, sizeQuery, tableName)
		if err == nil {
			stat.SizeBytes = size
		}
	} else {
		// Try to use dbstat if available
		sizeQuery := `SELECT SUM(pgsize) FROM dbstat WHERE name = ?`
		var size int64
		_ = c.db.GetContext(ctx, &size, sizeQuery, tableName)
		stat.SizeBytes = size
	}

	return stat, nil
}

func (c *Collector) collectRuntimeStats() RuntimeStats {
	uptime := time.Since(c.startTime).Seconds()
	return RuntimeStats{
		NumGoroutines: runtime.NumGoroutine(),
		NumCPU:        runtime.NumCPU(),
		UptimeSeconds: int64(uptime),
	}
}
