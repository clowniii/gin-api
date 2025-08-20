package admin

import (
	"go-apiadmin/internal/service"
	"go-apiadmin/internal/util/retcode"
	"go-apiadmin/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct{ d Dependencies }

func NewAuthHandler(d Dependencies) *AuthHandler { return &AuthHandler{d: d} }

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct{ Username, Password string }
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, retcode.JSON_PARSE_FAIL, "invalid body")
		return
	}
	token, err := h.d.Auth.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.Error(c, retcode.LOGIN_ERROR, err.Error())
		return
	}
	response.Success(c, gin.H{"token": token})
}

func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	uid := c.GetInt64("user_id")
	info, err := h.d.User.GetUserInfo(c.Request.Context(), uid)
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	response.Success(c, info)
}

func (h *AuthHandler) GetAccessMenu(c *gin.Context) {
	uid := c.GetInt64("user_id")
	menus, err := h.d.Menu.AccessMenu(c.Request.Context(), uid)
	if err != nil {
		response.Error(c, retcode.DB_READ_ERROR, err.Error())
		return
	}
	permSet, _ := h.d.Perm.GetUserPermissions(c.Request.Context(), uid)
	filtered := filterMenuTree(menus, permSet)
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

func parseIDParam(c *gin.Context, name string) int64 {
	v, _ := strconv.ParseInt(c.Query(name), 10, 64)
	return v
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
		showVal, _ := item["show"].(int)
		allowed := false
		if urlStr != "" {
			if urlStr[0] != '/' {
				urlStr = "/" + urlStr
			}
			if _, ok := perms[urlStr]; ok {
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
