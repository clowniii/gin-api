package admin

import (
	"errors"
	"fmt"
	"gin-app/internal/v1/services"

	"gin-app/models/admin"
)

type AuthService services.Service

func (s *AuthService) GetList(keywords string) (interface{}, error) {
	var objs = make(admin.AdminAuthGroupSlice, 0)
	data, err := objs.GetAuthGroupList(s.DB, keywords)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("数据获取失败：%s", err.Error()))
	}
	list := make(map[string]interface{})
	list["list"] = data
	return list, nil
}
func (s *AuthService) Add(obj admin.AdminAuthGroup) error {
	_, err := obj.Add(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据添加失败：%s", err.Error()))
	}
	return nil
}

func (s *AuthService) Edit(obj admin.AdminAuthGroup) error {
	_, err := obj.Edit(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据编辑失败：%s", err.Error()))
	}
	return nil
}

func (s *AuthService) Del(obj admin.AdminAuthGroup) error {
	_, err := obj.Del(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据删除失败：%s", err.Error()))
	}
	return nil
}

func (s *AuthService) ChangeStatus(id, status int64) error {
	obj := admin.AdminAuthGroup{ID: id}
	_, err := obj.Edit(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据删除失败：%s", err.Error()))
	}
	return nil
}
