package admin

import (
	"database/sql"

	"github.com/jmoiron/sqlx"

	"gin-app/models"
)

type AdminUserData struct {
	ID            int64  `db:"id" json:"id"`
	LoginTimes    int64  `db:"login_times" json:"login_times"`         //  账号登录次数
	LastLoginIp   int64  `db:"last_login_ip" json:"last_login_ip"`     //  最后登录ip
	LastLoginTime int64  `db:"last_login_time" json:"last_login_time"` //  最后登录时间
	Uid           int64  `db:"uid" json:"uid"`                         //  用户id
	HeadImg       string `db:"head_img" json:"head_img"`               //  用户头像

	UpdateData map[string]interface{}
}

func (d *AdminUserData) CreateOrUpdate(db *sqlx.DB) (res sql.Result, err error) {
	if d.ID == 0 {
		res, err = models.New(db).Insert(d)
		return
	} else {
		res, err = models.New(db).Update(d)
		return
	}

}
