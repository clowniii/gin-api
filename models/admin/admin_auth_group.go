package admin

import (
	"database/sql"

	"github.com/jmoiron/sqlx"

	"gin-app/models"
)

type AdminAuthGroup struct {
	ID          int64  `db:"id" json:"id"`
	Name        string `db:"name" json:"name"`               //  组名称
	Description string `db:"description" json:"description"` //  组描述
	Status      int64  `db:"status" json:"status"`           //  组状态：为1正常，为0禁用

	UpdateData map[string]interface{}
}
type AdminAuthGroupSlice []*AdminAuthGroup

func (m AdminAuthGroupSlice) GetAuthGroupList(db *sqlx.DB, keywords string) ([]*AdminAuthGroup, error) {
	var err error
	if keywords == "" {
		err = db.Select(&m, "select * from admin_auth_group")
	} else {
		err = db.Select(&m, "select * from admin_auth_group where name like ?", "%"+keywords+"%")
	}

	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *AdminAuthGroup) Add(db *sqlx.DB) (sql.Result, error) {
	res, err := models.New(db).Insert(m)
	return res, err
}
func (m *AdminAuthGroup) Edit(db *sqlx.DB) (sql.Result, error) {
	res, err := models.New(db).Update(m)
	return res, err
}

func (m *AdminAuthGroup) Del(db *sqlx.DB) (sql.Result, error) {
	res, err := models.New(db).Delete(m)
	return res, err
}
