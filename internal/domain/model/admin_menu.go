package model

// AdminMenu 对应原 admin_menu 表
// 仅迁移当前步骤需要的字段，可后续补充
// show: 0/1/2 (原系统语义) 这里保持整型

type AdminMenu struct {
	ID        int64  `gorm:"primaryKey;column:id" json:"id"`
	FID       int64  `gorm:"column:fid" json:"fid"`
	Title     string `gorm:"column:title" json:"title"`
	Icon      string `gorm:"column:icon" json:"icon"`
	URL       string `gorm:"column:url" json:"url"`
	Router    string `gorm:"column:router" json:"router"`
	Component string `gorm:"column:component" json:"component"`
	Sort      int    `gorm:"column:sort" json:"sort"`
	Show      int    `gorm:"column:show" json:"show"`
	Level     int    `gorm:"column:level" json:"level"`
}

func (AdminMenu) TableName() string { return "admin_menu" }
