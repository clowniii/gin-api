package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/metrics"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"
)

type WikiService struct {
	AppDAO    *dao.AdminAppDAO
	GroupDAO  *dao.AdminGroupDAO
	ListDAO   *dao.AdminInterfaceListDAO
	FieldsDAO *dao.AdminFieldsDAO
	Cache     cache.Cache // 使用统一 Cache 接口
}

// WikiLoginResult 登录返回结果
type WikiLoginResult struct {
	ID         int64  `json:"id"`
	AppID      string `json:"app_id"`
	AppSecret  string `json:"app_secret"`
	AppName    string `json:"app_name"`
	AppStatus  int8   `json:"app_status"`
	AppInfo    string `json:"app_info"`
	AppAPI     string `json:"app_api"`
	AppGroup   string `json:"app_group"`
	AppAPIShow string `json:"app_api_show"`
	ApiAuth    string `json:"apiAuth"`
}

// WikiUserInfo 用户信息
type WikiUserInfo struct {
	ID         int64  `json:"id"`
	AppID      string `json:"app_id"` // -1 表示后台登录
	AppAPIShow string `json:"app_api_show"`
}

func NewWikiService(app *dao.AdminAppDAO, grp *dao.AdminGroupDAO, list *dao.AdminInterfaceListDAO, fields *dao.AdminFieldsDAO, c cache.Cache) *WikiService {
	return &WikiService{AppDAO: app, GroupDAO: grp, ListDAO: list, FieldsDAO: fields, Cache: c}
}

// 增加缓存 key 前缀常量
const (
	cachePrefixSearch   = "wiki:search:"
	cachePrefixHotGroup = "wiki:hotgroups:"
	cachePrefixFields   = "wiki:fields:"
	cachePrefixAppInfo  = "wiki:appinfo:"
)

// NewWikiWithCache 语义化构造函数，等价于 NewWikiService
func NewWikiWithCache(app *dao.AdminAppDAO, grp *dao.AdminGroupDAO, list *dao.AdminInterfaceListDAO, fields *dao.AdminFieldsDAO, c cache.Cache) *WikiService {
	return NewWikiService(app, grp, list, fields, c)
}

func (s *WikiService) Login(ctx context.Context, appId, appSecret string, ttl time.Duration) (*WikiLoginResult, error) {
	if strings.TrimSpace(appId) == "" || strings.TrimSpace(appSecret) == "" {
		return nil, errors.New("AppId或AppSecret错误")
	}
	app, err := s.AppDAO.FindByAppIDAndSecret(ctx, appId, appSecret)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errors.New("AppId或AppSecret错误")
	}
	if app.AppStatus == 0 {
		return nil, errors.New("当前应用已被封禁，请联系管理员")
	}
	token := generateToken()
	info := WikiLoginResult{ID: app.ID, AppID: app.AppID, AppSecret: app.AppSecret, AppName: app.AppName, AppStatus: app.AppStatus, AppInfo: app.AppInfo, AppAPI: app.AppAPI, AppGroup: app.AppGroup, AppAPIShow: app.AppAPIShow, ApiAuth: token}
	b, _ := json.Marshal(info)
	_ = s.Cache.SetEX(ctx, "WikiLogin:"+token, string(b), ttl)
	_ = s.Cache.SetEX(ctx, "WikiLogin:"+intToStr(app.ID), token, ttl)
	return &info, nil
}

func (s *WikiService) Logout(ctx context.Context, token string, uid int64) {
	_ = s.Cache.Del(ctx, "WikiLogin:"+token, "WikiLogin:"+intToStr(uid))
}

