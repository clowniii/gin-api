package admin

import (
	"errors"
	"fmt"
	"gin-app/internal/v1/services"

	"gin-app/models/admin"
)

type AppService services.Service

func (s *AppService) GetList(keywords string) (interface{}, error) {
	var objs = make(admin.AdminAppSlice, 0)
	data, err := objs.GetAppList(s.DB, keywords)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("数据获取失败：%s", err.Error()))
	}
	list := make(map[string]interface{})
	list["list"] = data
	return list, nil
}
func (s *AppService) Add(obj admin.AdminApp) error {
	_, err := obj.Add(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据添加失败：%s", err.Error()))
	}
	return nil
}

func (s *AppService) Edit(obj admin.AdminApp) error {
	_, err := obj.Edit(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据编辑失败：%s", err.Error()))
	}
	return nil
}

func (s *AppService) Del(obj admin.AdminApp) error {
	_, err := obj.Del(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据删除失败：%s", err.Error()))
	}
	return nil
}

func (s *AppService) ChangeStatus(id, status int64) error {
	obj := admin.AdminApp{ID: id}
	_, err := obj.Edit(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据删除失败：%s", err.Error()))
	}
	return nil
}
