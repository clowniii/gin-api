package admin

import (
	"encoding/json"
	"strings"
	"time"

	"go-apiadmin/internal/metrics"
	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct{ d Dependencies }

func NewAuthHandler(d Dependencies) *AuthHandler { return &AuthHandler{d: d} }

func (h *AuthHandler) Login(c *gin.Context) {
	start := time.Now()
	var req struct{ Username, Password string }
	if err := c.ShouldBindJSON(&req); err != nil {
		metrics.AuthActionTotal.WithLabelValues("login", "parse_error").Inc()
		metrics.AuthActionDuration.WithLabelValues("login", "parse_error").Observe(time.Since(start).Seconds())
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	access, refresh, err := h.d.Auth.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		metrics.AuthActionTotal.WithLabelValues("login", "error").Inc()
		metrics.AuthActionDuration.WithLabelValues("login", "error").Observe(time.Since(start).Seconds())
		response.Error(c, retcode.LOGIN_ERROR, err.Error())
		return
	}
	var uid int64
	if claims, perr := h.d.JWT.Parse(access); perr == nil {
		uid = claims.UserID
	}
	userInfo, _ := h.d.User.GetUserInfo(c.Request.Context(), uid)
	menus, _ := h.d.Menu.AccessMenu(c.Request.Context(), uid)
	permSet, _ := h.d.Perm.GetUserPermissions(c.Request.Context(), uid)
	perms := make([]string, 0, len(permSet)) // 确保非 nil
	for p := range permSet {
		perms = append(perms, p)
	}
	// 普通用户过滤菜单（超级管理员 uid=1 保留全部）
	if uid != 1 {
		menus = filterMenuTree(menus, permSet)
	}
	resp := gin.H{"token": access, "refreshToken": refresh, "user": userInfo, "menu": menus, "access": perms, "perms": perms}
	if h.d.Cache != nil && uid > 0 {
		b, _ := json.Marshal(resp)
		ttl := time.Duration(h.d.Config.Auth.SessionTTLSeconds) * time.Second
		_ = h.d.Cache.SetEX(c.Request.Context(), h.sessionKey(uid), string(b), ttl)
		metrics.AuthSessionCacheSet.WithLabelValues("login").Inc()
	}
	metrics.AuthActionTotal.WithLabelValues("login", "success").Inc()
	metrics.AuthActionDuration.WithLabelValues("login", "success").Observe(time.Since(start).Seconds())
	response.Success(c, resp)
}

// Refresh 根据 refreshToken 生成新的 token 与新的/旧的 refreshToken（旋转）并返回最新用户信息
func (h *AuthHandler) Refresh(c *gin.Context) {
	start := time.Now()
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		metrics.AuthActionTotal.WithLabelValues("refresh", "parse_error").Inc()
		metrics.AuthActionDuration.WithLabelValues("refresh", "parse_error").Observe(time.Since(start).Seconds())
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	if req.RefreshToken == "" {
		metrics.AuthActionTotal.WithLabelValues("refresh", "missing").Inc()
		metrics.AuthActionDuration.WithLabelValues("refresh", "missing").Observe(time.Since(start).Seconds())
		response.Error(c, retcode.AUTH_ERROR, "missing refreshToken")
		return
	}
	access, newRefresh, uid, err := h.d.Auth.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		metrics.AuthActionTotal.WithLabelValues("refresh", "error").Inc()
		metrics.AuthActionDuration.WithLabelValues("refresh", "error").Observe(time.Since(start).Seconds())
		response.Error(c, retcode.AUTH_ERROR, err.Error())
		return
	}
	var respMap map[string]interface{}
	if h.d.Cache != nil && uid > 0 { // 尝试命中
		if v, _ := h.d.Cache.Get(c.Request.Context(), h.sessionKey(uid)); v != "" {
			_ = json.Unmarshal([]byte(v), &respMap)
			if respMap != nil {
				migrateAccessSlice(respMap)
				metrics.AuthSessionCacheHit.WithLabelValues("refresh").Inc()
				// 如果是普通用户且缓存中 menu 未过滤（缺少过滤信息无法准确判断），简单重新过滤一次
				if uid != 1 {
					// 重建 perms 集合
					permSet, _ := h.d.Perm.GetUserPermissions(c.Request.Context(), uid)
					if raw, ok := respMap["menu"].([]map[string]interface{}); ok {
						respMap["menu"] = filterMenuTree(raw, permSet)
					}
				}
			}
		}
	}
	if respMap == nil { // 未命中回源
		userInfo, _ := h.d.User.GetUserInfo(c.Request.Context(), uid)
		menus, _ := h.d.Menu.AccessMenu(c.Request.Context(), uid)
		permSet, _ := h.d.Perm.GetUserPermissions(c.Request.Context(), uid)
		perms := make([]string, 0, len(permSet))
		for p := range permSet {
			perms = append(perms, p)
		}
		if uid != 1 {
			menus = filterMenuTree(menus, permSet)
		}
		respMap = map[string]interface{}{"user": userInfo, "menu": menus, "access": perms, "perms": perms}
	} else {
		if a, ok := respMap["access"]; ok {
			if _, hasPerms := respMap["perms"]; !hasPerms {
				respMap["perms"] = a
			}
		}
		migrateAccessSlice(respMap)
	}
	respMap["token"] = access
	respMap["refreshToken"] = newRefresh
	if h.d.Cache != nil && uid > 0 {
		b, _ := json.Marshal(respMap)
		_ = h.d.Cache.SetEX(c.Request.Context(), h.sessionKey(uid), string(b), time.Duration(h.d.Config.Auth.SessionTTLSeconds)*time.Second)
		metrics.AuthSessionCacheSet.WithLabelValues("refresh").Inc()
	}
	metrics.AuthActionTotal.WithLabelValues("refresh", "success").Inc()
	metrics.AuthActionDuration.WithLabelValues("refresh", "success").Observe(time.Since(start).Seconds())
	response.Success(c, respMap)
}

