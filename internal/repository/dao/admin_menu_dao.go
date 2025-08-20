package dao

import (
	"context"
	"encoding/json"
	"strings"

	"go-apiadmin/internal/domain/model"

	"gorm.io/gorm"
)

type AdminMenuDAO struct{ DB *gorm.DB }

func NewAdminMenuDAO(db *gorm.DB) *AdminMenuDAO { return &AdminMenuDAO{DB: db} }

// ListMenus 获取菜单，可按关键词(title 模糊)；keywords 为空时返回全部
func (d *AdminMenuDAO) ListMenus(ctx context.Context, keywords string) ([]model.AdminMenu, error) {
	q := d.DB.WithContext(ctx).Model(&model.AdminMenu{})
	if keywords != "" {
		q = q.Where("title ILIKE ?", "%"+keywords+"%")
	}
	var list []model.AdminMenu
	if err := q.Order("sort ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (d *AdminMenuDAO) Create(ctx context.Context, m *model.AdminMenu) error {
	return d.DB.WithContext(ctx).Create(m).Error
}
func (d *AdminMenuDAO) Update(ctx context.Context, m *model.AdminMenu) error {
	return d.DB.WithContext(ctx).Model(&model.AdminMenu{}).Where("id=?", m.ID).Updates(m).Error
}
func (d *AdminMenuDAO) Delete(ctx context.Context, id int64) error {
	return d.DB.WithContext(ctx).Delete(&model.AdminMenu{}, id).Error
}
func (d *AdminMenuDAO) UpdateShow(ctx context.Context, id int64, show int) error {
	return d.DB.WithContext(ctx).Model(&model.AdminMenu{}).Where("id=?", id).Update("show", show).Error
}

// BuildTree 构建树结构
func BuildTree(list []model.AdminMenu) []map[string]interface{} {
	children := map[int64][]map[string]interface{}{}
	items := make([]map[string]interface{}, 0, len(list))
	for _, m := range list {
		item := map[string]interface{}{
			"id": m.ID, "fid": m.FID, "title": m.Title, "icon": m.Icon,
			"url": m.URL, "router": m.Router, "component": m.Component,
			"sort": m.Sort, "show": m.Show, "level": m.Level,
		}
		items = append(items, item)
		children[m.FID] = append(children[m.FID], item)
	}
	var attach func(node map[string]interface{})
	attach = func(node map[string]interface{}) {
		id := node["id"].(int64)
		if ch, ok := children[id]; ok {
			for _, c := range ch {
				attach(c)
			}
			node["children"] = ch
		}
	}
	var roots []map[string]interface{}
	for _, it := range items {
		if it["fid"].(int64) == 0 {
			attach(it)
			roots = append(roots, it)
		}
	}
	return roots
}

// NormalizeURL 确保 url 前缀
func NormalizeURL(u string) string {
	if u == "" {
		return u
	}
	if !strings.HasPrefix(strings.ToLower(u), "admin/") {
		return "admin/" + u
	}
	return u
}

// BuildTreeCachedJSON 将缓存中的 JSON 数组(菜单树) 反序列化
// 当缓存损坏或为空时返回 nil 让上层回源
func BuildTreeCachedJSON(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil
	}
	return v
}
