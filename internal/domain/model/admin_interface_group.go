package model

// AdminInterfaceGroup 接口分组表 (映射原: admin_interface_group 或通过 group_hash 逻辑)
// 若原系统仅在 admin_list 中以 group_hash 表示分组，这里抽象为单独表以便后续管理。
// 字段：id, name, app_id(或可为空), status, sort, remark, add_time, update_time, hash

type AdminInterfaceGroup struct {
	ID         int64  `gorm:"primaryKey" json:"id"`
	Name       string `gorm:"column:name" json:"name"`
	AppID      string `gorm:"column:app_id" json:"app_id"` // 关联 admin_app.app_id，可选
	Status     int8   `gorm:"column:status" json:"status"` // 1 启用 0 禁用
	Sort       int    `gorm:"column:sort" json:"sort"`
	Remark     string `gorm:"column:remark" json:"remark"`
	Hash       string `gorm:"column:hash" json:"hash"` // 唯一分组标识，对应旧 group_hash
	AddTime    int64  `gorm:"column:add_time" json:"add_time"`
	UpdateTime int64  `gorm:"column:update_time" json:"update_time"`
}

func (AdminInterfaceGroup) TableName() string { return "admin_interface_group" }
