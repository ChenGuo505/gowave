package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/ChenGuo505/gowave/config"
	"reflect"
	"strings"
	"time"
)

type GWDB struct {
	db *sql.DB
}

type GWSession struct {
	db           *GWDB
	tableName    string
	fields       []string
	placeHolders []string
	values       []any
	updateFields []string

	conditions      strings.Builder
	conditionValues []any
}

func Open() (*GWDB, error) {
	driverName := config.RootConfig.DataSource["driver"].(string)
	username := config.RootConfig.DataSource["username"].(string)
	password := config.RootConfig.DataSource["password"].(string)
	host := config.RootConfig.DataSource["host"].(string)
	port := config.RootConfig.DataSource["port"].(int64)
	database := config.RootConfig.DataSource["database"].(string)
	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", username, password, host, port, database)
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		panic(err)
	}
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetConnMaxIdleTime(time.Minute * 1)
	if err = db.Ping(); err != nil {
		panic(err)
	}
	return &GWDB{db: db}, nil
}

func (db *GWDB) NewSession() *GWSession {
	return &GWSession{
		db: db,
	}
}

func (db *GWDB) SetMaxIdleConns(n int) {
	db.db.SetMaxIdleConns(n)
}

func (db *GWDB) SetMaxOpenConns(n int) {
	db.db.SetMaxOpenConns(n)
}

func (db *GWDB) SetConnMaxLifetime(duration time.Duration) {
	db.db.SetConnMaxLifetime(duration)
}

func (db *GWDB) SetConnMaxIdleTime(duration time.Duration) {
	db.db.SetConnMaxIdleTime(duration)
}

func (db *GWDB) Close() error {
	return db.db.Close()
}

func (s *GWSession) Table(name string) *GWSession {
	s.tableName = name
	return s
}

func (s *GWSession) Insert(data any) (int64, int64, error) {
	s.setAttributes(data)
	sqlStr := fmt.Sprintf("insert into %s (%s) values (%s)", s.tableName, strings.Join(s.fields, ","), strings.Join(s.placeHolders, ","))
	statement, err := s.db.db.Prepare(sqlStr)
	if err != nil {
		return -1, -1, err
	}
	result, err := statement.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	s.fields = make([]string, 0)
	s.placeHolders = make([]string, 0)
	s.values = make([]any, 0)
	return lastInsertId, affectedRows, nil
}

func (s *GWSession) InsertBatch(data []any) (int64, int64, error) {
	if len(data) == 0 {
		return -1, -1, errors.New("data cannot be empty")
	}
	s.setAttributesBatch(data)
	sqlStr := fmt.Sprintf("insert into %s (%s) values ", s.tableName, strings.Join(s.fields, ","))
	var sb strings.Builder
	sb.WriteString(sqlStr)
	for i := range data {
		sb.WriteString("(")
		sb.WriteString(strings.Join(s.placeHolders, ","))
		sb.WriteString(")")
		if i < len(data)-1 {
			sb.WriteString(",")
		}
	}
	sqlStr = sb.String()
	statement, err := s.db.db.Prepare(sqlStr)
	if err != nil {
		return -1, -1, err
	}
	result, err := statement.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	s.fields = make([]string, 0)
	s.placeHolders = make([]string, 0)
	s.values = make([]any, 0)
	return lastInsertId, affectedRows, nil
}

func (s *GWSession) Update(data any) (int64, int64, error) {
	s.setUpdateFields(data)
	s.setValues(data)
	sqlStr := fmt.Sprintf("update table %s set %s where %s", s.tableName, strings.Join(s.updateFields, ","), s.conditions.String())
	s.values = append(s.values, s.conditionValues...)
	statement, err := s.db.db.Prepare(sqlStr)
	if err != nil {
		return -1, -1, err
	}
	result, err := statement.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	affectedRows, err := result.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	s.updateFields = make([]string, 0)
	s.values = make([]any, 0)
	s.conditions.Reset()
	s.conditionValues = make([]any, 0)
	return lastInsertId, affectedRows, nil
}

