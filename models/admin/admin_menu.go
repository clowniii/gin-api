package admin

import (
	"database/sql"

	"github.com/jmoiron/sqlx"

	"gin-app/models"
)

type AdminMenu struct {
	ID         int64        `db:"id" json:"id"`
	Title      string       `db:"title" json:"title"`
	Fid        int64        `db:"fid" json:"fid"`               //  父级菜单id
	Url        string       `db:"url" json:"url"`               //  链接
	Auth       int64        `db:"auth" json:"auth"`             //  是否需要登录才可以访问，1-需要，0-不需要
	Sort       int64        `db:"sort" json:"sort"`             //  排序
	Show       int64        `db:"show" json:"show"`             //  是否显示，1-显示，0-隐藏
	Icon       string       `db:"icon" json:"icon"`             //  菜单图标
	Level      int64        `db:"level" json:"level"`           //  菜单层级，1-一级菜单，2-二级菜单，3-按钮
	Component  string       `db:"component" json:"component"`   //  前端组件
	Router     string       `db:"router" json:"router"`         //  前端路由
	Log        int64        `db:"log" json:"log"`               //  是否记录日志，1-记录，0-不记录
	Permission int64        `db:"permission" json:"permission"` //  是否验证权限，1-鉴权，0-放行
	Method     int64        `db:"method" json:"method"`         //  请求方式，1-get, 2-post, 3-put, 4-delete
	Children   []*AdminMenu `json:"children"`

	UpdateData map[string]interface{}
}
type AdminMenuSlice []*AdminMenu

func (m AdminMenuSlice) GetMenuList(db *sqlx.DB, keywords string) ([]*AdminMenu, error) {
	var err error
	if keywords == "" {
		err = db.Select(&m, "select * from admin_menu")
	} else {
		err = db.Select(&m, "select * from admin_menu where title like ?", "%"+keywords+"%")
	}

	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *AdminMenu) Add(db *sqlx.DB) (sql.Result, error) {
	res, err := models.New(db).Insert(m)
	return res, err
}
func (m *AdminMenu) Edit(db *sqlx.DB) (sql.Result, error) {
	res, err := models.New(db).Update(m)
	return res, err
}

func (m *AdminMenu) Del(db *sqlx.DB) (sql.Result, error) {
	res, err := models.New(db).Delete(m)
	return res, err
}
