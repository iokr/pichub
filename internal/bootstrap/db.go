// Package bootstrap 负责依赖装配与进程生命周期管理。
package bootstrap

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/logger"
)

// NewDB 打开 MySQL 连接、配置连接池并做一次连通性检查。
func NewDB(cfg *config.Config, log *logger.Logger) (*gorm.DB, error) {
	mc := gomysql.NewConfig()
	mc.User = cfg.Database.Username
	mc.Passwd = cfg.Database.Password
	mc.Net = "tcp"
	mc.Addr = net.JoinHostPort(cfg.Database.Host, strconv.Itoa(cfg.Database.Port))
	mc.DBName = cfg.Database.Database
	mc.ParseTime = cfg.Database.ParseTime
	mc.Collation = cfg.Database.Collation
	mc.Params = map[string]string{"charset": cfg.Database.Charset}
	if cfg.Database.Loc != "" {
		loc, err := time.LoadLocation(cfg.Database.Loc)
		if err != nil {
			return nil, fmt.Errorf("load database location %q: %w", cfg.Database.Loc, err)
		}
		mc.Loc = loc
	}

	db, err := gorm.Open(mysql.Open(mc.FormatDSN()), &gorm.Config{
		Logger:                                   newGormLogger(cfg.Database, log),
		DisableForeignKeyConstraintWhenMigrating: true,
		NowFunc:                                  func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	if cfg.Database.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	}
	if cfg.Database.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)
	}
	if cfg.Database.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.Database.ConnMaxIdleTime)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping database %s:%d: %w", cfg.Database.Host, cfg.Database.Port, err)
	}
	if log != nil {
		log.Info("database connected",
			logger.String("host", cfg.Database.Host),
			logger.Int("port", cfg.Database.Port),
			logger.String("database", cfg.Database.Database))
	}
	return db, nil
}

// gormWriter 把 GORM 日志转交给项目日志器。
type gormWriter struct{ log *logger.Logger }

func (w gormWriter) Printf(format string, args ...any) {
	if w.log != nil {
		w.log.Warn("gorm: " + fmt.Sprintf(format, args...))
	}
}

func newGormLogger(c config.DatabaseConfig, log *logger.Logger) gormlogger.Interface {
	level := gormlogger.Warn
	switch strings.ToLower(c.LogLevel) {
	case "silent":
		level = gormlogger.Silent
	case "error":
		level = gormlogger.Error
	case "info":
		level = gormlogger.Info
	}
	slow := c.SlowThreshold
	if slow <= 0 {
		slow = 200 * time.Millisecond
	}
	return gormlogger.New(gormWriter{log: log}, gormlogger.Config{
		SlowThreshold:             slow,
		LogLevel:                  level,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	})
}
