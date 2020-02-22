package core

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"strconv"
	"strings"
	"trojan/database"
)

type Mysql struct {
	Enabled          bool     `json:"enabled"`
	ServerAddr       string   `json:"server_addr"`
	ServerPort       int      `json:"server_port"`
	Database         string   `json:"database"`
	Username         string   `json:"username"`
	Password         string   `json:"password"`
}

type User struct {
	ID               uint
	Username         string
	Password         string
	Quota            int64
	Download         uint64
	Upload           uint64
}

func (mysql *Mysql)GetDB() *sql.DB {
	conn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", mysql.Username, mysql.Password, mysql.ServerAddr, mysql.ServerPort, mysql.Database)
	db, err := sql.Open("mysql", conn)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	return db
}

func (mysql *Mysql)CreateTable() {
	db := mysql.GetDB()
	defer db.Close()
	if _, err := db.Exec(`
CREATE TABLE users (
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

func (mysql *Mysql)CreateUser(username string, password string) error {
	db := mysql.GetDB()
	defer db.Close()
	encryPass := sha256.Sum224([]byte(password))
	if _, err := db.Exec(fmt.Sprintf("INSERT INTO users(username, password, quota) VALUES ('%s', '%x', -1);", username, encryPass)); err != nil {
		fmt.Println(err)
		return err
	}
	if err := database.SetValue(username + "_pass", password); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (mysql *Mysql)DeleteUser(id uint) error {
	db := mysql.GetDB()
	defer db.Close()
	userList := *mysql.GetData(strconv.Itoa(int(id)))
	_ = database.DelValue(userList[0].Username + "_pass")
	if _, err := db.Exec(fmt.Sprintf("DELETE FROM users WHERE id=%d;", id)); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (mysql *Mysql)SetQuota(id uint, quota int) error {
	db := mysql.GetDB()
	defer db.Close()
	if _, err := db.Exec(fmt.Sprintf("UPDATE users SET quota=%d WHERE id=%d;", quota, id)); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (mysql *Mysql)CleanData(id uint) error {
	db := mysql.GetDB()
	defer db.Close()
	if _, err := db.Exec(fmt.Sprintf("UPDATE users SET download=0 AND upload=0 WHERE id=%d;", id)); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (mysql *Mysql)GetData(ids ...string) *[]User {
	var dataList []User
	querySQL := "SELECT * FROM users"
	db := mysql.GetDB()
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
			password   string
			download   uint64
			upload     uint64
			quota      int64
			id         uint
		)
		if err := rows.Scan(&id, &username, &password, &quota, &download, &upload); err != nil {
			fmt.Println(err)
			return nil
		}
		dataList = append(dataList, User{ID:id, Username:username, Password:password, Download:download, Upload:upload, Quota:quota})
	}
	return &dataList
}