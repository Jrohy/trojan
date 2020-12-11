package core

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"time"
	"trojan/util"

	mysqlDriver "github.com/go-sql-driver/mysql"

	"strconv"
	"strings"

	// mysql sql驱动
	_ "github.com/go-sql-driver/mysql"
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
	ID          uint
	Username    string
	Password    string
	Quota       int64
	Download    uint64
	Upload      uint64
	UseDays     uint
	ExpiredDate string
}

// PageQuery 分页查询的结构体
type PageQuery struct {
	PageNum  int
	CurPage  int
	Total    int
	PageSize int
	DataList []*User
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
    passwordShow VARCHAR(255) NOT NULL,
    quota BIGINT NOT NULL DEFAULT 0,
    download BIGINT UNSIGNED NOT NULL DEFAULT 0,
    upload BIGINT UNSIGNED NOT NULL DEFAULT 0,
    useDays int(10) DEFAULT NULL,
    expiredDate char(10) DEFAULT NULL,
    PRIMARY KEY (id),
    INDEX (password)
);
    `); err != nil {
		fmt.Println(err)
	}
}

func queryUserList(db *sql.DB, sql string) ([]*User, error) {
	var (
		username    string
		originPass  string
		passShow    string
		download    uint64
		upload      uint64
		quota       int64
		id          uint
		useDays     uint
		expiredDate string
	)
	var userList []*User
	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&id, &username, &originPass, &passShow, &quota, &download, &upload, &useDays, &expiredDate); err != nil {
			return nil, err
		}
		userList = append(userList, &User{
			ID:          id,
			Username:    username,
			Password:    passShow,
			Download:    download,
			Upload:      upload,
			Quota:       quota,
			UseDays:     useDays,
			ExpiredDate: expiredDate,
		})
	}
	return userList, nil
}

func queryUser(db *sql.DB, sql string) (*User, error) {
	var (
		username    string
		originPass  string
		passShow    string
		download    uint64
		upload      uint64
		quota       int64
		id          uint
		useDays     uint
		expiredDate string
	)
	row := db.QueryRow(sql)
	if err := row.Scan(&id, &username, &originPass, &passShow, &quota, &download, &upload, &useDays, &expiredDate); err != nil {
		return nil, err
	}
	return &User{ID: id, Username: username, Password: originPass, Download: download, Upload: upload, Quota: quota, UseDays: useDays, ExpiredDate: expiredDate}, nil
}

// CreateUser 创建Trojan用户
func (mysql *Mysql) CreateUser(username string, base64Pass string, originPass string) error {
	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	defer db.Close()
	encryPass := sha256.Sum224([]byte(originPass))
	if _, err := db.Exec(fmt.Sprintf("INSERT INTO users(username, password, passwordShow, quota) VALUES ('%s', '%x', '%s', -1);", username, encryPass, base64Pass)); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// UpdateUser 更新Trojan用户名和密码
func (mysql *Mysql) UpdateUser(id uint, username string, base64Pass string, originPass string) error {
	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	defer db.Close()
	encryPass := sha256.Sum224([]byte(originPass))
	if _, err := db.Exec(fmt.Sprintf("UPDATE users SET username='%s', password='%x', passwordShow='%s' WHERE id=%d;", username, encryPass, base64Pass, id)); err != nil {
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
	if userList, err := mysql.GetData(strconv.Itoa(int(id))); err != nil {
		return err
	} else if userList != nil && len(userList) == 0 {
		return fmt.Errorf("不存在id为%d的用户", id)
	}
	if _, err := db.Exec(fmt.Sprintf("DELETE FROM users WHERE id=%d;", id)); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// MonthlyResetData 设置了过期时间的用户，每月定时清空使用流量
func (mysql *Mysql) MonthlyResetData() error {
	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	defer db.Close()
	userList, err := queryUserList(db, "SELECT * FROM users WHERE useDays is NOT NULL AND quota != 0")
	if err != nil {
		fmt.Println(err)
		return err
	}
	for _, user := range userList {
		if _, err := db.Exec(fmt.Sprintf("UPDATE users SET download=0, upload=0 WHERE id=%d;", user.ID)); err != nil {
			fmt.Println(err)
			return err
		}
	}
	return nil
}

// DailyCheckExpire 检查是否有过期，过期了设置流量上限为0
func (mysql *Mysql) DailyCheckExpire() error {
	now := time.Now()
	utc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fmt.Println(err)
		return err
	}
	todayDay := now.In(utc).Format("2006-01-02")
	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	defer db.Close()
	userList, err := queryUserList(db, "SELECT * FROM users WHERE useDays is NOT NULL AND quota != 0")
	if err != nil {
		fmt.Println(err)
		return err
	}
	for _, user := range userList {
		if user.ExpiredDate == todayDay {
			if _, err := db.Exec(fmt.Sprintf("UPDATE users SET quota=0 WHERE id=%d;", user.ID)); err != nil {
				fmt.Println(err)
				return err
			}
		}
	}
	return nil
}

// CancelExpire 取消过期时间
func (mysql *Mysql) CancelExpire(id uint) error {
	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	defer db.Close()
	if _, err := db.Exec(fmt.Sprintf("UPDATE users SET useDays=NULL, expiredDate=NULL WHERE id=%d;", id)); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// SetExpire 设置过期时间
func (mysql *Mysql) SetExpire(id uint, useDays uint) error {
	now := time.Now()
	utc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		fmt.Println(err)
		return err
	}
	addDay, _ := time.ParseDuration(strconv.Itoa(int(24*(useDays+1))) + "h")
	expiredDate := now.Add(addDay).In(utc).Format("2006-01-02")

	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	defer db.Close()
	if _, err := db.Exec(fmt.Sprintf("UPDATE users SET useDays=%d, expiredDate='%s' WHERE id=%d;", addDay, expiredDate, id)); err != nil {
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
ADD COLUMN useDays int(10) DEFAULT NULL,
ADD COLUMN expiredDate char(10) DEFAULT NULL;
`); err != nil {
			fmt.Println(err)
			return err
		}
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

