package model

// AdminAuthGroup 对应 admin_auth_group
// status: 1 正常 0 禁用

type AdminAuthGroup struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"size:50" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	Status      int8   `gorm:"column:status" json:"status"`
}

func (AdminAuthGroup) TableName() string { return "admin_auth_group" }