// GroupList 根据 appInfo 构建分组+接口
func (s *WikiService) GroupList(ctx context.Context, appInfo WikiUserInfo) ([]map[string]interface{}, error) {
	groups, err := s.GroupDAO.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	apis, err := s.ListDAO.ListAllActive(ctx)
	if err != nil {
		return nil, err
	}
	// hash -> api slice
	apiByGroup := map[string][]model.AdminInterfaceList{}
	for _, a := range apis {
		apiByGroup[a.GroupHash] = append(apiByGroup[a.GroupHash], a)
	}
	var res []map[string]interface{}
	if appInfo.AppID == "-1" { // 后台用户全部
		for _, g := range groups {
			item := map[string]interface{}{"id": g.ID, "name": g.Name, "description": g.Description, "status": g.Status, "hash": g.Hash, "hot": g.Hot}
			if apiSlice, ok := apiByGroup[g.Hash]; ok {
				item["api_info"] = apiSlice
			}
			res = append(res, item)
		}
	} else {
		// 解析 app_api_show JSON: { groupHash: [apiHash,...] }
		show := parseAPIShow(appInfo.AppAPIShow)
		// 构造 map: hash->api
		apiMap := map[string]model.AdminInterfaceList{}
		for _, a := range apis {
			apiMap[a.Hash] = a
		}
		groupMap := map[string]model.AdminGroup{}
		for _, g := range groups {
			groupMap[g.Hash] = g
		}
		for gh, list := range show {
			g, ok := groupMap[gh]
			if !ok {
				continue
			}
			item := map[string]interface{}{"id": g.ID, "name": g.Name, "description": g.Description, "status": g.Status, "hash": g.Hash, "hot": g.Hot}
			var apiInfo []model.AdminInterfaceList
			for _, ah := range list {
				if api, ok2 := apiMap[ah]; ok2 {
					apiInfo = append(apiInfo, api)
				}
			}
			if len(apiInfo) > 0 {
				item["api_info"] = apiInfo
				res = append(res, item)
			}
		}
	}
	return res, nil
}

func (s *WikiService) Detail(ctx context.Context, hash string, domain string) (map[string]interface{}, error) {
	api, err := s.ListDAO.FindByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if api == nil {
		return nil, errors.New("接口hash非法")
	}
	reqFields, _ := s.FieldsDAO.ListByHashAndType(ctx, hash, 0)
	respFields, _ := s.FieldsDAO.ListByHashAndType(ctx, hash, 1)
	_ = s.GroupDAO.IncrHot(ctx, api.GroupHash)
	url := domain + "/api/" + func() string {
		if api.HashType == 1 {
			return api.APIClass
		} else {
			return api.Hash
		}
	}()
	dataType := map[int]string{0: "Integer", 1: "String", 2: "Boolean", 3: "Enum", 4: "Float", 5: "File", 6: "Array", 7: "Object", 8: "Mobile"}
	return map[string]interface{}{
		"request":  reqFields,
		"response": respFields,
		"dataType": dataType,
		"apiList":  api,
		"url":      url,
	}, nil
}

// Search 按关键字搜索接口(APIClass / Info 模糊匹配)，限制条数（增加缓存）
func (s *WikiService) Search(ctx context.Context, keyword string, limit int) []map[string]interface{} {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" || limit <= 0 {
		return nil
	}
	ckey := cachePrefixSearch + keyword + ":" + intToStr(int64(limit))
	if s.Cache != nil {
		if str, err := s.Cache.Get(ctx, ckey); err == nil && str != "" {
			if cache.IsNilSentinel(str) { // sentinel 命中
				metrics.CacheNilHit.Inc()
				return []map[string]interface{}{}
			}
			var cached []map[string]interface{}
			_ = json.Unmarshal([]byte(str), &cached)
			if len(cached) > 0 {
				return cached
			}
		}
	}
	list, err := s.ListDAO.ListAllActive(ctx)
	if err != nil {
		return nil
	}
	res := make([]map[string]interface{}, 0, limit)
	for _, v := range list {
		if len(res) >= limit {
			break
		}
		lc := strings.ToLower(v.APIClass)
		li := strings.ToLower(v.Info)
		if strings.Contains(lc, keyword) || strings.Contains(li, keyword) {
			res = append(res, map[string]interface{}{
				"id": v.ID, "api_class": v.APIClass, "hash": v.Hash, "info": v.Info, "group_hash": v.GroupHash, "status": v.Status,
			})
		}
	}
	if s.Cache != nil {
		if len(res) == 0 { // 空结果防穿透
			_ = s.Cache.SetEX(ctx, ckey, cache.WrapNil(true), 10*time.Second)
			return res
		}
		b, _ := json.Marshal(res)
		_ = s.Cache.SetEX(ctx, ckey, string(b), 30*time.Second)
	}
	return res
}

