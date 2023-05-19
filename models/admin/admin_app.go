package admin

import (
	"database/sql"

	"github.com/jmoiron/sqlx"

	"gin-app/models"
)

type AdminApp struct {
	ID         int64  `db:"id" json:"id"`
	AppId      string `db:"app_id" json:"app_id"`             //  应用id
	AppSecret  string `db:"app_secret" json:"app_secret"`     //  应用密码
	AppName    string `db:"app_name" json:"app_name"`         //  应用名称
	AppStatus  int64  `db:"app_status" json:"app_status"`     //  应用状态：0表示禁用，1表示启用
	AppInfo    string `db:"app_info" json:"app_info"`         //  应用说明
	AppApi     string `db:"app_api" json:"app_api"`           //  当前应用允许请求的全部api接口
	AppGroup   string `db:"app_group" json:"app_group"`       //  当前应用所属的应用组唯一标识
	AppAddTime int64  `db:"app_add_time" json:"app_add_time"` //  应用创建时间
	AppApiShow string `db:"app_api_show" json:"app_api_show"` //  前台样式显示所需数据格式

	UpdateData map[string]interface{}
}
type AdminAppSlice []*AdminApp

func (a AdminAppSlice) GetAppList(db *sqlx.DB, keywords string) ([]*AdminApp, error) {
	var err error
	if keywords == "" {
		err = db.Select(&a, "select * from admin_app")
	} else {
		err = db.Select(&a, "select * from admin_app where admin_app.app_name like ?", "%"+keywords+"%")
	}

	if err != nil {
		return nil, err
	}
	return a, nil
}

func (m *AdminApp) Add(db *sqlx.DB) (sql.Result, error) {
	res, err := models.New(db).Insert(m)
	return res, err
}
func (m *AdminApp) Edit(db *sqlx.DB) (sql.Result, error) {
	res, err := models.New(db).Update(m)
	return res, err
}

func (m *AdminApp) Del(db *sqlx.DB) (sql.Result, error) {
	res, err := models.New(db).Delete(m)
	return res, err
}
