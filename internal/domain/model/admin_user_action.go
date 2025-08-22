package model

// AdminUserAction 对应原 admin_user_action 操作日志表
// 兼容原字段: action_name, uid, nickname, add_time, data, url
// 扩展字段: method, status, latency_ms, ip

type AdminUserAction struct {
	ID         int64  `gorm:"primaryKey" json:"id"`
	ActionName string `gorm:"column:action_name;size:50" json:"action_name"`
	UID        int64  `gorm:"column:uid;index" json:"uid"`
	Nickname   string `gorm:"column:nickname;size:50" json:"nickname"`
	AddTime    int64  `gorm:"column:add_time" json:"add_time"`
	Data       string `gorm:"column:data" json:"data"`
	URL        string `gorm:"column:url;size:200" json:"url"`
	Method     string `gorm:"column:method;size:10" json:"method"`
	Status     int    `gorm:"column:status" json:"status"`
	LatencyMs  int64  `gorm:"column:latency_ms" json:"latency_ms"`
	IP         string `gorm:"column:ip;size:64" json:"ip"`
}

func (AdminUserAction) TableName() string { return "admin_user_action" }
