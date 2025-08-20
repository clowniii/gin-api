package wiki

import (
	"fmt"
	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type WikiHandler struct{ d Dependencies }

func NewWikiHandler(d Dependencies) *WikiHandler { return &WikiHandler{d: d} }

var (
	_onceErrCode sync.Once
	_errCodeResp gin.H
)

func (h *WikiHandler) ErrorCode(c *gin.Context) {
	_onceErrCode.Do(func() {
		codes := retcode.All()
		keys := make([]string, 0, len(codes))
		for k := range codes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		arr := make([]gin.H, 0, len(keys))
		for _, k := range keys {
			v := codes[k]
			arr = append(arr, gin.H{"en_code": k, "code": v.Code, "chinese": v.Message})
		}
		co := h.d.Config.AppMeta.Name + " " + h.d.Config.AppMeta.Version
		_errCodeResp = gin.H{"code": retcode.SUCCESS, "msg": "success", "data": gin.H{"data": arr, "co": co}}
	})
	c.Set("resp", _errCodeResp)
	c.Status(http.StatusOK)
}
func (h *WikiHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" form:"username"`
		Password string `json:"password" form:"password"`
	}
	if err := c.ShouldBind(&req); err != nil || req.Username == "" || req.Password == "" {
		c.Set("resp", gin.H{"code": retcode.LOGIN_ERROR, "msg": "AppId或AppSecret错误", "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	ttl := time.Duration(h.d.Config.Wiki.OnlineTimeSeconds) * time.Second
	if ttl <= 0 {
		ttl = 86400 * time.Second
	}
	info, err := h.d.Wiki.Login(c.Request.Context(), req.Username, req.Password, ttl)
	if err != nil {
		msg := err.Error()
		if msg == "当前应用已被封禁，请联系管理员" {
			c.Set("resp", gin.H{"code": retcode.LOGIN_ERROR, "msg": msg, "data": gin.H{}})
		} else {
			c.Set("resp", gin.H{"code": retcode.LOGIN_ERROR, "msg": "AppId或AppSecret错误", "data": gin.H{}})
		}
		c.Status(http.StatusOK)
		return
	}
	c.Set("resp", gin.H{"code": retcode.SUCCESS, "msg": "登录成功", "data": info})
	c.Status(http.StatusOK)
}
func (h *WikiHandler) GroupList(c *gin.Context) {
	ui, _ := c.Get("wiki_user")
	user := toWikiUserInfo(ui)
	list, err := h.d.Wiki.GroupList(c.Request.Context(), user)
	if err != nil {
		c.Set("resp", gin.H{"code": retcode.DB_READ_ERROR, "msg": err.Error(), "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	co := h.d.Config.AppMeta.Name + " " + h.d.Config.AppMeta.Version
	c.Set("resp", gin.H{"code": retcode.SUCCESS, "msg": "success", "data": gin.H{"data": list, "co": co}})
	c.Status(http.StatusOK)
}
func (h *WikiHandler) Detail(c *gin.Context) {
	hash := c.Query("hash")
	if hash == "" {
		c.Set("resp", gin.H{"code": retcode.NOT_EXISTS, "msg": "缺少必要参数", "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	m, err := h.d.Wiki.Detail(c.Request.Context(), hash, c.Request.Host)
	if err != nil {
		c.Set("resp", gin.H{"code": retcode.NOT_EXISTS, "msg": err.Error(), "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	m["co"] = h.d.Config.AppMeta.Name + " " + h.d.Config.AppMeta.Version
	c.Set("resp", gin.H{"code": retcode.SUCCESS, "msg": "success", "data": m})
	c.Status(http.StatusOK)
}
func (h *WikiHandler) Logout(c *gin.Context) {
	apiAuth := c.GetHeader("ApiAuth")
	if apiAuth == "" {
		apiAuth = c.GetHeader("Api-Auth")
	}
	apiAuth = strings.TrimSpace(apiAuth)
	ui, _ := c.Get("wiki_user")
	user := toWikiUserInfo(ui)
	h.d.Wiki.Logout(c.Request.Context(), apiAuth, user.ID)
	c.Set("resp", gin.H{"code": retcode.SUCCESS, "msg": "登出成功", "data": gin.H{}})
	c.Status(http.StatusOK)
}
func (h *WikiHandler) Search(c *gin.Context) {
	kw := strings.TrimSpace(c.Query("keyword"))
	if kw == "" {
		c.Set("resp", gin.H{"code": retcode.SUCCESS, "msg": "success", "data": gin.H{"list": []interface{}{}, "count": 0}})
		c.Status(http.StatusOK)
		return
	}
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	list := h.d.Wiki.Search(c.Request.Context(), kw, limit)
	c.Set("resp", gin.H{"code": retcode.SUCCESS, "msg": "success", "data": gin.H{"list": list, "count": len(list)}})
	c.Status(http.StatusOK)
}
func (h *WikiHandler) GroupHot(c *gin.Context) {
	limit := 10
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	groups := h.d.Wiki.HotGroups(c.Request.Context(), limit)
	c.Set("resp", gin.H{"code": retcode.SUCCESS, "msg": "success", "data": gin.H{"list": groups}})
	c.Status(http.StatusOK)
}
func (h *WikiHandler) Fields(c *gin.Context) {
	hash := strings.TrimSpace(c.Query("hash"))
	if hash == "" {
		c.Set("resp", gin.H{"code": retcode.NOT_EXISTS, "msg": "缺少必要参数", "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	info, err := h.d.Wiki.Fields(c.Request.Context(), hash)
	if err != nil {
		c.Set("resp", gin.H{"code": retcode.NOT_EXISTS, "msg": err.Error(), "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	c.Set("resp", gin.H{"code": retcode.SUCCESS, "msg": "success", "data": info})
	c.Status(http.StatusOK)
}
func (h *WikiHandler) AppInfo(c *gin.Context) {
	ui, _ := c.Get("wiki_user")
	user := toWikiUserInfo(ui)
	info, err := h.d.Wiki.AppInfo(c.Request.Context(), user)
	if err != nil {
		c.Set("resp", gin.H{"code": retcode.NOT_EXISTS, "msg": err.Error(), "data": gin.H{}})
		c.Status(http.StatusOK)
		return
	}
	c.Set("resp", gin.H{"code": retcode.SUCCESS, "msg": "success", "data": info})
	c.Status(http.StatusOK)
}
func (h *WikiHandler) DataType(c *gin.Context) {
	c.Set("resp", gin.H{"code": retcode.SUCCESS, "msg": "success", "data": gin.H{"dataType": h.d.Wiki.DataTypeMap()}})
	c.Status(http.StatusOK)
}

func toWikiUserInfo(v interface{}) service.WikiUserInfo {
	switch t := v.(type) {
	case service.WikiUserInfo:
		return t
	case map[string]interface{}:
		var u service.WikiUserInfo
		if id, ok := t["id"].(float64); ok {
			u.ID = int64(id)
		} else if id2, ok := t["id"].(int64); ok {
			u.ID = id2
		}
		if appID, ok := t["app_id"].(string); ok {
			u.AppID = appID
		} else if appID2, ok := t["app_id"].(float64); ok {
			u.AppID = fmt.Sprint(int64(appID2))
		}
		if s, ok := t["app_api_show"].(string); ok {
			u.AppAPIShow = s
		}
		return u
	default:
		return service.WikiUserInfo{}
	}
}
