/*
 * @Author: kamalyes 501893067@qq.com
 * @Date: 2026-01-24 00:00:00
 * @LastEditors: kamalyes 501893067@qq.com
 * @LastEditTime: 2026-01-24 16:00:00
 * @FilePath: \go-stress\statistics\sqlite.go
 * @Description: SQLite存储层 - 持久化请求详情（支持无限存储）
 *
 * Copyright (c) 2026 by kamalyes, All Rights Reserved.
 */
package statistics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kamalyes/go-logger"
	_ "github.com/mattn/go-sqlite3"
)

// StatusFilter 状态过滤器枚举
type StatusFilter int

const (
	StatusFilterAll     StatusFilter = iota // 全部
	StatusFilterSuccess                     // 成功
	StatusFilterFailed                      // 失败
	StatusFilterSkipped                     // 跳过
)

// String 返回状态过滤器的字符串表示
func (s StatusFilter) String() string {
	switch s {
	case StatusFilterSuccess:
		return "success"
	case StatusFilterFailed:
		return "failed"
	case StatusFilterSkipped:
		return "skipped"
	default:
		return "all"
	}
}

// ParseStatusFilter 从字符串解析状态过滤器
func ParseStatusFilter(s string) StatusFilter {
	switch s {
	case "success":
		return StatusFilterSuccess
	case "failed":
		return StatusFilterFailed
	case "skipped":
		return StatusFilterSkipped
	default:
		return StatusFilterAll
	}
}

// DetailStorageInterface 存储接口（统一 SQLite 和 Memory 实现）
type DetailStorageInterface interface {
	// Write 写入请求详情
	Write(detail *RequestDetail)

	// Query 分页查询请求详情
	Query(offset, limit int, statusFilter StatusFilter) ([]*RequestDetail, error)

	// Count 统计总数
	Count(statusFilter StatusFilter) (int, error)

	// Close 关闭存储并释放资源
	Close() error

	// GetNodeID 获取节点ID（用于分布式场景）
	GetNodeID() string
}

// DetailStorage SQLite持久化存储（实现 DetailStorageInterface）
type DetailStorage struct {
	db          *sql.DB
	writeChan   chan *RequestDetail
	batchSize   int
	flushTicker *time.Ticker
	wg          sync.WaitGroup
	closed      bool
	mu          sync.Mutex
	nodeID      string // 节点ID（分布式模式下标识数据来源）
	logger      logger.ILogger

	// 统计信息
	writeCount    uint64 // 写入总数
	flushCount    uint64 // 刷新次数
	dropCount     uint64 // 丢弃数（通道满）
	lastFlushTime time.Time
}

