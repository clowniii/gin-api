package admin

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// 复制通用工具 (与原 handler/util.go 保持一致, 后续可公共化)

func qInt(c *gin.Context, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}
	return def
}
func qInt64(c *gin.Context, key string) int64 {
	v := c.Query(key)
	if v == "" {
		return 0
	}
	i, _ := strconv.ParseInt(v, 10, 64)
	return i
}
func qInt8Ptr(c *gin.Context, key string) *int8 {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	iv, err := strconv.ParseInt(v, 10, 8)
	if err != nil {
		return nil
	}
	vv := int8(iv)
	return &vv
}
func int8Ptr(v int8) *int8                { return &v }
func pageLimit(c *gin.Context) (int, int) { return qInt(c, "page", 1), qInt(c, "limit", 20) }
