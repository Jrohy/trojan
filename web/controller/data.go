package controller

import (
	"fmt"
	"github.com/robfig/cron/v3"
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
	responseBody.Data = map[string]interface{}{
		"resetDay": c.Entries()[len(c.Entries())-1].Next.Day(),
	}
	return &responseBody
}

// UpdateResetDay 更新重置流量日
func UpdateResetDay(day uint) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	if day > 31 || day < 1 {
		responseBody.Msg = fmt.Sprintf("%d为非正常日期", day)
		return &responseBody
	}
	c.Remove(c.Entries()[len(c.Entries())-1].ID)
	c.AddFunc(fmt.Sprintf("0 0 %d * *", day), func() {
		monthlyResetJob()
	})
	fmt.Println("Updated schedule task: ")
	for _, t := range c.Entries() {
		fmt.Printf("%+v\n", t)
	}
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
	c.AddFunc("@monthly", func() {
		monthlyResetJob()
	})
	c.Start()
}
