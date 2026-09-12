package controller

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"trojan/core"
)

type certInfo struct {
	CertPath     string    `json:"certPath"`
	KeyPath      string    `json:"keyPath"`
	Subject      string    `json:"subject"`
	CommonName   string    `json:"commonName"`
	Issuer       string    `json:"issuer"`
	NotBefore    time.Time `json:"notBefore"`
	NotAfter     time.Time `json:"notAfter"`
	DNSNames     []string  `json:"dnsNames"`
	IPAddresses  []string  `json:"ipAddresses"`
	SerialNumber string    `json:"serialNumber"`
	DaysLeft     int       `json:"daysLeft"`
}

func loadCertificateInfo() (*certInfo, error) {
	config := core.GetConfig()
	if config == nil {
		return nil, errors.New("load config failed")
	}
	data, err := os.ReadFile(config.SSl.Cert)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid certificate pem")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	ipList := make([]string, 0, len(cert.IPAddresses))
	for _, ip := range cert.IPAddresses {
		ipList = append(ipList, ip.String())
	}
	return &certInfo{
		CertPath:     config.SSl.Cert,
		KeyPath:      config.SSl.Key,
		Subject:      cert.Subject.String(),
		CommonName:   cert.Subject.CommonName,
		Issuer:       cert.Issuer.String(),
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		DNSNames:     cert.DNSNames,
		IPAddresses:  ipList,
		SerialNumber: cert.SerialNumber.String(),
		DaysLeft:     int(time.Until(cert.NotAfter).Hours() / 24),
	}, nil
}

func primaryCertDomain(info *certInfo) string {
	if len(info.DNSNames) > 0 {
		return info.DNSNames[0]
	}
	for _, ip := range info.IPAddresses {
		if ip != "" {
			return ip
		}
	}
	return info.CommonName
}

// CertInfo 获取当前TLS证书详情
func CertInfo() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	info, err := loadCertificateInfo()
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	responseBody.Data = info
	return &responseBody
}

// RenewCert 触发证书续签
func RenewCert() *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}
	defer TimeCost(time.Now(), &responseBody)
	if _, err := os.Stat("/root/.acme.sh/acme.sh"); err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	info, err := loadCertificateInfo()
	if err != nil {
		responseBody.Msg = err.Error()
		return &responseBody
	}
	domain := primaryCertDomain(info)
	if domain == "" {
		responseBody.Msg = "certificate domain is empty"
		return &responseBody
	}
	eccArg := ""
	if strings.Contains(info.CertPath, "_ecc") || strings.Contains(info.KeyPath, "_ecc") {
		eccArg = " --ecc"
	}
	unit := fmt.Sprintf("trojan-cert-renew-%d", time.Now().Unix())
	script := fmt.Sprintf(`exec >/tmp/trojan-cert-renew.log 2>&1
set -u
echo "[$$(date -Is)] certificate renewal started"
sleep 1

renew_status=0
restart_status=0
web_status=0
domain=%s

systemctl stop trojan-web
/root/.acme.sh/acme.sh --renew -d "$${domain}"%s --force --home /root/.acme.sh || renew_status=$$?
systemctl restart trojan || restart_status=$$?
systemctl start trojan-web || web_status=$$?

echo "[$$(date -Is)] certificate renewal finished: renew=$${renew_status} trojan=$${restart_status} web=$${web_status}"
if [ "$${renew_status}" -ne 0 ]; then exit "$${renew_status}"; fi
if [ "$${restart_status}" -ne 0 ]; then exit "$${restart_status}"; fi
exit "$${web_status}"`, strconv.Quote(domain), eccArg)
	cmd := exec.Command("systemd-run", "--unit", unit, "--description", "trojan certificate renewal", "--collect", "/bin/bash", "-lc", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		responseBody.Msg = strings.TrimSpace(string(out))
		if responseBody.Msg == "" {
			responseBody.Msg = err.Error()
		}
		return &responseBody
	}
	responseBody.Data = map[string]string{
		"domain": domain,
		"log":    "/tmp/trojan-cert-renew.log",
		"unit":   unit,
	}
	return &responseBody
}
