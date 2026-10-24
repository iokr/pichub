//go:build ignore

// 临时诊断脚本：查看远程库迁移状态，用完即删。
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := os.Getenv("DSN")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer db.Close()

	var ver, verComment string
	_ = db.QueryRow("SELECT VERSION(), @@version_comment").Scan(&ver, &verComment)
	fmt.Println("mysql version:", ver, "|", verComment)
	var charset, coll string
	_ = db.QueryRow("SELECT @@character_set_server, @@collation_server").Scan(&charset, &coll)
	fmt.Println("server charset/collation:", charset, coll)

	for _, t := range []string{"schema_migrations", "user_groups", "users", "email_tokens"} {
		var n int64
		if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", t)).Scan(&n); err != nil {
			fmt.Println("count", t, "err:", err)
			continue
		}
		fmt.Println("count", t, "=", n)
	}

	var versions string
	_ = db.QueryRow("SELECT IFNULL(GROUP_CONCAT(version), '(empty)') FROM schema_migrations").Scan(&versions)
	fmt.Println("schema_migrations versions:", versions)
}
