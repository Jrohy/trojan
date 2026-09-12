package controller

import (
	"errors"
	"math"
	"time"
	"trojan/core"
)

const portalExpiringSoonDays = 7

type portalTrafficSummary struct {
	Quota      int64    `json:"quota"`
	Used       uint64   `json:"used"`
	Remaining  uint64   `json:"remaining"`
	Upload     uint64   `json:"upload"`
	Download   uint64   `json:"download"`
	UsageRatio *float64 `json:"usageRatio"`
	Unlimited  bool     `json:"unlimited"`
}

type portalOverview struct {
	Username      string               `json:"username"`
	Status        string               `json:"status"`
	StatusText    string               `json:"statusText"`
	ExpiryDate    string               `json:"expiryDate"`
	DaysRemaining *int                 `json:"daysRemaining"`
	Traffic       portalTrafficSummary `json:"traffic"`
	UpdatedAt     string               `json:"updatedAt"`
}

func addTraffic(upload, download uint64) uint64 {
	if math.MaxUint64-upload < download {
		return math.MaxUint64
	}
	return upload + download
}

func portalStatus(now time.Time, expiryDate string, quota int64, used uint64) (string, string, *int, error) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return "", "", nil, err
	}
	today, _ := time.ParseInLocation("2006-01-02", now.In(location).Format("2006-01-02"), location)
	var daysRemaining *int
	if expiryDate != "" {
		expiry, err := time.ParseInLocation("2006-01-02", expiryDate, location)
		if err != nil {
			return "", "", nil, errors.New("账号到期日期格式无效")
		}
		days := int(expiry.Sub(today).Hours() / 24)
		daysRemaining = &days
		if days < 0 {
			return "expired", "账号已过期", daysRemaining, nil
		}
	}

	if quota >= 0 && used >= uint64(quota) {
		return "traffic_exhausted", "流量已用尽", daysRemaining, nil
	}
	if daysRemaining != nil && *daysRemaining <= portalExpiringSoonDays {
		return "expiring_soon", "即将到期", daysRemaining, nil
	}
	return "normal", "账号正常", daysRemaining, nil
}

// PortalOverview returns the minimum account and traffic information for the authenticated portal user.
func PortalOverview(userID uint) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	mysql := core.GetMysql()
	usage, err := mysql.PortalUserUsageByID(userID)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	used := addTraffic(usage.Upload, usage.Download)
	status, statusText, daysRemaining, err := portalStatus(time.Now(), usage.ExpiryDate, usage.Quota, used)
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	traffic := portalTrafficSummary{
		Quota:     usage.Quota,
		Used:      used,
		Upload:    usage.Upload,
		Download:  usage.Download,
		Unlimited: usage.Quota < 0,
	}
	if usage.Quota >= 0 {
		quota := uint64(usage.Quota)
		if used < quota {
			traffic.Remaining = quota - used
		}
		ratio := 100.0
		if quota > 0 {
			ratio = math.Min(100, float64(used)/float64(quota)*100)
		}
		traffic.UsageRatio = &ratio
	}
	responseBody.Data = portalOverview{
		Username:      usage.Username,
		Status:        status,
		StatusText:    statusText,
		ExpiryDate:    usage.ExpiryDate,
		DaysRemaining: daysRemaining,
		Traffic:       traffic,
		UpdatedAt:     time.Now().In(time.FixedZone("Asia/Shanghai", 8*60*60)).Format(time.RFC3339),
	}
	return &responseBody
}