// HotGroups 返回最热分组（按 Hot 值倒序）（增加缓存）
func (s *WikiService) HotGroups(ctx context.Context, limit int) []map[string]interface{} {
	if limit <= 0 {
		return nil
	}
	ckey := cachePrefixHotGroup + intToStr(int64(limit))
	if s.Cache != nil {
		if str, err := s.Cache.Get(ctx, ckey); err == nil && str != "" {
			if cache.IsNilSentinel(str) {
				metrics.CacheNilHit.Inc()
				return []map[string]interface{}{}
			}
			var cached []map[string]interface{}
			_ = json.Unmarshal([]byte(str), &cached)
			if len(cached) > 0 {
				return cached
			}
		}
	}
	groups, err := s.GroupDAO.ListAll(ctx)
	if err != nil {
		return nil
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Hot > groups[j].Hot })
	if len(groups) > limit {
		groups = groups[:limit]
	}
	res := make([]map[string]interface{}, 0, len(groups))
	for _, g := range groups {
		res = append(res, map[string]interface{}{"id": g.ID, "name": g.Name, "description": g.Description, "hot": g.Hot, "hash": g.Hash})
	}
	if s.Cache != nil {
		if len(res) == 0 { // 空结果 sentinel
			_ = s.Cache.SetEX(ctx, ckey, cache.WrapNil(true), 10*time.Second)
			return res
		}
		b, _ := json.Marshal(res)
		_ = s.Cache.SetEX(ctx, ckey, string(b), 60*time.Second)
	}
	return res
}

// Fields 获取请求与响应字段（增加缓存）
func (s *WikiService) Fields(ctx context.Context, hash string) (map[string]interface{}, error) {
	if strings.TrimSpace(hash) == "" {
		return nil, errors.New("hash required")
	}
	ckey := cachePrefixFields + hash
	if s.Cache != nil {
		if str, err := s.Cache.Get(ctx, ckey); err == nil && str != "" {
			var m map[string]interface{}
			_ = json.Unmarshal([]byte(str), &m)
			if len(m) > 0 {
				return m, nil
			}
		}
	}
	req, err := s.FieldsDAO.ListByHashAndType(ctx, hash, 0)
	if err != nil {
		return nil, err
	}
	resp, err := s.FieldsDAO.ListByHashAndType(ctx, hash, 1)
	if err != nil {
		return nil, err
	}
	res := map[string]interface{}{"request": req, "response": resp, "dataType": s.DataTypeMap()}
	if s.Cache != nil {
		b, _ := json.Marshal(res)
		_ = s.Cache.SetEX(ctx, ckey, string(b), 120*time.Second)
	}
	return res, nil
}

// AppInfo 获取当前登录应用信息（后台登录 app_id=-1 则返回空）（增加缓存）
func (s *WikiService) AppInfo(ctx context.Context, user WikiUserInfo) (map[string]interface{}, error) {
	if user.AppID == "-1" {
		return map[string]interface{}{"app_id": "-1"}, nil
	}
	ckey := cachePrefixAppInfo + intToStr(user.ID)
	if s.Cache != nil {
		if str, err := s.Cache.Get(ctx, ckey); err == nil && str != "" {
			var m map[string]interface{}
			_ = json.Unmarshal([]byte(str), &m)
			if len(m) > 0 {
				return m, nil
			}
		}
	}
	app, err := s.AppDAO.FindByID(ctx, user.ID)
	if err != nil || app == nil {
		return nil, errors.New("应用不存在")
	}
	res := map[string]interface{}{"id": app.ID, "app_id": app.AppID, "app_name": app.AppName, "app_info": app.AppInfo, "app_group": app.AppGroup}
	if s.Cache != nil {
		b, _ := json.Marshal(res)
		_ = s.Cache.SetEX(ctx, ckey, string(b), 300*time.Second)
	}
	return res, nil
}

// DataTypeMap 返回字段数据类型映射
func (s *WikiService) DataTypeMap() map[int]string {
	return map[int]string{0: "Integer", 1: "String", 2: "Boolean", 3: "Enum", 4: "Float", 5: "File", 6: "Array", 7: "Object", 8: "Mobile"}
}

// ===== util helpers =====
func generateToken() string { b := make([]byte, 16); rand.Read(b); return hex.EncodeToString(b) }

func parseAPIShow(s string) map[string][]string {
	if s == "" {
		return map[string][]string{}
	}
	var m map[string][]string
	_ = json.Unmarshal([]byte(s), &m)
	if m == nil {
		m = map[string][]string{}
	}
	return m
}

func intToStr(i int64) string { return strconv.FormatInt(i, 10) }
func RandString(n int) string { b := make([]byte, n); rand.Read(b); return hex.EncodeToString(b)[:n] }