// CleanDataByName 清空指定用户名流量统计数据
func (mysql *Mysql) CleanDataByName(usernames []string) error {
	db := mysql.GetDB()
	if db == nil {
		return errors.New("can't connect mysql")
	}
	defer db.Close()
	runSql := "UPDATE users SET download=0, upload=0 WHERE username in ("
	for i, name := range usernames {
		runSql = runSql + "'" + name + "'"
		if i == len(usernames)-1 {
			runSql = runSql + ")"
		} else {
			runSql = runSql + ","
		}
	}
	if _, err := db.Exec(runSql); err != nil {
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
	user, err := queryUser(db, fmt.Sprintf("SELECT * FROM users WHERE username='%s'", name))
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return user
}

// GetUserByPass 通过密码来获取用户
func (mysql *Mysql) GetUserByPass(pass string) *User {
	db := mysql.GetDB()
	if db == nil {
		return nil
	}
	defer db.Close()
	user, err := queryUser(db, fmt.Sprintf("SELECT * FROM users WHERE passwordShow='%s'", pass))
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return user
}

// PageList 通过分页获取用户记录
func (mysql *Mysql) PageList(curPage int, pageSize int) (*PageQuery, error) {
	var (
		total int
	)

	db := mysql.GetDB()
	if db == nil {
		return nil, errors.New("连接mysql失败")
	}
	defer db.Close()
	offset := (curPage - 1) * pageSize
	querySQL := fmt.Sprintf("SELECT * FROM users LIMIT %d, %d", offset, pageSize)
	userList, err := queryUserList(db, querySQL)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	db.QueryRow("SELECT COUNT(id) FROM users").Scan(&total)
	return &PageQuery{
		CurPage:  curPage,
		PageSize: pageSize,
		Total:    total,
		DataList: userList,
		PageNum:  (total + pageSize - 1) / pageSize,
	}, nil
}

// GetData 获取用户记录
func (mysql *Mysql) GetData(ids ...string) ([]*User, error) {
	querySQL := "SELECT * FROM users"
	db := mysql.GetDB()
	if db == nil {
		return nil, errors.New("连接mysql失败")
	}
	defer db.Close()
	if len(ids) > 0 {
		querySQL = querySQL + " WHERE id in (" + strings.Join(ids, ",") + ")"
	}
	userList, err := queryUserList(db, querySQL)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return userList, nil
}
