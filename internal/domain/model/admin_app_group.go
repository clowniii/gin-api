package model

// AdminAppGroup 应用分组

type AdminAppGroup struct {
	ID          int64  `gorm:"primaryKey" json:"id"`
	Name        string `gorm:"column:name" json:"name"`
	Description string `gorm:"column:description" json:"description"`
	Status      int8   `gorm:"column:status" json:"status"`
	Hash        string `gorm:"column:hash" json:"hash"`
}

func (AdminAppGroup) TableName() string { return "admin_app_group" }
