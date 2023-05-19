package admin

import (
	"errors"
	"fmt"
	"gin-app/internal/v1/services"

	"gin-app/models/admin"
	"gin-app/utils"
)

type MenuService services.Service

func (s *MenuService) GetMenuList(keywords string) (interface{}, error) {
	var objs = make(admin.AdminMenuSlice, 0)
	data, err := objs.GetMenuList(s.DB, keywords)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("数据获取失败：%s", err.Error()))
	}
	list := make(map[string]interface{})
	list["list"] = utils.GenerateMenuTree(data, false)
	return list, nil
}
func (s *MenuService) Add(obj admin.AdminMenu) error {
	if obj.Url != "" {
		obj.Url = "admin/" + obj.Url
	}
	_, err := obj.Add(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据添加失败：%s", err.Error()))
	}
	return nil
}

func (s *MenuService) Edit(obj admin.AdminMenu) error {
	if obj.Url != "" {
		obj.Url = "admin/" + obj.Url
	}
	_, err := obj.Edit(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据编辑失败：%s", err.Error()))
	}
	return nil
}

func (s *MenuService) Del(obj admin.AdminMenu) error {
	if obj.ID == 0 {
		return errors.New("缺少必填参数")
	}
	_, err := obj.Del(s.DB)
	if err != nil {
		return errors.New(fmt.Sprintf("数据删除失败：%s", err.Error()))
	}
	return nil
}
