package models

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/jmoiron/sqlx"
)

type BaseModel struct {
	tableName string
	orderBy   string
	limit     int
	page      int
	condition map[string]interface{}

	whereString string
	table       interface{}
	field       string
	fieldZ      string //占位符
	values      []interface{}
	*sqlx.DB
}

func New(db *sqlx.DB) *BaseModel {
	return &BaseModel{
		orderBy:   "",
		limit:     0,
		page:      0,
		condition: nil,
		DB:        db,
	}
}

// Table 用户传入表名
func (b *BaseModel) Table(table string) *BaseModel {
	b.tableName = table
	return b
}

func (b *BaseModel) Where(exp ...string) *BaseModel {
	if len(exp)%2 == 0 && len(exp) > 0 {
		for k, v := range exp {
			if k%2 != 0 {
				if b.condition == nil {
					b.condition = make(map[string]interface{})
				}
				b.condition["where:"+exp[k-1]] = v
			}
		}
	}
	return b
}

func (b *BaseModel) WhereIn(conName string, exp []string) *BaseModel {
	b.condition["whereIn:"+conName] = exp
	return b
}

func (b *BaseModel) WhereLike(exp ...string) *BaseModel {
	if len(exp)%1 == 0 {
		for k, v := range exp {
			if k&1 == 0 {
				b.condition["whereLike:"+exp[k-1]] = v
			}
		}
	}
	return b
}

func (b *BaseModel) Insert(table interface{}) (sql.Result, error) {
	b.table = table
	sqlStr := fmt.Sprintf("insert into %s (", b.tableName)
	b.getFieldAndValues(true)
	sqlStr = sqlStr + b.field + ")" + "VALUE (" + b.fieldZ + ")"
	return b.DB.Exec(sqlStr, b.values...)
}

func (b *BaseModel) Update(table interface{}) (sql.Result, error) {
	if b.tableName == "" {
		b.table = table
	}
	b.getFieldAndValues(false)
	if b.field == "" {
		return nil, errors.New("参数不合法")
	}
	b.getWhereString()
	sqlStr := fmt.Sprintf("update %s set ", b.tableName)

	sqlStr = sqlStr + b.field + "where " + b.whereString

	return b.DB.Exec(sqlStr, b.values...)
}
func (b *BaseModel) Delete(table interface{}) (sql.Result, error) {
	b.table = table
	b.getFieldAndValues(false)
	b.getWhereString()
	sqlStr := fmt.Sprintf("DELETE from %s", b.tableName)

	sqlStr = sqlStr + b.field + "where " + b.whereString
	return b.DB.Exec(sqlStr, b.values...)
}

func (b *BaseModel) getWhereString() {
	for k, v := range b.condition {
		if strings.Contains(k, "where:") {
			tempField := strings.Split(k, ":")
			if b.whereString == "" {
				b.whereString += tempField[1] + "=" + fmt.Sprintf("%v", v)

			} else {
				b.whereString += "and " + tempField[1] + "=" + fmt.Sprintf("%v", v)
			}
		}
	}
}
func (b *BaseModel) getFieldAndValues(isInsert bool) {
	v := reflect.ValueOf(b.table).Elem()
	t := v.Type()
	if b.tableName == "" {
		b.tableName = t.Name()
		b.tableName = strings.Replace(b.tableName, "Res", "", -1)
		b.tableName = strings.Replace(b.tableName, "Req", "", -1)
	}
	b.toSnakeCase()
	var m []map[string]interface{}
	if t.Kind() == reflect.Struct {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			value := v.Field(i).Interface()
			if field.Name != "ID" && isInsert {
				if !reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
					m = append(m, map[string]interface{}{field.Tag.Get("json"): value})
				}
			}
			if field.Name == "UpdateData" && !isInsert {
				for k, v := range value.(map[string]interface{}) {
					m = append(m, map[string]interface{}{k: v})
				}
			}

			if field.Name == "ID" && !isInsert && !reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
				b.Where("id", strconv.Itoa(int(v.Field(i).Interface().(int64))))
			}
		}
	}
	for i := 0; i < len(m); i++ {
		for k, v := range m[i] {
			if i == 0 {
				if isInsert {
					b.field += k
					b.fieldZ += "?"
				} else {
					b.field += k + "=" + "? "
				}
			} else {
				if isInsert {
					b.field += "," + k
					b.fieldZ += "," + "?"
				} else {
					b.field += " ," + k + "= ? "
				}
			}
			b.values = append(b.values, v)
		}
	}
}
func (b *BaseModel) toSnakeCase() {
	var builder strings.Builder
	for i, char := range b.tableName {
		if i == 0 && !unicode.IsUpper(char) {
			break
		} else {
			if char >= 'A' && char <= 'Z' {
				if i > 0 {
					builder.WriteRune('_')
				}
				builder.WriteRune(char + ('a' - 'A'))
			} else {
				builder.WriteRune(char)
			}
		}
	}
	b.tableName = builder.String()
}
