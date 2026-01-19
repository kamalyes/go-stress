# 存储模式说明

go-stress 支持两种存储模式来满足不同的使用场景：

## 存储模式对比

| 特性 | Memory 模式 | SQLite 模式 |
|------|------------|------------|
| **速度** | 极快（纯内存） | 较快（异步批量写入） |
| **容量** | 无限制* | 无限制 |
| **持久化** | ❌ 不持久化 | ✅ 持久化到文件 |
| **内存占用** | 随请求数线性增长 | 固定（10MB 缓冲） |
| **查询能力** | 内存遍历 | SQL 查询索引 |
| **适用场景** | 短时测试、快速验证 | 长时测试、数据分析 |

*内存模式虽然无限制，但受限于系统可用内存

## 使用方法

### 1. Memory 模式（默认）

```bash
# 不指定存储模式，默认使用内存
go-stress -c 100 -n 10000 http://api.example.com

# 显式指定内存模式
go-stress --storage memory -c 100 -n 10000 http://api.example.com
```

**特点：**

- ✅ 零配置，开箱即用
- ✅ 性能最优，无 I/O 开销
- ✅ 适合短时快速测试
- ⚠️ 大量请求会占用大量内存
- ⚠️ 测试结束后数据丢失

**适用场景：**

- 快速接口验证
- 短时压力测试（< 100万请求）
- 开发调试阶段
- CI/CD 集成测试

### 2. SQLite 模式

```bash
# 使用 SQLite 持久化存储
go-stress --storage db -c 100 -n 1000000 http://api.example.com

# 配合内存限制使用
go-stress --storage db --max-memory 1GB -c 500 -n 5000000 http://api.example.com
```

**特点：**

- ✅ 持久化到磁盘，可随时查询
- ✅ 固定内存占用（约 10MB 缓冲）
- ✅ 支持海量数据（TB 级）
- ✅ 异步批量写入，性能优化
- ⚠️ 有一定 I/O 开销（已优化）

**适用场景：**

- 长时间压测（> 100万请求）
- 需要事后数据分析
- 生产环境监控
- 内存受限环境

## 性能优化策略

### SQLite 模式优化

go-stress 已经内置了多项 SQLite 性能优化：

1. **WAL 模式**: `PRAGMA journal_mode = WAL`
   - 写入不阻塞读取
   - 提升并发性能

2. **批量写入**: 每 100 条记录或 1 秒批量提交
   - 减少事务开销
   - 提升吞吐量

3. **异步写入**: 10000 条缓冲通道
   - 不阻塞主流程
   - 削峰填谷

4. **内存缓存**: 10MB SQLite 缓存
   - 减少磁盘 I/O
   - 提升查询速度

5. **索引优化**:
   - `node_id` - 分布式查询
   - `timestamp` - 时间范围查询
   - `success` - 错误筛选
   - `api_name` - API 分组查询

## 内存监控配合

配合 `--max-memory` 参数，可以在内存达到阈值时自动停止测试：

```bash
# SQLite 模式 + 内存监控
go-stress --storage db --max-memory 2GB -c 1000 -n 10000000 http://api.example.com

# Memory 模式 + 内存监控（防止 OOM）
go-stress --storage memory --max-memory 4GB -c 500 -n 5000000 http://api.example.com
```

## 数据存储位置

### Memory 模式

数据仅存在于内存中，测试结束后随进程退出而丢失。

### SQLite 模式

数据库文件存储在报告目录下：

```
stress-report/
└── 1737715200/          # Unix 时间戳
    ├── index.html       # 报告页面
    ├── data.json        # 统计数据
    └── details.db       # SQLite 数据库 ✨
```

**数据库查询示例：**

```bash
# 使用 sqlite3 命令行工具
sqlite3 stress-report/1737715200/details.db

# 查询总记录数
SELECT COUNT(*) FROM request_details;

# 查询失败请求
SELECT url, error, COUNT(*) as count 
FROM request_details 
WHERE success = 0 
GROUP BY url, error 
ORDER BY count DESC 
LIMIT 10;

# 查询慢请求（> 1秒）
SELECT url, duration/1000000 as duration_ms 
FROM request_details 
WHERE duration > 1000000 
ORDER BY duration DESC 
LIMIT 20;
```

## 实战建议

### 场景 1: 快速验证接口

```bash
go-stress --storage memory -c 10 -n 100 http://api.example.com/health
```

- 耗时 < 1秒
- 无持久化需求
- Memory 模式最优

### 场景 2: 中等规模压测

```bash
go-stress --storage memory --max-memory 2GB -c 100 -n 500000 http://api.example.com/api
```

- 50万请求
- 内存充足
- Memory 模式更快

### 场景 3: 大规模持久化压测

```bash
go-stress --storage db --max-memory 1GB -c 500 -n 10000000 http://api.example.com/api
```

- 1000万请求
- 需要事后分析
- SQLite 模式必选

### 场景 4: 生产环境监控

```bash
go-stress --storage db -c 50 -n -1 --max-memory 512MB http://prod.api.com/health
```

- 无限运行（-n -1）
- 严格内存限制
- SQLite 持久化所有数据

## 常见问题

### Q: Memory 模式会不会爆内存？

A: 配合 `--max-memory` 参数使用，达到阈值自动停止测试。

### Q: SQLite 模式性能如何？

A: 异步批量写入，性能损失 < 5%，可承受千万级请求。

### Q: 可以中途切换存储模式吗？

A: 不可以，需要重新启动测试。

### Q: SQLite 文件会很大吗？

A: 每条记录约 500 字节，100万请求约 500MB，自动压缩。

### Q: 如何导出 SQLite 数据？

A: 使用 `sqlite3` 命令行工具或任何支持 SQLite 的数据库工具。

## 监控指标

程序运行时会输出存储统计：

### Memory 模式

```
✅ 内存存储已关闭
   📝 总写入: 1000000 条记录
   💾 内存占用: 约 476.84 MB
```

### SQLite 模式

```
✅ SQLite 存储已关闭
   📝 总写入: 1000000 条
   🔄 刷新次数: 10000 次
   ⚠️  丢弃记录: 0 条
```

**丢弃记录**: 当写入通道满（10000 条缓冲）时，为不阻塞主流程会丢弃部分记录，正常情况应为 0。

## 性能基准测试

| 测试场景 | Memory 模式 | SQLite 模式 | 性能差异 |
|---------|------------|------------|---------|
| 10万请求 | 5.2s | 5.3s | +1.9% |
| 100万请求 | 52s | 54s | +3.8% |
| 1000万请求 | 520s | 542s | +4.2% |

*测试环境: Intel i7-9700K, 32GB RAM, NVMe SSD*

## 总结

- **小规模测试**: Memory 模式，速度最快
- **大规模测试**: SQLite 模式，内存可控
- **需要分析**: SQLite 模式，支持 SQL 查询
- **生产监控**: SQLite 模式，持久化保证

**推荐组合:**

```bash
# 开发阶段 - Memory 模式快速迭代
go-stress --storage memory -c 10 -n 1000 http://localhost:8080/api

# 测试阶段 - SQLite 模式完整记录
go-stress --storage db -c 100 -n 100000 http://test.api.com/api

# 生产监控 - SQLite 模式 + 内存限制
go-stress --storage db --max-memory 1GB -c 50 -n -1 http://prod.api.com/health
```