func (s *GWSession) Equals(field string, value any) *GWSession {
	s.conditions.WriteString(fmt.Sprintf("%s = ?", field))
	s.conditionValues = append(s.conditionValues, value)
	return s
}

func (s *GWSession) NotEquals(field string, value any) *GWSession {
	s.conditions.WriteString(fmt.Sprintf("%s != ?", field))
	s.conditionValues = append(s.conditionValues, value)
	return s
}

func (s *GWSession) GreaterThan(field string, value any) *GWSession {
	s.conditions.WriteString(fmt.Sprintf("%s > ?", field))
	s.conditionValues = append(s.conditionValues, value)
	return s
}

func (s *GWSession) LessThan(field string, value any) *GWSession {
	s.conditions.WriteString(fmt.Sprintf("%s < ?", field))
	s.conditionValues = append(s.conditionValues, value)
	return s
}

func (s *GWSession) GreaterThanOrEqual(field string, value any) *GWSession {
	s.conditions.WriteString(fmt.Sprintf("%s >= ?", field))
	s.conditionValues = append(s.conditionValues, value)
	return s
}

func (s *GWSession) LessThanOrEqual(field string, value any) *GWSession {
	s.conditions.WriteString(fmt.Sprintf("%s <= ?", field))
	s.conditionValues = append(s.conditionValues, value)
	return s
}

func (s *GWSession) And() *GWSession {
	s.conditions.WriteString(" and ")
	return s
}

func (s *GWSession) Or() *GWSession {
	s.conditions.WriteString(" or ")
	return s
}

func (s *GWSession) setAttributes(data any) {
	s.setFieldsAndPlaceHolders(data)
	s.setValues(data)
}

func (s *GWSession) setAttributesBatch(dataArray []any) {
	s.setFieldsAndPlaceHolders(dataArray[0])
	for _, data := range dataArray {
		s.setValues(data)
	}
}

func (s *GWSession) setFieldsAndPlaceHolders(data any) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr {
		panic(errors.New("data must be a pointer to a struct"))
	}
	tElem := t.Elem()
	for i := 0; i < tElem.NumField(); i++ {
		fieldName := tElem.Field(i).Name
		tag := tElem.Field(i).Tag.Get("orm")
		if tag == "" {
			tag = strings.ToLower(fieldToColumn(fieldName))
		} else {
			if strings.Contains(tag, "auto_increment") {
				continue
			}
			if strings.Contains(tag, ",") {
				tag = tag[:strings.Index(tag, ",")]
			}
		}
		s.fields = append(s.fields, tag)
		s.placeHolders = append(s.placeHolders, "?")
	}
}

func (s *GWSession) setUpdateFields(data any) {
	s.updateFields = make([]string, 0)
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr {
		panic(errors.New("data must be a pointer to a struct"))
	}
	tElem := t.Elem()
	for i := 0; i < tElem.NumField(); i++ {
		fieldName := tElem.Field(i).Name
		tag := tElem.Field(i).Tag.Get("orm")
		if tag == "" {
			tag = strings.ToLower(fieldToColumn(fieldName))
		} else {
			if strings.Contains(tag, "auto_increment") {
				continue
			}
			if strings.Contains(tag, ",") {
				tag = tag[:strings.Index(tag, ",")]
			}
		}
		s.updateFields = append(s.updateFields, fmt.Sprintf("%s = ?", tag))
	}
}

func (s *GWSession) setValues(data any) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr {
		panic(errors.New("data must be a pointer to a struct"))
	}
	v := reflect.ValueOf(data).Elem()
	for i := 0; i < t.Elem().NumField(); i++ {
		s.values = append(s.values, v.Field(i).Interface())
	}
}

func fieldToColumn(fieldName string) string {
	var nameSlice = fieldName[:]
	var sb strings.Builder
	lastIndex := 0
	for idx, value := range nameSlice {
		if value >= 'A' && value <= 'Z' {
			if idx == 0 {
				continue
			}
			sb.WriteString(fieldName[lastIndex:idx])
			sb.WriteString("_")
			lastIndex = idx
		}
	}
	sb.WriteString(fieldName[lastIndex:])
	return sb.String()
}
