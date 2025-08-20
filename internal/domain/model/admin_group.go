package model

// AdminGroup 对应原 admin_group (Wiki 接口组)
// 与 AdminInterfaceGroup 区分: 该表用于 Wiki 展示/统计 (含 hot, image)

type AdminGroup struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"column:name" json:"name"`
	Description string `gorm:"column:description" json:"description"`
	Status      int8   `gorm:"column:status" json:"status"`
	Hash        string `gorm:"column:hash" json:"hash"`
	CreateTime  int64  `gorm:"column:create_time" json:"create_time"`
	UpdateTime  int64  `gorm:"column:update_time" json:"update_time"`
	Image       string `gorm:"column:image" json:"image"`
	Hot         int64  `gorm:"column:hot" json:"hot"`
}

func (AdminGroup) TableName() string { return "admin_group" }
