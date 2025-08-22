package model

// AdminMenu 对应原 admin_menu 表（目录/菜单 + 按钮权限）
// 保持与旧库字段一致，便于无损迁移。

type AdminMenu struct {
	ID         int64  `gorm:"primaryKey;column:id" json:"id"`
	Title      string `gorm:"column:title;size:50" json:"title"`
	FID        int64  `gorm:"column:fid" json:"fid"`
	URL        string `gorm:"column:url;size:50" json:"url"`
	Auth       int8   `gorm:"column:auth" json:"auth"` // 是否需要登录
	Sort       int    `gorm:"column:sort" json:"sort"`
	Show       int8   `gorm:"column:show" json:"show"` // 是否显示
	Icon       string `gorm:"column:icon;size:50" json:"icon"`
	Level      int8   `gorm:"column:level" json:"level"` // 菜单层级 1/2/3
	Component  string `gorm:"column:component;size:255" json:"component"`
	Router     string `gorm:"column:router;size:255" json:"router"`
	Log        int8   `gorm:"column:log" json:"log"`               // 是否记录日志
	Permission int8   `gorm:"column:permission" json:"permission"` // 是否鉴权
	Method     int8   `gorm:"column:method" json:"method"`         // 请求方式 1GET 2POST 3PUT 4DELETE
}

func (AdminMenu) TableName() string { return "admin_menu" }
