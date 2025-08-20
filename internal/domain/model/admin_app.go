package model

// AdminApp 应用信息表
// 兼容原字段命名，gorm 使用 column 指定

type AdminApp struct {
	ID         int64  `gorm:"primaryKey" json:"id"`
	AppID      string `gorm:"column:app_id" json:"app_id"`
	AppSecret  string `gorm:"column:app_secret" json:"app_secret"`
	AppName    string `gorm:"column:app_name" json:"app_name"`
	AppStatus  int8   `gorm:"column:app_status" json:"app_status"`
	AppInfo    string `gorm:"column:app_info" json:"app_info"`
	AppAPI     string `gorm:"column:app_api" json:"app_api"`
	AppGroup   string `gorm:"column:app_group" json:"app_group"`
	AppAddTime int64  `gorm:"column:app_add_time" json:"app_add_time"`
	AppAPIShow string `gorm:"column:app_api_show" json:"app_api_show"`
}

func (AdminApp) TableName() string { return "admin_app" }
