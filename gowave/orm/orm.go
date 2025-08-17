package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/ChenGuo505/gowave/config"
	"github.com/ChenGuo505/gowave/log"
	"reflect"
	"strings"
	"time"
)

type GWDB struct {
	db     *sql.DB
	logger *log.Logger
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

	tx      *sql.Tx
	isBegin bool
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
	return &GWDB{db: db, logger: log.GWLogger}, nil
}

func (db *GWDB) NewSession(tableName string) *GWSession {
	return &GWSession{
		db:        db,
		tableName: tableName,
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

func (s *GWSession) Exec(sqlStr string, args ...any) (int64, int64, error) {
	s.db.logger.Info(sqlStr)
	var stmt *sql.Stmt
	var err error
	if s.isBegin {
		stmt, err = s.tx.Prepare(sqlStr)
	} else {
		stmt, err = s.db.db.Prepare(sqlStr)
	}
	if err != nil {
		return -1, -1, err
	}
	result, err := stmt.Exec(args...)
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
	return lastInsertId, affectedRows, nil
}

func (s *GWSession) Insert(data any) (int64, int64, error) {
	s.setAttributes(data)
	sqlStr := fmt.Sprintf("insert into %s (%s) values (%s)", s.tableName, strings.Join(s.fields, ","), strings.Join(s.placeHolders, ","))
	s.db.logger.Info(sqlStr)
	var stmt *sql.Stmt
	var err error
	if s.isBegin {
		stmt, err = s.tx.Prepare(sqlStr)
	} else {
		stmt, err = s.db.db.Prepare(sqlStr)
	}
	if err != nil {
		return -1, -1, err
	}
	result, err := stmt.Exec(s.values...)
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
	s.db.logger.Info(sqlStr)
	var stmt *sql.Stmt
	var err error
	if s.isBegin {
		stmt, err = s.tx.Prepare(sqlStr)
	} else {
		stmt, err = s.db.db.Prepare(sqlStr)
	}
	if err != nil {
		return -1, -1, err
	}
	result, err := stmt.Exec(s.values...)
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
	sqlStr := fmt.Sprintf("update table %s set %s %s", s.tableName, strings.Join(s.updateFields, ","), s.conditions.String())
	s.values = append(s.values, s.conditionValues...)
	s.db.logger.Info(sqlStr)
	var stmt *sql.Stmt
	var err error
	if s.isBegin {
		stmt, err = s.tx.Prepare(sqlStr)
	} else {
		stmt, err = s.db.db.Prepare(sqlStr)
	}
	if err != nil {
		return -1, -1, err
	}
	result, err := stmt.Exec(s.values...)
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

func (s *GWSession) SelectOne(data any, fields ...string) (any, error) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr {
		return nil, errors.New("data must be a pointer to a struct")
	}
	fieldsStr := "*"
	if len(fields) > 0 {
		fieldsStr = strings.Join(fields, ",")
	}
	sqlStr := fmt.Sprintf("select %s from %s where %s", fieldsStr, s.tableName, s.conditions.String())
	s.db.logger.Info(sqlStr)
	var stmt *sql.Stmt
	var err error
	if s.isBegin {
		stmt, err = s.tx.Prepare(sqlStr)
	} else {
		stmt, err = s.db.db.Prepare(sqlStr)
	}
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(s.conditionValues...)
	if err != nil {
		return nil, err
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	if rows.Next() {
		fieldsScan := make([]any, len(columns))
		values := make([]any, len(columns))
		for i := range fieldsScan {
			fieldsScan[i] = &values[i]
		}
		err := rows.Scan(fieldsScan...)
		if err != nil {
			return nil, err
		}
		tElem := t.Elem()
		vElem := reflect.ValueOf(data).Elem()
		for i := 0; i < tElem.NumField(); i++ {
			name := tElem.Field(i).Name
			tag := tElem.Field(i).Tag.Get("orm")
			if tag == "" {
				tag = strings.ToLower(name)
			} else {
				if strings.Contains(tag, ",") {
					tag = tag[:strings.Index(tag, ",")]
				}
			}
			for j, column := range columns {
				if tag == column {
					val := values[j]
					valOf := reflect.ValueOf(val)
					fieldType := tElem.Field(i).Type
					convert := reflect.ValueOf(valOf.Interface()).Convert(fieldType)
					vElem.Field(i).Set(convert)
				}
			}
		}
	}
	s.conditions.Reset()
	s.conditionValues = make([]any, 0)
	return data, nil
}

func (s *GWSession) SelectAll(data any, fields ...string) ([]any, error) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr {
		return nil, errors.New("data must be a pointer to a struct")
	}
	fieldsStr := "*"
	if len(fields) > 0 {
		fieldsStr = strings.Join(fields, ",")
	}
	sqlStr := fmt.Sprintf("select %s from %s where %s", fieldsStr, s.tableName, s.conditions.String())
	s.db.logger.Info(sqlStr)
	var stmt *sql.Stmt
	var err error
	if s.isBegin {
		stmt, err = s.tx.Prepare(sqlStr)
	} else {
		stmt, err = s.db.db.Prepare(sqlStr)
	}
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(s.conditionValues...)
	if err != nil {
		return nil, err
	}
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]any, 0)
	for rows.Next() {
		data = reflect.New(t.Elem()).Interface()
		fieldsScan := make([]any, len(columns))
		values := make([]any, len(columns))
		for i := range fieldsScan {
			fieldsScan[i] = &values[i]
		}
		err := rows.Scan(fieldsScan...)
		if err != nil {
			return nil, err
		}
		tElem := t.Elem()
		vElem := reflect.ValueOf(data).Elem()
		for i := 0; i < tElem.NumField(); i++ {
			name := tElem.Field(i).Name
			tag := tElem.Field(i).Tag.Get("orm")
			if tag == "" {
				tag = strings.ToLower(name)
			} else {
				if strings.Contains(tag, ",") {
					tag = tag[:strings.Index(tag, ",")]
				}
			}
			for j, column := range columns {
				if tag == column {
					val := values[j]
					valOf := reflect.ValueOf(val)
					fieldType := tElem.Field(i).Type
					convert := reflect.ValueOf(valOf.Interface()).Convert(fieldType)
					vElem.Field(i).Set(convert)
				}
			}
		}
		result = append(result, data)
	}
	s.conditions.Reset()
	s.conditionValues = make([]any, 0)
	return result, nil
}

