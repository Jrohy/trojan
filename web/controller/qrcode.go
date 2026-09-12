package controller

import (
	"net/http"
	"os/exec"

	"github.com/gin-gonic/gin"
)

// QRCode 生成分享二维码PNG
func QRCode(c *gin.Context) {
	data := c.Query("data")
	if data == "" {
		c.String(http.StatusBadRequest, "data is empty")
		return
	}
	cmd := exec.Command("qrencode", "-t", "PNG", "-o", "-", "-s", "6", "-m", "2", data)
	out, err := cmd.Output()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "image/png", out)
}
