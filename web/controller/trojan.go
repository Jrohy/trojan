package controller

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
	"io"
	"log"
	"strconv"
	"strings"
	"time"
	"trojan/core"
	"trojan/trojan"
	websocket "trojan/util"
)

// Start 启动trojan
func Start() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	trojan.Start()
	return &responseBody
}

// Stop 停止trojan
func Stop() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	trojan.Stop()
	return &responseBody
}

// Restart 重启trojan
func Restart() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	trojan.Restart()
	return &responseBody
}

// Update trojan更新
func Update() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	trojan.InstallTrojan()
	return &responseBody
}

// SetLogLevel 修改trojan日志等级
func SetLogLevel(level int) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	core.WriteLogLevel(level)
	trojan.Restart()
	return &responseBody
}

// GetLogLevel 获取trojan日志等级
func GetLogLevel() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	config := core.GetConfig()
	responseBody.Data = map[string]interface{}{
		"loglevel": &config.LogLevel,
	}
	return &responseBody
}

// Log 通过ws查看trojan实时日志
func Log(c *gin.Context) {
	var (
		wsConn *websocket.WsConnection
		err    error
	)
	if wsConn, err = websocket.InitWebsocket(c.Writer, c.Request); err != nil {
		fmt.Println(err)
		return
	}
	defer wsConn.WsClose()
	param := c.DefaultQuery("line", "300")
	if param == "-1" {
		param = "--no-tail"
	} else {
		param = "-n " + param
	}
	result, err := trojan.LogChan(param, wsConn.CloseChan)
	if err != nil {
		fmt.Println(err)
		wsConn.WsClose()
		return
	}
	for line := range result {
		if err := wsConn.WsWrite(ws.TextMessage, []byte(line+"\n")); err != nil {
			log.Println("can't send: ", line)
			break
		}
	}
}

// ImportCsv 导入csv文件到trojan数据库
func ImportCsv(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	defer file.Close()
	filename := header.Filename
	if !strings.Contains(filename, ".csv") {
		responseBody.Msg = "仅支持导入csv格式的文件"
		return &responseBody
	}
	reader := csv.NewReader(bufio.NewReader(file))
	var userList []*core.User
	for {
		line, readErr := reader.Read()
		if readErr == io.EOF {
			break
		} else if readErr != nil {
			responseBody.Msg = readErr.Error()
			return &responseBody
		}
		quota, _ := strconv.Atoi(line[4])
		download, _ := strconv.Atoi(line[5])
		upload, _ := strconv.Atoi(line[6])
		useDays, _ := strconv.Atoi(line[7])
		userList = append(userList, &core.User{
			Username:    line[1],
			Password:    line[2],
			EncryptPass: line[3],
			Quota:       int64(quota),
			Download:    uint64(download),
			Upload:      uint64(upload),
			UseDays:     uint(useDays),
			ExpiryDate:  line[8],
		})
	}
	mysql := core.GetMysql()
	db := mysql.GetDB()
	if _, err = db.Exec("DROP TABLE IF EXISTS users;"); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	if _, err = db.Exec(core.CreateTableSql); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	for _, user := range userList {
		if _, err = db.Exec(fmt.Sprintf(`
INSERT INTO users(username, password, passwordShow, quota, download, upload, useDays, expiryDate) VALUES ('%s','%s','%s', %d, %d, %d, %d, '%s');`,
			user.Username, user.EncryptPass, user.Password, user.Quota, user.Download, user.Upload, user.UseDays, user.ExpiryDate)); err != nil {
			responseBody.Msg = err.Error()
			return &responseBody
		}
	}
	return &responseBody
}

// ExportCsv 导出trojan表数据到csv文件
func ExportCsv(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	var dataBytes = new(bytes.Buffer)
	//设置UTF-8 BOM, 防止中文乱码
	dataBytes.WriteString("\xEF\xBB\xBF")
	mysql := core.GetMysql()
	userList, err := mysql.GetData()
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	wr := csv.NewWriter(dataBytes)
	for _, user := range userList {
		singleUser := []string{
			strconv.Itoa(int(user.ID)),
			user.Username,
			user.Password,
			user.EncryptPass,
			strconv.Itoa(int(user.Quota)),
			strconv.Itoa(int(user.Download)),
			strconv.Itoa(int(user.Upload)),
			strconv.Itoa(int(user.UseDays)),
			user.ExpiryDate,
		}
		wr.Write(singleUser)
	}
	wr.Flush()
	c.Writer.Header().Set("Content-type", "application/octet-stream")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s", fmt.Sprintf("%s.csv", mysql.Database)))
	c.String(200, dataBytes.String())
	return nil
}
