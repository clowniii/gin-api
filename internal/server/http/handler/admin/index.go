package admin

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type IndexHandler struct{ d Dependencies }

func NewIndexHandler(d Dependencies) *IndexHandler { return &IndexHandler{d: d} }

func (h *IndexHandler) Upload(c *gin.Context) {
	f, err := c.FormFile("file")
	if err != nil {
		c.Set("resp", gin.H{"code": 40001, "msg": "缺少文件", "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	maxBytes := int64(h.d.Config.Upload.MaxSizeMB) * 1024 * 1024
	if maxBytes > 0 && f.Size > maxBytes {
		c.Set("resp", gin.H{"code": 40002, "msg": "文件过大", "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(f.Filename), "."))
	allowed := h.d.Config.Upload.AllowedExt
	if len(allowed) > 0 {
		ok := false
		for _, e := range allowed {
			if strings.ToLower(e) == ext {
				ok = true
				break
			}
		}
		if !ok {
			c.Set("resp", gin.H{"code": 40003, "msg": "不支持的文件类型", "data": gin.H{}})
			c.Status(http.StatusOK)
			return
		}
	}
	dir := filepath.Join("upload", time.Now().Format("20060102"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.Set("resp", gin.H{"code": 500, "msg": "创建目录失败", "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	newName := randomHex(12) + "_" + time.Now().Format("150405") + "." + ext
	fullPath := filepath.Join(dir, newName)
	if err := c.SaveUploadedFile(f, fullPath); err != nil {
		c.Set("resp", gin.H{"code": 500, "msg": "文件上传失败", "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	fileUrl := "/" + fullPath
	c.Set("resp", gin.H{"code": 0, "msg": "success", "data": gin.H{"fileName": newName, "fileUrl": fileUrl}})
	c.Status(http.StatusOK)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return time.Now().Format("150405")
	}
	return hex.EncodeToString(b)
}
