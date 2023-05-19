package admin

import (
	"errors"
	"reflect"

	"github.com/jmoiron/sqlx"

	"gin-app/models"
)

type AdminUser struct {
	ID         int64                 `db:"id" json:"id" `
	Username   string                `db:"username" json:"username"`       //  用户名
	Nickname   string                `db:"nickname" json:"nickname"`       //  用户昵称
	Password   string                `db:"password" json:"password"`       //  用户密码
	CreateTime int64                 `db:"create_time" json:"create_time"` //  注册时间
	CreateIp   int64                 `db:"create_ip" json:"create_ip"`     //  注册ip
	UpdateTime int64                 `db:"update_time" json:"update_time"` //  更新时间
	Status     int64                 `db:"status" json:"status"`           //  账号状态 0封号 1正常
	Openid     string                `db:"openid" json:"openid"`           //  三方登录唯一id
	ApiAuth    string                `json:"apiAuth"`
	UserData   *AdminUserData        `json:"userData"`
	Access     *AdminAuthGroupAccess `json:"access"`
	Menu       []*AdminMenu          `json:"menu"`
	AuthRule   []*AdminAuthRule      `json:"admin_auth_rule"`

	GroupId    []string `json:"group_id"`
	UpdateData map[string]interface{}
}
type ClientUser struct {
	ID       int    `json:"id"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	IP       int64  `json:"IP"`
}

type AdminUserSlice []*AdminUser

func (u AdminUserSlice) GetAdminUserList(db *sqlx.DB, keywords string) ([]*AdminUser, error) {
	var err error
	if keywords == "" {
		err = db.Select(&u, "select * from admin_user")
	} else {
		err = db.Select(&u, "select * from admin_user where admin_user.username like ?", "%"+keywords+"%")
	}

	if err != nil {
		return nil, err
	}
	return u, nil
}
func (u *AdminUser) SetUpdateData(data map[string]interface{}) (err error) {
	if u.UpdateData == nil {
		u.UpdateData = make(map[string]interface{})
	}
	for k, v := range data {
		if reflect.TypeOf(k).String() != "string" {
			return errors.New("map类型错误，请检查")
		}
		u.UpdateData[k] = v
	}
	return nil
}
func (u *AdminUser) Add(db *sqlx.DB) (err error) {
	_, err = models.New(db).Insert(u)
	return
}

func (u *AdminUser) Edit(db *sqlx.DB, exp ...string) (err error) {
	if u.UpdateData == nil {
		err = errors.New("请传入需更新参数")
		return
	}
	_, err = models.New(db).Table("admin_user").Where(exp...).Update(u)
	return
}

func (u *AdminUser) Del(db *sqlx.DB, exp ...string) (err error) {
	isSuper := u.IsAdministrator()
	if isSuper {
		err = errors.New("超级管理员不允许删除")
	} else {
		_, err = models.New(db).Where(exp...).Delete(u)
	}
	return
}

// GetUserData 获取用户数据
func (u *AdminUser) GetUserData(db *sqlx.DB) (err error) {
	if u.UserData == nil {
		u.UserData = &AdminUserData{}
	}
	err = db.Get(u.UserData, "select * from admin_user_data where uid = ? limit 1", u.ID)
	if err != nil {
		return err
	}
	return nil
}

// GetAccess 获取用户权限数据
func (u *AdminUser) GetAccess(db *sqlx.DB) (err error) {
	isSupper := u.IsAdministrator()
	if u.Access == nil {
		u.Access = &AdminAuthGroupAccess{}
	}
	if u.Menu == nil {
		u.Menu = AdminMenuSlice{}
	}
	if u.AuthRule == nil {
		u.AuthRule = AdminAuthRuleSlice{}
	}
	if isSupper {
		err = db.Select(&u.Menu, "select * from admin_menu")
		if err != nil {
			return
		}
	} else {
		err = db.Get(u.Access, "select * from admin_auth_group_access where uid = ? limit 1", u.ID)
		if err != nil {
			return
		}
		err = db.Select(&u.AuthRule, "select * from admin_auth_rule where group_id in (?)", u.Access.GroupId)
		if err != nil {
			return
		}
	}

	return nil
}

// GetAccessMenuData 获取当前用户的允许菜单
func (u *AdminUser) GetAccessMenuData(db *sqlx.DB) (err error) {
	if len(u.Menu) == 0 {
		err = u.GetAccess(db)
		if err != nil {
			return
		}
	}
	return nil
}

func (u *AdminUser) GetGroup(db *sqlx.DB) (err error) {
	isSupper := u.IsAdministrator()
	if !isSupper {
		if u.Access == nil {
			u.Access = &AdminAuthGroupAccess{}
		}
		err = db.Get(u.Access, "select * from admin_auth_group_access where uid = ? limit 1", u.ID)
		if err != nil {
			return
		}
	}
	return
}

func (u *AdminUser) IsAdministrator() bool {
	return u.ID == 1
}
