package admin

import (
	"go-apiadmin/internal/pkg/cache"
	"net/http"

	"github.com/gin-gonic/gin"
)

type CacheHandler struct{ d Dependencies }

func NewCacheHandler(d Dependencies) *CacheHandler { return &CacheHandler{d: d} }

func (h *CacheHandler) Metrics(c *gin.Context) {
	t := c.Query("type")
	if t == "perm" {
		m := h.d.Perm.SnapshotMetrics()
		c.Set("resp", gin.H{"code": 0, "msg": "success", "data": gin.H{"perm": m}})
		c.Status(http.StatusOK)
		return
	}
	lc, ok := h.d.Cache.(*cache.LayeredCache)
	layeredMetrics := interface{}(gin.H{})
	if ok && lc != nil {
		layeredMetrics = lc.SnapshotMetrics()
	}
	if t == "all" {
		permM := h.d.Perm.SnapshotMetrics()
		c.Set("resp", gin.H{"code": 0, "msg": "success", "data": gin.H{"layered": layeredMetrics, "perm": permM}})
		c.Status(http.StatusOK)
		return
	}
	c.Set("resp", gin.H{"code": 0, "msg": "success", "data": gin.H{"layered": layeredMetrics}})
	c.Status(http.StatusOK)
}
func (h *CacheHandler) Reset(c *gin.Context) {
	t := c.Query("type")
	if t == "perm" {
		h.d.Perm.ResetMetrics()
		c.Set("resp", gin.H{"code": 0, "msg": "success", "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	if t == "all" {
		if lc, ok := h.d.Cache.(*cache.LayeredCache); ok && lc != nil {
			lc.ResetMetrics()
		}
		h.d.Perm.ResetMetrics()
		c.Set("resp", gin.H{"code": 0, "msg": "success", "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	if lc, ok := h.d.Cache.(*cache.LayeredCache); ok && lc != nil {
		lc.ResetMetrics()
	}
	c.Set("resp", gin.H{"code": 0, "msg": "success", "data": gin.H{}})
	c.Status(http.StatusOK)
}
