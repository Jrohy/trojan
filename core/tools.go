package core

import (
	"bufio"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"trojan/util"
)

// UpgradeDB 升级数据库表结构以及迁移数据
func (mysql *Mysql) UpgradeDB() error {
	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	var field string
	error := db.QueryRow("SHOW COLUMNS FROM users LIKE 'passwordShow';").Scan(&field)
	if error == sql.ErrNoRows {
		fmt.Println(util.Yellow("正在进行数据库升级, 请稍等.."))
		if _, err := db.Exec("ALTER TABLE users ADD COLUMN passwordShow VARCHAR(255) NOT NULL AFTER password;"); err != nil {
			fmt.Println(err)
			return err
		}
		userList, err := mysql.GetData()
		if err != nil {
			fmt.Println(err)
			return err
		}
		for _, user := range userList {
			pass, _ := GetValue(fmt.Sprintf("%s_pass", user.Username))
			if pass != "" {
				base64Pass := base64.StdEncoding.EncodeToString([]byte(pass))
				if _, err := db.Exec(fmt.Sprintf("UPDATE users SET passwordShow='%s' WHERE id=%d;", base64Pass, user.ID)); err != nil {
					fmt.Println(err)
					return err
				}
				DelValue(fmt.Sprintf("%s_pass", user.Username))
			}
		}
	}
	error = db.QueryRow("SHOW COLUMNS FROM users LIKE 'useDays';").Scan(&field)
	if error == sql.ErrNoRows {
		fmt.Println(util.Yellow("正在进行数据库升级, 请稍等.."))
		if _, err := db.Exec(`
ALTER TABLE users
ADD COLUMN useDays int(10) DEFAULT 0,
ADD COLUMN expiryDate char(10) DEFAULT '';
`); err != nil {
			fmt.Println(err)
			return err
		}
	}
	return nil
}

// DumpSql 导出sql
func (mysql *Mysql) DumpSql(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	writer.WriteString("DROP TABLE IF EXISTS users;")
	writer.WriteString(createTableSql)
	db := mysql.GetDB()
	userList, err := queryUserList(db, "SELECT * FROM users;")
	if err != nil {
		return err
	}
	for _, user := range userList {
		writer.WriteString(fmt.Sprintf(`
INSERT INTO users(username, password, passwordShow, quota, download, upload, useDays, expiryDate) VALUES ('%s','%s','%s', %d, %d, %d, %d, '%s');`,
			user.Username, user.EncryptPass, user.Password, user.Quota, user.Download, user.Upload, user.UseDays, user.ExpiryDate))
	}
	writer.WriteString("\n")
	writer.Flush()
	return nil
}

// ExecSql 执行sql
func (mysql *Mysql) ExecSql(filePath string) error {
	db := mysql.GetDB()
	fileByte, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	sqlStr := string(fileByte)
	sqls := strings.Split(strings.Replace(sqlStr, "\r\n", "\n", -1), ";\n")
	for _, s := range sqls {
		s = strings.TrimSpace(s)
		if s != "" {
			if _, err = db.Exec(s); err != nil {
				return err
			}
		}
	}
	return nil
}
