package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"go-apiadmin/internal/domain/model"
	"go-apiadmin/internal/metrics"
	"go-apiadmin/internal/pkg/cache"
	"go-apiadmin/internal/repository/dao"
)

type FieldsService struct {
	DAO          *dao.AdminFieldsDAO
	InterfaceDAO *dao.AdminInterfaceListDAO
	Cache        cache.Cache // key: hash:type:page:limit -> json(ListFieldsResult)
}

type ListFieldsParams struct {
	Hash        string
	Type        int8
	Page, Limit int
}

type FieldDTO struct {
	ID        int64  `json:"id"`
	FieldName string `json:"field_name"`
	Hash      string `json:"hash"`
	DataType  int8   `json:"data_type"`
	Default   string `json:"default"`
	IsMust    int8   `json:"is_must"`
	Range     string `json:"range"`
	Info      string `json:"info"`
	Type      int8   `json:"type"`
	ShowName  string `json:"show_name"`
}

type ListFieldsResult struct {
	List     []FieldDTO     `json:"list"`
	Count    int64          `json:"count"`
	DataType map[int]string `json:"dataType"`
	ApiInfo  interface{}    `json:"apiInfo"`
}

var dataTypeMap = map[int]string{1: "Integer", 2: "String", 3: "Array", 4: "Float", 5: "Boolean", 6: "File", 7: "Enum", 8: "Mobile", 9: "Object"}

func DataTypeMap() map[int]string { return dataTypeMap }

func NewFieldsService(d *dao.AdminFieldsDAO, ifl *dao.AdminInterfaceListDAO) *FieldsService {
	return &FieldsService{DAO: d, InterfaceDAO: ifl, Cache: cache.NewSimpleAdapter(cache.New(60 * time.Second))}
}

func (s *FieldsService) List(ctx context.Context, p ListFieldsParams) (*ListFieldsResult, error) {
	if strings.TrimSpace(p.Hash) == "" {
		return nil, errors.New("hash required")
	}
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Limit <= 0 {
		p.Limit = 20
	}
	ck := s.cacheKey(p.Hash, p.Type, p.Page, p.Limit)
	if s.Cache != nil {
		if str, _ := s.Cache.Get(ctx, ck); str != "" {
			if cache.IsNilSentinel(str) { // 空 sentinel
				metrics.CacheNilHit.Inc()
				return &ListFieldsResult{List: []FieldDTO{}, Count: 0, DataType: dataTypeMap, ApiInfo: nil}, nil
			}
			var r ListFieldsResult
			if json.Unmarshal([]byte(str), &r) == nil {
				return &r, nil
			}
		}
	}
	list, total, err := s.DAO.List(ctx, p.Hash, p.Type, p.Page, p.Limit)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 { // 空结果防穿透
		if s.Cache != nil {
			_ = s.Cache.SetEX(ctx, ck, cache.WrapNil(true), 10*time.Second)
		}
		return &ListFieldsResult{List: []FieldDTO{}, Count: 0, DataType: dataTypeMap, ApiInfo: nil}, nil
	}
	res := make([]FieldDTO, 0, len(list))
	for _, m := range list {
		res = append(res, FieldDTO{ID: m.ID, FieldName: m.FieldName, Hash: m.Hash, DataType: m.DataType, Default: m.Default, IsMust: m.IsMust, Range: m.Range, Info: m.Info, Type: m.Type, ShowName: m.ShowName})
	}
	var apiInfo interface{}
	if ifc, _ := s.InterfaceDAO.FindByHash(ctx, p.Hash); ifc != nil {
		_ = json.Unmarshal([]byte(ifc.ReturnStr), &apiInfo)
	}
	result := &ListFieldsResult{List: res, Count: total, DataType: dataTypeMap, ApiInfo: apiInfo}
	if s.Cache != nil {
		b, _ := json.Marshal(result)
		_ = s.Cache.SetEX(ctx, ck, string(b), 60*time.Second)
	}
	return result, nil
}

type AddFieldParams struct {
	FieldName, Hash, Default, Range, Info, ShowName string
	DataType, IsMust, Type                          int8
}

type EditFieldParams struct {
	ID                                              int64
	FieldName, Hash, Default, Range, Info, ShowName *string
	DataType, IsMust, Type                          *int8
}

