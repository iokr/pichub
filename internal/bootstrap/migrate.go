package bootstrap

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/pichub/pichub/internal/logger"
	"github.com/pichub/pichub/migrations"
)

const createMigrationsTable = "CREATE TABLE IF NOT EXISTS `schema_migrations` (" +
	"`version` VARCHAR(191) NOT NULL COMMENT '迁移版本'," +
	"`applied_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '执行时间(UTC)'," +
	"PRIMARY KEY (`version`)" +
	") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='数据库迁移记录'"

// RunMigrate 按文件名顺序执行尚未应用的迁移脚本。
func RunMigrate(db *gorm.DB, log *logger.Logger) error {
	if err := db.Exec(createMigrationsTable).Error; err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	files, err := migrations.UpFiles()
	if err != nil {
		return fmt.Errorf("list migration files: %w", err)
	}
	applied := 0
	for _, name := range files {
		version := strings.TrimSuffix(name, ".up.sql")
		var count int64
		if err := db.Table("schema_migrations").Where("version = ?", version).Count(&count).Error; err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if count > 0 {
			continue
		}
		content, err := migrations.Read(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		for i, stmt := range splitSQL(content) {
			if err := db.Exec(stmt).Error; err != nil {
				return fmt.Errorf("apply migration %s (statement %d): %w", name, i+1, err)
			}
		}
		if err := db.Exec("INSERT INTO `schema_migrations` (`version`) VALUES (?)", version).Error; err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		applied++
		if log != nil {
			log.Info("migration applied", logger.String("version", version))
		}
	}
	if applied == 0 && log != nil {
		log.Info("database schema is up to date")
	}
	return nil
}

// splitSQL 按分号切分 SQL 脚本，忽略注释并保护引号内的分号。
func splitSQL(src string) []string {
	var (
		out                            []string
		buf                            strings.Builder
		inSingle, inDouble, inBacktick bool
		lineComment, blockComment      bool
	)
	rs := []rune(src)
	at := func(i int) rune {
		if i >= 0 && i < len(rs) {
			return rs[i]
		}
		return 0
	}
	isSpace := func(r rune) bool { return r == ' ' || r == '\t' || r == '\r' || r == '\n' }

	for i := 0; i < len(rs); i++ {
		c, next := rs[i], at(i+1)
		switch {
		case lineComment:
			if c == '\n' {
				lineComment = false
				buf.WriteRune(c)
			}
			continue
		case blockComment:
			if c == '*' && next == '/' {
				blockComment = false
				i++
			}
			continue
		case inSingle:
			buf.WriteRune(c)
			if c == '\\' && next != 0 {
				buf.WriteRune(next)
				i++
				continue
			}
			if c == '\'' {
				if next == '\'' {
					buf.WriteRune(next)
					i++
					continue
				}
				inSingle = false
			}
			continue
		case inDouble:
			buf.WriteRune(c)
			if c == '\\' && next != 0 {
				buf.WriteRune(next)
				i++
				continue
			}
			if c == '"' {
				if next == '"' {
					buf.WriteRune(next)
					i++
					continue
				}
				inDouble = false
			}
			continue
		case inBacktick:
			buf.WriteRune(c)
			if c == '`' {
				if next == '`' {
					buf.WriteRune(next)
					i++
					continue
				}
				inBacktick = false
			}
			continue
		}

		switch {
		case c == '-' && next == '-' && (isSpace(at(i+2)) || at(i+2) == 0):
			lineComment = true
			i++
		case c == '#':
			lineComment = true
		case c == '/' && next == '*':
			blockComment = true
			i++
		case c == '\'':
			inSingle = true
			buf.WriteRune(c)
		case c == '"':
			inDouble = true
			buf.WriteRune(c)
		case c == '`':
			inBacktick = true
			buf.WriteRune(c)
		case c == ';':
			if stmt := strings.TrimSpace(buf.String()); stmt != "" {
				out = append(out, stmt)
			}
			buf.Reset()
		default:
			buf.WriteRune(c)
		}
	}
	if stmt := strings.TrimSpace(buf.String()); stmt != "" {
		out = append(out, stmt)
	}
	return out
}
