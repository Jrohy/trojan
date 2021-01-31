package controller

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"strconv"
	"time"
	"trojan/core"
	"trojan/trojan"
)

var c *cron.Cron

// SetData 设置流量限制
func SetData(id uint, quota int) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	if err := mysql.SetQuota(id, quota); err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

// CleanData 清空流量
func CleanData(id uint) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	if err := mysql.CleanData(id); err != nil {
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

func monthlyResetJob() {
	mysql := core.GetMysql()
	if err := mysql.MonthlyResetData(); err != nil {
		fmt.Println("MonthlyResetError: " + err.Error())
	}
}

// GetResetDay 获取重置日
func GetResetDay() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	dayStr, _ := core.GetValue("reset_day")
	day, _ := strconv.Atoi(dayStr)
	responseBody.Data = map[string]interface{}{
		"resetDay": day,
	}
	return &responseBody
}

// UpdateResetDay 更新重置流量日
func UpdateResetDay(day uint) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	if day > 31 || day < 0 {
		responseBody.Msg = fmt.Sprintf("%d为非正常日期", day)
		return &responseBody
	}
	dayStr, _ := core.GetValue("reset_day")
	oldDay, _ := strconv.Atoi(dayStr)
	if day == uint(oldDay) {
		return &responseBody
	}
	if len(c.Entries()) > 1 {
		c.Remove(c.Entries()[len(c.Entries())-1].ID)
	}
	if day != 0 {
		c.AddFunc(fmt.Sprintf("0 0 %d * *", day), func() {
			monthlyResetJob()
		})
	}
	core.SetValue("reset_day", strconv.Itoa(int(day)))
	return &responseBody
}

// SheduleTask 定时任务
func SheduleTask() {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	c = cron.New(cron.WithLocation(loc))
	c.AddFunc("@daily", func() {
		mysql := core.GetMysql()
		if needRestart, err := mysql.DailyCheckExpire(); err != nil {
			fmt.Println("DailyCheckError: " + err.Error())
		} else if needRestart {
			trojan.Restart()
		}
	})

	dayStr, _ := core.GetValue("reset_day")
	if dayStr == "" {
		dayStr = "1"
		core.SetValue("reset_day", dayStr)
	}
	day, _ := strconv.Atoi(dayStr)
	if day != 0 {
		c.AddFunc(fmt.Sprintf("0 0 %d * *", day), func() {
			monthlyResetJob()
		})
	}
	c.Start()
}