func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	uid := c.GetInt64("user_id")
	if uid <= 0 {
		response.Error(c, retcode.AUTH_ERROR, "invalid uid")
		return
	}
	if h.d.Cache != nil { // 缓存命中直接返回
		if v, _ := h.d.Cache.Get(c.Request.Context(), h.sessionKey(uid)); v != "" {
			var data map[string]interface{}
			if json.Unmarshal([]byte(v), &data) == nil {
				migrateAccessSlice(data)
				metrics.AuthSessionCacheHit.WithLabelValues("userinfo").Inc()
				response.Success(c, data)
				return
			}
		}
	}
	userInfo, err := h.d.User.GetUserInfo(c.Request.Context(), uid)
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	menus, _ := h.d.Menu.AccessMenu(c.Request.Context(), uid)
	permSet, _ := h.d.Perm.GetUserPermissions(c.Request.Context(), uid)
	if uid != 1 {
		menus = filterMenuTree(menus, permSet)
	}
	perms := make([]string, 0, len(permSet))
	for p := range permSet {
		perms = append(perms, p)
	}
	resp := gin.H{"user": userInfo, "menu": menus, "access": perms, "perms": perms}
	if h.d.Cache != nil {
		b, _ := json.Marshal(resp)
		_ = h.d.Cache.SetEX(c.Request.Context(), h.sessionKey(uid), string(b), time.Duration(h.d.Config.Auth.SessionTTLSeconds)*time.Second)
		metrics.AuthSessionCacheSet.WithLabelValues("userinfo").Inc()
	}
	response.Success(c, resp)
}

func (h *AuthHandler) GetAccessMenu(c *gin.Context) {
	uid := c.GetInt64("user_id")
	menus, err := h.d.Menu.AccessMenu(c.Request.Context(), uid)
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	permSet, _ := h.d.Perm.GetUserPermissions(c.Request.Context(), uid)
	filtered := menus
	if uid != 1 {
		filtered = filterMenuTree(menus, permSet)
	}
	response.Success(c, filtered)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if len(auth) < 8 {
		response.Error(c, retcode.AUTH_ERROR, "missing token")
		return
	}
	token := auth[7:]
	claims, err := h.d.JWT.Parse(token)
	if err != nil {
		response.Error(c, retcode.AUTH_ERROR, "invalid token")
		return
	}
	_ = h.d.Auth.Logout(c.Request.Context(), claims.JTI)
	response.Success(c, gin.H{"ok": true})
}

