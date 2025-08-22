package model

// AdminAuthGroupAccess 用户与权限组关系 (兼容旧结构)
// group_id 保持为字符串(varchar(255))，旧库里可能存多个组ID（分隔符需按历史实现解析）
// 后续若做规范化拆分，可新增迁移脚本创建关系表并替换业务读取逻辑。

type AdminAuthGroupAccess struct {
	ID      int64  `gorm:"primaryKey;column:id" json:"id"`
	UID     int64  `gorm:"column:uid;index" json:"uid"`
	GroupID string `gorm:"column:group_id;size:255;index" json:"group_id"`
}

func (AdminAuthGroupAccess) TableName() string { return "admin_auth_group_access" }
