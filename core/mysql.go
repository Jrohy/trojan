package core

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"log"

	// mysql sql驱动
	_ "github.com/go-sql-driver/mysql"
	"strconv"
	"strings"
)

// Mysql 结构体
type Mysql struct {
	Enabled    bool   `json:"enabled"`
	ServerAddr string `json:"server_addr"`
	ServerPort int    `json:"server_port"`
	Database   string `json:"database"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Cafile     string `json:"cafile"`
}

// User 用户表记录结构体
type User struct {
	ID       uint
	Username string
	Password string
	Quota    int64
	Download uint64
	Upload   uint64
}

// GetDB 获取mysql数据库连接
func (mysql *Mysql) GetDB() *sql.DB {
	// 屏蔽mysql驱动包的日志输出
	mysqlDriver.SetLogger(log.New(ioutil.Discard, "", 0))
	conn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", mysql.Username, mysql.Password, mysql.ServerAddr, mysql.ServerPort, mysql.Database)
	db, err := sql.Open("mysql", conn)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	return db
}

// CreateTable 不存在trojan user表则自动创建
func (mysql *Mysql) CreateTable() {
	db := mysql.GetDB()
	defer db.Close()
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS users (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(64) NOT NULL,
    password CHAR(56) NOT NULL,
    quota BIGINT NOT NULL DEFAULT 0,
    download BIGINT UNSIGNED NOT NULL DEFAULT 0,
    upload BIGINT UNSIGNED NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    INDEX (password)
);
    `); err != nil {
		fmt.Println(err)
	}
}

// CreateUser 创建Trojan用户
func (mysql *Mysql) CreateUser(username string, password string) error {
	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	defer db.Close()
	encryPass := sha256.Sum224([]byte(password))
	if _, err := db.Exec(fmt.Sprintf("INSERT INTO users(username, password, quota) VALUES ('%s', '%x', -1);", username, encryPass)); err != nil {
		fmt.Println(err)
		return err
	}
	if err := SetValue(username+"_pass", password); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// UpdateUser 更新Trojan用户名和密码
func (mysql *Mysql) UpdateUser(id uint, username string, password string) error {
	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	defer db.Close()
	encryPass := sha256.Sum224([]byte(password))
	if _, err := db.Exec(fmt.Sprintf("UPDATE users SET username='%s', password='%x' WHERE id=%d;", username, encryPass, id)); err != nil {
		fmt.Println(err)
		return err
	}
	if err := SetValue(username+"_pass", password); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// DeleteUser 删除用户
func (mysql *Mysql) DeleteUser(id uint) error {
	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	defer db.Close()
	userList := mysql.GetData(strconv.Itoa(int(id)))
	if userList == nil {
		return errors.New("can't connnect mysql")
	}
	if userList[0].Username != "admin" {
		_ = DelValue(userList[0].Username + "_pass")
	}
	if _, err := db.Exec(fmt.Sprintf("DELETE FROM users WHERE id=%d;", id)); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// SetQuota 限制流量
func (mysql *Mysql) SetQuota(id uint, quota int) error {
	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	defer db.Close()
	if _, err := db.Exec(fmt.Sprintf("UPDATE users SET quota=%d WHERE id=%d;", quota, id)); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// CleanData 清空流量统计
func (mysql *Mysql) CleanData(id uint) error {
	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	defer db.Close()
	if _, err := db.Exec(fmt.Sprintf("UPDATE users SET download=0, upload=0 WHERE id=%d;", id)); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// GetUserByName 通过用户名来获取用户
func (mysql *Mysql) GetUserByName(name string) *User {
	db := mysql.GetDB()
	if db == nil {
		return nil
	}
	defer db.Close()
	var (
		username   string
		originPass string
		download   uint64
		upload     uint64
		quota      int64
		id         uint
	)
	row := db.QueryRow(fmt.Sprintf("SELECT * FROM users WHERE username='%s'", name))
	if err := row.Scan(&id, &username, &originPass, &quota, &download, &upload); err != nil {
		return nil
	}
	return &User{ID: id, Username: username, Password: originPass, Download: download, Upload: upload, Quota: quota}
}

// GetData 获取用户记录
func (mysql *Mysql) GetData(ids ...string) []*User {
	var dataList []*User
	querySQL := "SELECT * FROM users"
	db := mysql.GetDB()
	if db == nil {
		return nil
	}
	defer db.Close()
	if len(ids) > 0 {
		querySQL = querySQL + " WHERE id in (" + strings.Join(ids, ",") + ")"
	}
	rows, err := db.Query(querySQL)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var (
			username   string
			originPass string
			download   uint64
			upload     uint64
			quota      int64
			id         uint
		)
		if err := rows.Scan(&id, &username, &originPass, &quota, &download, &upload); err != nil {
			fmt.Println(err)
			return nil
		}
		password, err := GetValue(username + "_pass")
		if err != nil {
			password = ""
		}
		dataList = append(dataList, &User{ID: id, Username: username, Password: password, Download: download, Upload: upload, Quota: quota})
	}
	return dataList
}