func (h *AuthHandler) DelMember(c *gin.Context) {
	gid := qInt64(c, "gid")
	uid := qInt64(c, "uid")
	if gid <= 0 || uid <= 0 {
		response.Error(c, retcode.EMPTY_PARAMS, "缺少必要参数")
		return
	}
	if err := h.d.AuthGroup.DeleteMember(c.Request.Context(), gid, uid); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// GetGroups 返回全部启用状态的组
func (h *AuthHandler) GetGroups(c *gin.Context) {
	res, err := h.d.AuthGroup.List(c.Request.Context())
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	var list []interface{}
	if res != nil {
		for _, g := range res.List {
			if g.Status == 1 {
				list = append(list, g)
			}
		}
	}
	response.Success(c, gin.H{"list": list, "count": len(list)})
}

// GetRuleList 获取组的规则树
func (h *AuthHandler) GetRuleList(c *gin.Context) {
	gid := qInt64(c, "group_id")
	menus, err := h.d.Menu.List(c.Request.Context(), "")
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	var ruleSet map[string]struct{}
	if gid > 0 {
		rules, err := h.d.AuthRule.List(c.Request.Context(), service.ListRuleParams{GroupID: &gid})
		if err != nil {
			response.Error(c, retcode.DB_READ_ERROR, err.Error())
			return
		}
		ruleSet = make(map[string]struct{}, len(rules.List))
		for _, r := range rules.List {
			ruleSet[r.URL] = struct{}{}
		}
	}
	list := buildRuleTreeFromMenu(menus.List, ruleSet)
	response.Success(c, gin.H{"list": list})
}

func buildRuleTreeFromMenu(src interface{}, ruleSet map[string]struct{}) []map[string]interface{} {
	nodes, _ := src.([]map[string]interface{})
	var dfs func([]map[string]interface{}) []map[string]interface{}
	dfs = func(arr []map[string]interface{}) []map[string]interface{} {
		res := make([]map[string]interface{}, 0, len(arr))
		for _, n := range arr {
			m := map[string]interface{}{"title": n["title"], "key": n["url"]}
			if ch, ok := n["children"].([]map[string]interface{}); ok && len(ch) > 0 {
				m["expand"] = true
				m["children"] = dfs(ch)
			} else if ruleSet != nil {
				if u, _ := n["url"].(string); u != "" {
					if _, ok2 := ruleSet[u]; ok2 {
						m["checked"] = true
					}
				}
			}
			res = append(res, m)
		}
		return res
	}
	return dfs(nodes)
}

func (h *AuthHandler) EditRule(c *gin.Context) {
	var req struct {
		ID    int64    `json:"id" form:"id"`
		Rules []string `json:"rules" form:"rules[]"`
	}
	if err := c.ShouldBind(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "JSON数据格式错误")
		return
	}
	if req.ID <= 0 {
		response.Error(c, retcode.EMPTY_PARAMS, "缺少必要参数")
		return
	}
	if err := h.d.AuthRule.BulkEditRules(c.Request.Context(), req.ID, req.Rules); err != nil {
		response.Error(c, retcode.DB_SAVE_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func filterMenuTree(nodes []map[string]interface{}, perms map[string]struct{}) []map[string]interface{} {
	var res []map[string]interface{}
	for _, n := range nodes {
		item := map[string]interface{}{}
		for k, v := range n {
			if k != "children" {
				item[k] = v
			}
		}
		var filteredChildren []map[string]interface{}
		if chRaw, ok := n["children"]; ok {
			if chSlice, ok2 := chRaw.([]map[string]interface{}); ok2 {
				filteredChildren = filterMenuTree(chSlice, perms)
			}
		}
		urlStr, _ := item["url"].(string)
		var showVal int64
		switch v := item["show"].(type) {
		case int:
			showVal = int64(v)
		case int8:
			showVal = int64(v)
		case int16:
			showVal = int64(v)
		case int32:
			showVal = int64(v)
		case int64:
			showVal = v
		case float64:
			showVal = int64(v)
		}
		allowed := false
		if urlStr != "" {
			if urlStr[0] != '/' {
				urlStr = "/" + urlStr
			}
			if _, ok := perms[strings.ToLower(urlStr)]; ok {
				allowed = true
			}
		}
		if len(filteredChildren) > 0 {
			item["children"] = filteredChildren
		}
		if allowed || len(filteredChildren) > 0 {
			if showVal == 1 || len(filteredChildren) > 0 {
				res = append(res, item)
			}
		}
	}
	return res
}

func (h *AuthHandler) sessionKey(uid int64) string {
	return "user:session:" + strconv.FormatInt(uid, 10)
}

func migrateAccessSlice(m map[string]interface{}) {
	// 如果 access 为 nil 或不是期望类型，尝试从 perms 构建
	if v, ok := m["access"]; !ok || v == nil {
		if p, ok2 := m["perms"]; ok2 {
			switch arr := p.(type) {
			case []string:
				if arr == nil {
					m["access"] = []string{}
				} else {
					m["access"] = arr
				}
			case []interface{}:
				res := make([]string, 0, len(arr))
				for _, it := range arr {
					if s, ok3 := it.(string); ok3 {
						res = append(res, s)
					}
				}
				m["access"] = res
			default:
				m["access"] = []string{}
			}
		} else {
			m["access"] = []string{}
		}
	}
	// 若 access 是 []interface{} 需要转换
	switch arr := m["access"].(type) {
	case []interface{}:
		res := make([]string, 0, len(arr))
		for _, it := range arr {
			if s, ok := it.(string); ok {
				res = append(res, s)
			}
		}
		m["access"] = res
	case nil:
		m["access"] = []string{}
	}
	// perms 也保证为非 nil 切片
	if v, ok := m["perms"]; !ok || v == nil {
		m["perms"] = m["access"]
	}
}
