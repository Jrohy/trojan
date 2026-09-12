package controller

import (
	"sync"
	"time"
	"trojan/core"
	"trojan/trojan"
)

const (
	dashboardPackageQuota = uint64(2 * 1024 * 1024 * 1024 * 1024)
	dashboardCacheTTL     = 30 * time.Second
)

type dashboardServiceSummary struct {
	State   string `json:"state"`
	Running bool   `json:"running"`
	Type    string `json:"type"`
	Uptime  string `json:"uptime"`
}

type dashboardAccountSummary struct {
	Total            int `json:"total"`
	Normal           int `json:"normal"`
	ExpiringSoon     int `json:"expiringSoon"`
	Expired          int `json:"expired"`
	TrafficExhausted int `json:"trafficExhausted"`
}

type dashboardCertificateSummary struct {
	CommonName string `json:"commonName"`
	NotAfter   string `json:"notAfter"`
	DaysLeft   int    `json:"daysLeft"`
	Warning    bool   `json:"warning"`
	Error      string `json:"error,omitempty"`
}

type dashboardMonthCapacity struct {
	PackageQuota   uint64  `json:"packageQuota"`
	Remaining      uint64  `json:"remaining"`
	UsagePercent   float64 `json:"usagePercent"`
	ElapsedDays    int     `json:"elapsedDays"`
	DaysInMonth    int     `json:"daysInMonth"`
	DailyAverage   uint64  `json:"dailyAverage"`
	ProjectedTotal uint64  `json:"projectedTotal"`
}

type dashboardOverviewData struct {
	Service     dashboardServiceSummary     `json:"service"`
	Accounts    dashboardAccountSummary     `json:"accounts"`
	Certificate dashboardCertificateSummary `json:"certificate"`
	Traffic     accessTrafficOverview       `json:"traffic"`
	Capacity    dashboardMonthCapacity      `json:"capacity"`
	Network     networkTrafficSnapshot      `json:"network"`
	UpdatedAt   string                      `json:"updatedAt"`
}

var dashboardOverviewCache = struct {
	sync.Mutex
	Data      dashboardOverviewData
	ExpiresAt time.Time
	Ready     bool
}{}

func dashboardAccounts(now time.Time) (dashboardAccountSummary, error) {
	result := dashboardAccountSummary{}
	users, err := core.GetMysql().GetData()
	if err != nil {
		return result, err
	}
	for _, user := range users {
		if isOpsProbeHash(user.EncryptPass) {
			continue
		}
		result.Total++
		status, _, _, statusErr := portalStatus(now, user.ExpiryDate, user.Quota, addTraffic(user.Upload, user.Download))
		if statusErr != nil {
			return result, statusErr
		}
		switch status {
		case "expiring_soon":
			result.ExpiringSoon++
		case "expired":
			result.Expired++
		case "traffic_exhausted":
			result.TrafficExhausted++
		default:
			result.Normal++
		}
	}
	return result, nil
}

func monthCapacity(now time.Time, monthTotal uint64) dashboardMonthCapacity {
	local := now.In(chinaLocation())
	daysInMonth := time.Date(local.Year(), local.Month()+1, 0, 0, 0, 0, 0, local.Location()).Day()
	elapsedDays := local.Day()
	dailyAverage := monthTotal / uint64(elapsedDays)
	projected := dailyAverage * uint64(daysInMonth)
	remaining := uint64(0)
	if monthTotal < dashboardPackageQuota {
		remaining = dashboardPackageQuota - monthTotal
	}
	return dashboardMonthCapacity{
		PackageQuota:   dashboardPackageQuota,
		Remaining:      remaining,
		UsagePercent:   float64(monthTotal) / float64(dashboardPackageQuota) * 100,
		ElapsedDays:    elapsedDays,
		DaysInMonth:    daysInMonth,
		DailyAverage:   dailyAverage,
		ProjectedTotal: projected,
	}
}

func buildDashboardOverview() (dashboardOverviewData, error) {
	now := time.Now()
	traffic, err := accessHistoryTrafficOverview()
	if err != nil {
		return dashboardOverviewData{}, err
	}
	accounts, err := dashboardAccounts(now)
	if err != nil {
		return dashboardOverviewData{}, err
	}
	certificate := dashboardCertificateSummary{}
	if info, certErr := loadCertificateInfo(); certErr != nil {
		certificate.Error = certErr.Error()
		certificate.Warning = true
	} else {
		certificate.CommonName = info.CommonName
		certificate.NotAfter = info.NotAfter.Format(time.RFC3339)
		certificate.DaysLeft = info.DaysLeft
		certificate.Warning = info.DaysLeft <= 30
	}
	state := trojan.ActiveState()
	return dashboardOverviewData{
		Service: dashboardServiceSummary{
			State:   state,
			Running: state == "active",
			Type:    trojan.Type(),
			Uptime:  trojan.UpTime(),
		},
		Accounts:    accounts,
		Certificate: certificate,
		Traffic:     traffic,
		Capacity:    monthCapacity(now, traffic.Month.Total),
		Network:     NetworkTrafficSnapshot(),
		UpdatedAt:   now.In(chinaLocation()).Format(time.RFC3339),
	}, nil
}

// DashboardOverview returns cached business statistics. Expensive journal collection runs
// independently every five minutes and is never triggered by a page refresh.
func DashboardOverview(force bool) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	dashboardOverviewCache.Lock()
	defer dashboardOverviewCache.Unlock()
	if !force && dashboardOverviewCache.Ready && time.Now().Before(dashboardOverviewCache.ExpiresAt) {
		responseBody.Data = dashboardOverviewCache.Data
		return &responseBody
	}
	data, err := buildDashboardOverview()
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	dashboardOverviewCache.Data = data
	dashboardOverviewCache.ExpiresAt = time.Now().Add(dashboardCacheTTL)
	dashboardOverviewCache.Ready = true
	responseBody.Data = data
	return &responseBody
}