func (s *FieldsService) Add(ctx context.Context, p AddFieldParams) (int64, error) {
	if p.FieldName == "" || p.Hash == "" {
		return 0, errors.New("field_name & hash required")
	}
	m := &model.AdminField{FieldName: p.FieldName, Hash: p.Hash, DataType: p.DataType, Default: p.Default, IsMust: p.IsMust, Range: p.Range, Info: p.Info, Type: p.Type, ShowName: pickShowName(p.ShowName, p.FieldName)}
	if err := s.DAO.Create(ctx, m); err != nil {
		return 0, err
	}
	s.invalidateHash(p.Hash)
	return m.ID, nil
}

func (s *FieldsService) Edit(ctx context.Context, p EditFieldParams) error {
	if p.ID <= 0 {
		return errors.New("invalid id")
	}
	m, err := s.DAO.FindByID(ctx, p.ID)
	if err != nil {
		return err
	}
	if m == nil {
		return errors.New("not found")
	}
	if p.FieldName != nil {
		m.FieldName = *p.FieldName
	}
	if p.Hash != nil {
		m.Hash = *p.Hash
	}
	if p.Default != nil {
		m.Default = *p.Default
	}
	if p.Range != nil {
		m.Range = *p.Range
	}
	if p.Info != nil {
		m.Info = *p.Info
	}
	if p.ShowName != nil {
		m.ShowName = pickShowName(*p.ShowName, m.FieldName)
	}
	if p.DataType != nil {
		m.DataType = *p.DataType
	}
	if p.IsMust != nil {
		m.IsMust = *p.IsMust
	}
	if p.Type != nil {
		m.Type = *p.Type
	}
	if err := s.DAO.Update(ctx, m); err != nil {
		return err
	}
	s.invalidateHash(m.Hash)
	if p.Hash != nil {
		s.invalidateHash(*p.Hash)
	}
	return nil
}

func (s *FieldsService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	m, _ := s.DAO.FindByID(ctx, id)
	err := s.DAO.Delete(ctx, id)
	if m != nil {
		s.invalidateHash(m.Hash)
	}
	return err
}

type BatchUploadParams struct {
	Hash string
	Type int8
	JSON string
}

func (s *FieldsService) BatchUpload(ctx context.Context, p BatchUploadParams) error {
	if p.Hash == "" || p.JSON == "" {
		return errors.New("hash & json required")
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(p.JSON), &parsed); err != nil {
		return err
	}
	if ifc, _ := s.InterfaceDAO.FindByHash(ctx, p.Hash); ifc != nil {
		ifc.ReturnStr = p.JSON
		_ = s.InterfaceDAO.Update(ctx, &model.AdminInterfaceList{ID: ifc.ID, ReturnStr: p.JSON})
	}
	dataNode, ok := parsed["data"]
	if !ok {
		dataNode = parsed
	}
	var collect []model.AdminField
	buildFieldsRecursive(&collect, p.Hash, p.Type, "data", dataNode)
	if err := s.DAO.DeleteByHash(ctx, p.Hash); err != nil {
		return err
	}
	for i := range collect {
		_ = s.DAO.Create(ctx, &collect[i])
	}
	s.invalidateHash(p.Hash)
	return nil
}

// 缓存 key
func (s *FieldsService) cacheKey(hash string, typ int8, page, limit int) string {
	return hash + ":" + strings.TrimSpace(string(rune(typ))) + ":" + _intToStr(int64(page)) + ":" + _intToStr(int64(limit))
}

// 失效指定 hash
func (s *FieldsService) invalidateHash(hash string) {
	if hash == "" || s.Cache == nil {
		return
	}
	// 简化: 由于接口未提供 Keys 遍历能力, 直接忽略精细失效; 生产可维护二级索引
}

func pickShowName(show, field string) string {
	if strings.TrimSpace(show) == "" {
		return field
	}
	return show
}

func buildFieldsRecursive(out *[]model.AdminField, hash string, typ int8, key string, val interface{}) {
	field := model.AdminField{FieldName: key, ShowName: key, Hash: hash, IsMust: 1, Type: typ, DataType: 2}
	switch v := val.(type) {
	case map[string]interface{}:
		field.DataType = 9
		*out = append(*out, field)
		for k, child := range v {
			buildFieldsRecursive(out, hash, typ, k, child)
		}
	case []interface{}:
		field.DataType = 3
		*out = append(*out, field)
		if len(v) > 0 {
			buildFieldsRecursive(out, hash, typ, key, v[0])
		}
	case float64:
		if float64(int64(v)) == v {
			field.DataType = 1
		} else {
			field.DataType = 4
		}
		*out = append(*out, field)
	case bool:
		field.DataType = 5
		*out = append(*out, field)
	default:
		field.DataType = 2
		*out = append(*out, field)
	}
}

func _intToStr(i64 int64) string { return strconv.FormatInt(i64, 10) }
