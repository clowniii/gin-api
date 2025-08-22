package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-apiadmin/internal/domain/model"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type AdminMenuDAO struct{ DB *gorm.DB }

func NewAdminMenuDAO(db *gorm.DB) *AdminMenuDAO { return &AdminMenuDAO{DB: db} }

// tracer 获取 DAO 层 tracer
func (d *AdminMenuDAO) tracer() trace.Tracer { return otel.Tracer("dao.admin_menu") }

// ListMenus 获取菜单，可按关键词(title 模糊)；keywords 为空时返回全部
func (d *AdminMenuDAO) ListMenus(ctx context.Context, keywords string) ([]model.AdminMenu, error) {
	ctx, span := d.tracer().Start(ctx, "AdminMenuDAO.ListMenus")
	defer span.End()
	q := d.DB.WithContext(ctx).Model(&model.AdminMenu{})
	if keywords != "" {
		q = q.Where("title ILIKE ?", "%"+keywords+"%")
	}
	var list []model.AdminMenu
	if err := q.Order("sort ASC").Find(&list).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("list menus: %w", err)
	}
	return list, nil
}

func (d *AdminMenuDAO) Create(ctx context.Context, m *model.AdminMenu) error {
	ctx, span := d.tracer().Start(ctx, "AdminMenuDAO.Create")
	defer span.End()
	if err := d.DB.WithContext(ctx).Create(m).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("create menu: %w", err)
	}
	return nil
}
func (d *AdminMenuDAO) Update(ctx context.Context, m *model.AdminMenu) error {
	ctx, span := d.tracer().Start(ctx, "AdminMenuDAO.Update")
	defer span.End()
	if err := d.DB.WithContext(ctx).Model(&model.AdminMenu{}).Where("id=?", m.ID).Updates(m).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("update menu id=%d: %w", m.ID, err)
	}
	return nil
}
func (d *AdminMenuDAO) Delete(ctx context.Context, id int64) error {
	ctx, span := d.tracer().Start(ctx, "AdminMenuDAO.Delete")
	defer span.End()
	if err := d.DB.WithContext(ctx).Delete(&model.AdminMenu{}, id).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("delete menu id=%d: %w", id, err)
	}
	return nil
}
func (d *AdminMenuDAO) UpdateShow(ctx context.Context, id int64, show int) error {
	ctx, span := d.tracer().Start(ctx, "AdminMenuDAO.UpdateShow")
	defer span.End()
	if err := d.DB.WithContext(ctx).Model(&model.AdminMenu{}).Where("id=?", id).Update("show", show).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("update show id=%d: %w", id, err)
	}
	return nil
}

// BuildTree 构建树结构（补充 auth/log/permission/method 字段，便于前端与过滤逻辑使用）
func BuildTree(list []model.AdminMenu) []map[string]interface{} {
	children := map[int64][]map[string]interface{}{}
	items := make([]map[string]interface{}, 0, len(list))
	for _, m := range list {
		item := map[string]interface{}{
			"id": m.ID, "fid": m.FID, "title": m.Title, "icon": m.Icon,
			"url": m.URL, "router": m.Router, "component": m.Component,
			"sort": m.Sort, "show": m.Show, "level": m.Level,
			"auth": m.Auth, "log": m.Log, "permission": m.Permission, "method": m.Method,
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
