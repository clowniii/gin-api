package model

// AdminAuthGroupAccess 用户与权限组关系
// group_id 原为字符串(存多组ID)，这里规范化为一行一条关系，需后续迁移脚本拆分
// 如果保持兼容，可先保留原结构；此处采用规范化: user_id, group_id
// 若需兼容旧库(单行存多个ID)，后续添加自定义加载逻辑。

type AdminAuthGroupAccess struct {
	ID      int64 `gorm:"primaryKey" json:"id"`
	UID     int64 `gorm:"column:uid" json:"uid"`
	GroupID int64 `gorm:"column:group_id" json:"group_id"`
}

func (AdminAuthGroupAccess) TableName() string { return "admin_auth_group_access" }