func (s *GWSession) Delete() (int64, int64, error) {
	sqlStr := fmt.Sprintf("delete from %s %s", s.tableName, s.conditions.String())
	s.db.logger.Info(sqlStr)
	var stmt *sql.Stmt
	var err error
	if s.isBegin {
		stmt, err = s.tx.Prepare(sqlStr)
	} else {
		stmt, err = s.db.db.Prepare(sqlStr)
	}
	if err != nil {
		return -1, -1, err
	}
	result, err := stmt.Exec(s.conditionValues...)
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
	s.conditions.Reset()
	s.conditionValues = make([]any, 0)
	return lastInsertId, affectedRows, nil
}

func (s *GWSession) Begin() error {
	tx, err := s.db.db.Begin()
	if err != nil {
		return err
	}
	s.tx = tx
	s.isBegin = true
	return nil
}

func (s *GWSession) Commit() error {
	if !s.isBegin {
		return errors.New("transaction not started")
	}
	err := s.tx.Commit()
	if err != nil {
		return err
	}
	s.isBegin = false
	s.tx = nil
	return nil
}

func (s *GWSession) Rollback() error {
	if !s.isBegin {
		return errors.New("transaction not started")
	}
	err := s.tx.Rollback()
	if err != nil {
		return err
	}
	s.isBegin = false
	s.tx = nil
	return nil
}