// NewDetailStorage 创建存储实例
func NewDetailStorage(dbPath, nodeID string, log logger.ILogger) (*DetailStorage, error) {
	// 如果不是内存模式，确保目录存在
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建目录失败: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 设置连接池（SQLite 仅支持单写多读）
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // 连接永不过期

	// 优化 SQLite 性能
	pragmas := []string{
		"PRAGMA journal_mode = WAL",    // Write-Ahead Logging 模式
		"PRAGMA synchronous = NORMAL",  // 平衡性能和安全性
		"PRAGMA cache_size = 10000",    // 10MB 缓存
		"PRAGMA temp_store = MEMORY",   // 临时表存内存
		"PRAGMA mmap_size = 268435456", // 256MB 内存映射
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			log.Warnf("⚠️  执行 %s 失败: %v", pragma, err)
		}
	}

	// 创建表结构
	schema := `
	CREATE TABLE IF NOT EXISTS request_details (
		id TEXT PRIMARY KEY,
		node_id TEXT NOT NULL,
		group_id TEXT,
		api_name TEXT,
		timestamp INTEGER NOT NULL,
		url TEXT,
		method TEXT,
		query TEXT,
		headers TEXT,
		body TEXT,
		duration INTEGER NOT NULL,
		status_code INTEGER,
		success INTEGER NOT NULL,
		skipped INTEGER NOT NULL,
		size INTEGER,
		error TEXT,
		response_body TEXT,
		response_headers TEXT,
		verifications TEXT,
		extracted_vars TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_node_id ON request_details(node_id);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON request_details(timestamp);
	CREATE INDEX IF NOT EXISTS idx_success ON request_details(success);
	CREATE INDEX IF NOT EXISTS idx_skipped ON request_details(skipped);
	CREATE INDEX IF NOT EXISTS idx_api_name ON request_details(api_name);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("创建表失败: %w", err)
	}

	if dbPath != ":memory:" {
		log.Infof("💾 SQLite 存储已启用: %s (节点: %s)", dbPath, nodeID)
	} else {
		log.Infof("💾 SQLite 内存模式已启用 (节点: %s)", nodeID)
	}

	storage := &DetailStorage{
		db:            db,
		writeChan:     make(chan *RequestDetail, 10000), // 1万缓冲
		batchSize:     100,                              // 每100条批量写入
		flushTicker:   time.NewTicker(1 * time.Second),  // 每秒强制刷新
		nodeID:        nodeID,
		logger:        log,
		writeCount:    0,
		flushCount:    0,
		dropCount:     0,
		lastFlushTime: time.Now(),
	}

	// 启动异步写入协程
	storage.wg.Add(1)
	go storage.batchWriter()

	return storage, nil
}

// Write 异步写入请求详情（实现 DetailStorageInterface）
func (s *DetailStorage) Write(detail *RequestDetail) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	select {
	case s.writeChan <- detail:
		// 写入成功
	default:
		// 通道满了，丢弃（避免阻塞主流程）
		s.dropCount++
		if s.dropCount%1000 == 1 { // 每1000次丢弃警告一次
			s.logger.Warnf("⚠️  写入通道已满，已丢弃 %d 条记录", s.dropCount)
		}
	}
}

// batchWriter 批量写入协程
func (s *DetailStorage) batchWriter() {
	defer s.wg.Done()

	batch := make([]*RequestDetail, 0, s.batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		start := time.Now()
		if err := s.insertBatch(batch); err != nil {
			// 写入失败，记录日志但不阻塞
			s.logger.Errorf("❌ 批量写入 %d 条记录失败: %v", len(batch), err)
		} else {
			s.writeCount += uint64(len(batch))
			s.flushCount++
			s.lastFlushTime = time.Now()

			// 每写入10000条记录输出一次统计
			if s.writeCount%10000 == 0 {
				duration := time.Since(start)
				s.logger.Debugf("📊 已写入 %d 条记录 (本次: %d 条, 耗时: %v)",
					s.writeCount, len(batch), duration)
			}
		}

		batch = batch[:0] // 清空但保留容量
	}

	for {
		select {
		case detail, ok := <-s.writeChan:
			if !ok {
				// 通道关闭，刷新剩余数据
				flush()
				return
			}

			batch = append(batch, detail)
			if len(batch) >= s.batchSize {
				flush()
			}

		case <-s.flushTicker.C:
			// 定时刷新
			flush()
		}
	}
}

// insertBatch 批量插入
func (s *DetailStorage) insertBatch(batch []*RequestDetail) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO request_details (
			id, node_id, group_id, api_name, timestamp, url, method, query, headers, body,
			duration, status_code, success, skipped, size, error,
			response_body, response_headers, verifications, extracted_vars
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, detail := range batch {
		// 序列化复杂字段
		headersJSON, _ := json.Marshal(detail.Headers)
		respHeadersJSON, _ := json.Marshal(detail.ResponseHeaders)
		verificationsJSON, _ := json.Marshal(detail.Verifications)
		extractedVarsJSON, _ := json.Marshal(detail.ExtractedVars)

		_, err := stmt.Exec(
			detail.ID,
			s.nodeID, // 记录节点ID
			detail.GroupID,
			detail.APIName,
			detail.Timestamp.Unix(),
			detail.URL,
			detail.Method,
			detail.Query,
			string(headersJSON),
			detail.Body,
			detail.Duration.Microseconds(),
			detail.StatusCode,
			boolToInt(detail.Success),
			boolToInt(detail.Skipped),
			detail.Size,
			detail.Error,
			detail.ResponseBody,
			string(respHeadersJSON),
			string(verificationsJSON),
			string(extractedVarsJSON),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Query 分页查询请求详情
func (s *DetailStorage) Query(offset, limit int, statusFilter StatusFilter) ([]*RequestDetail, error) {
	query := "SELECT * FROM request_details"

	// 根据状态过滤
	switch statusFilter {
	case StatusFilterSuccess:
		query += " WHERE success = 1 AND skipped = 0"
	case StatusFilterFailed:
		query += " WHERE success = 0 AND skipped = 0"
	case StatusFilterSkipped:
		query += " WHERE skipped = 1"
	case StatusFilterAll:
	default:
		// 默认查询全部
	}

	query += " ORDER BY id DESC LIMIT ? OFFSET ?"

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*RequestDetail
	for rows.Next() {
		detail, err := s.scanDetail(rows)
		if err != nil {
			continue
		}
		results = append(results, detail)
	}

	return results, nil
}

// Count 统计总数
func (s *DetailStorage) Count(statusFilter StatusFilter) (int, error) {
	query := "SELECT COUNT(*) FROM request_details"

	// 根据状态过滤
	switch statusFilter {
	case StatusFilterSuccess:
		query += " WHERE success = 1 AND skipped = 0"
	case StatusFilterFailed:
		query += " WHERE success = 0 AND skipped = 0"
	case StatusFilterSkipped:
		query += " WHERE skipped = 1"
	case StatusFilterAll:
		// 不添加过滤条件
	default:
		// 默认查询全部
	}

	var count int
	err := s.db.QueryRow(query).Scan(&count)
	return count, err
}

// scanDetail 扫描行数据
func (s *DetailStorage) scanDetail(rows *sql.Rows) (*RequestDetail, error) {
	var (
		detail                               RequestDetail
		nodeID                               string
		timestamp, duration                  int64
		success, skipped                     int
		headersJSON, respHeadersJSON         string
		verificationsJSON, extractedVarsJSON string
	)

	err := rows.Scan(
		&detail.ID, &nodeID, &detail.GroupID, &detail.APIName, &timestamp,
		&detail.URL, &detail.Method, &detail.Query, &headersJSON, &detail.Body,
		&duration, &detail.StatusCode, &success, &skipped, &detail.Size, &detail.Error,
		&detail.ResponseBody, &respHeadersJSON, &verificationsJSON, &extractedVarsJSON,
	)
	if err != nil {
		return nil, err
	}

	detail.Timestamp = time.Unix(timestamp, 0)
	detail.Duration = time.Duration(duration) * time.Microsecond
	detail.Success = success == 1
	detail.Skipped = skipped == 1

	json.Unmarshal([]byte(headersJSON), &detail.Headers)
	json.Unmarshal([]byte(respHeadersJSON), &detail.ResponseHeaders)
	json.Unmarshal([]byte(verificationsJSON), &detail.Verifications)
	json.Unmarshal([]byte(extractedVarsJSON), &detail.ExtractedVars)

	return &detail, nil
}

// Close 关闭存储
func (s *DetailStorage) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	// 关闭写入通道，触发 batchWriter 刷新剩余数据
	close(s.writeChan)

	// 停止定时器
	s.flushTicker.Stop()

	// 等待 batchWriter 完成（会自动刷新剩余数据）
	s.wg.Wait()

	// 输出最终统计
	s.logger.Infof("✅ SQLite 存储已关闭")
	s.logger.Infof("   📝 总写入: %d 条", s.writeCount)
	s.logger.Infof("   🔄 刷新次数: %d 次", s.flushCount)
	if s.dropCount > 0 {
		s.logger.Warnf("   ⚠️  丢弃记录: %d 条", s.dropCount)
	}

	return s.db.Close()
}

// GetNodeID 获取节点ID（实现 DetailStorageInterface）
func (s *DetailStorage) GetNodeID() string {
	return s.nodeID
}

// GetStats 获取存储统计信息
func (s *DetailStorage) GetStats() (writeCount, flushCount, dropCount uint64) {
	return s.writeCount, s.flushCount, s.dropCount
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
