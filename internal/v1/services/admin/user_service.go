package admin

import (
	"errors"
	"fmt"
	"gin-app/internal/v1/services"
	"strconv"
	"strings"

	"gin-app/models/admin"
)

type UserService services.Service

// GetList 获取用户列表
func (s *UserService) GetList(keywords string) (interface{}, error) {
	users := make(admin.AdminUserSlice, 0)
	userList, err := users.GetAdminUserList(s.DB, keywords)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("数据获取失败：%s", err.Error()))
	}
	for _, v := range userList {
		_ = v.GetUserData(s.DB)
		err = v.GetGroup(s.DB)
		if err == nil && v.Access != nil {
			v.GroupId = strings.Split(v.Access.GroupId, ",")
		}
		v.Password = ""
	}

	list := make(map[string]interface{})
	list["list"] = userList
	return list, nil
}

// GetUsersByGid 获取权限组下关联用户
func (s *UserService) GetUsersByGid(size, page, gid int) (interface{}, error) {
	users := make(admin.AdminUserSlice, 0)
	var authGroupAccess = make([]admin.AdminAuthGroupAccess, 0)
	err := s.DB.Select(&authGroupAccess, "select * from admin_auth_group_access where find_in_set(?,`group_id`)", gid)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("数据获取失败：%s", err.Error()))
	}
	uIds := make([]int64, 0)
	for _, v := range authGroupAccess {
		uIds = append(uIds, v.Uid)
	}
	start := size * (page - 1)
	err = s.DB.Select(&users, "select * from admin_user where id in ? limit ?,?", uIds, start, size)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("数据获取失败：%s", err.Error()))
	}
	for _, v := range users {
		_ = v.GetUserData(s.DB)
	}

	list := make(map[string]interface{})
	list["list"] = users
	list["count"] = len(authGroupAccess)
	return list, nil
}
func (s *UserService) Add(u admin.AdminUser) error {
	err := u.Add(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据添加失败：%s", err.Error()))
	}
	return nil
}

func (s *UserService) Edit(u admin.AdminUser) error {
	err := u.Edit(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据编辑失败：%s", err.Error()))
	}
	return nil
}

func (s *UserService) Del(u admin.AdminUser) error {
	if u.IsAdministrator() {
		return errors.New("超级管理员不允许删除")
	}
	err := u.Del(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据删除失败：%s", err.Error()))
	}
	return nil
}

func (s *UserService) ChangeStatus(u admin.AdminUser) error {
	if u.IsAdministrator() {
		return errors.New("超级管理员不允许改状态")
	}
	status := strconv.Itoa(int(u.Status))
	err := u.Edit(s.DB, "status", status)
	if err != nil {
		return errors.New(fmt.Sprintf("数据编辑失败：%s", err.Error()))
	}
	return nil
}