func (s *GWSession) Count() (int64, error) {
	sqlStr := fmt.Sprintf("select count(*) from %s %s", s.tableName, s.conditions.String())
	s.db.logger.Info(sqlStr)
	statement, err := s.db.db.Prepare(sqlStr)
	if err != nil {
		return -1, err
	}
	row := statement.QueryRow(s.conditionValues...)
	var count int64
	err = row.Scan(&count)
	if err != nil {
		return -1, err
	}
	s.conditions.Reset()
	s.conditionValues = make([]any, 0)
	return count, nil
}

func (s *GWSession) Equals(field string, value any) *GWSession {
	if s.conditions.String() == "" {
		s.conditions.WriteString("where ")
	}
	s.conditions.WriteString(fmt.Sprintf("%s = ?", field))
	s.conditionValues = append(s.conditionValues, value)
	return s
}

func (s *GWSession) NotEquals(field string, value any) *GWSession {
	if s.conditions.String() == "" {
		s.conditions.WriteString("where ")
	}
	s.conditions.WriteString(fmt.Sprintf("%s != ?", field))
	s.conditionValues = append(s.conditionValues, value)
	return s
}

func (s *GWSession) GreaterThan(field string, value any) *GWSession {
	if s.conditions.String() == "" {
		s.conditions.WriteString("where ")
	}
	s.conditions.WriteString(fmt.Sprintf("%s > ?", field))
	s.conditionValues = append(s.conditionValues, value)
	return s
}

func (s *GWSession) LessThan(field string, value any) *GWSession {
	if s.conditions.String() == "" {
		s.conditions.WriteString("where ")
	}
	s.conditions.WriteString(fmt.Sprintf("%s < ?", field))
	s.conditionValues = append(s.conditionValues, value)
	return s
}

func (s *GWSession) GreaterThanOrEqual(field string, value any) *GWSession {
	if s.conditions.String() == "" {
		s.conditions.WriteString("where ")
	}
	s.conditions.WriteString(fmt.Sprintf("%s >= ?", field))
	s.conditionValues = append(s.conditionValues, value)
	return s
}

func (s *GWSession) LessThanOrEqual(field string, value any) *GWSession {
	if s.conditions.String() == "" {
		s.conditions.WriteString("where ")
	}
	s.conditions.WriteString(fmt.Sprintf("%s <= ?", field))
	s.conditionValues = append(s.conditionValues, value)
	return s
}

func (s *GWSession) Like(field string, value any) *GWSession {
	if s.conditions.String() == "" {
		s.conditions.WriteString("where ")
	}
	s.conditions.WriteString(fmt.Sprintf("%s like ?", field))
	s.conditionValues = append(s.conditionValues, "%"+value.(string)+"%")
	return s
}

func (s *GWSession) LikeLeft(field string, value any) *GWSession {
	if s.conditions.String() == "" {
		s.conditions.WriteString("where ")
	}
	s.conditions.WriteString(fmt.Sprintf("%s like ?", field))
	s.conditionValues = append(s.conditionValues, "%"+value.(string))
	return s
}

func (s *GWSession) LikeRight(field string, value any) *GWSession {
	if s.conditions.String() == "" {
		s.conditions.WriteString("where ")
	}
	s.conditions.WriteString(fmt.Sprintf("%s like ?", field))
	s.conditionValues = append(s.conditionValues, value.(string)+"%")
	return s
}

func (s *GWSession) GroupBy(fields ...string) *GWSession {
	if len(fields) == 0 {
		return s
	}
	s.conditions.WriteString(" group by ")
	s.conditions.WriteString(strings.Join(fields, ","))
	return s
}

func (s *GWSession) OrderBy(asc bool, field string) *GWSession {
	if len(field) == 0 {
		return s
	}
	s.conditions.WriteString(" order by ")
	s.conditions.WriteString(field)
	if asc {
		s.conditions.WriteString(" asc")
	} else {
		s.conditions.WriteString(" desc")
	}
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
