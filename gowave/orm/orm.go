package orm

import (
	"database/sql"
	"errors"
	"fmt"
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
}

func Open(driverName, dataSourceName string) (*GWDB, error) {
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
	s.setFields(data)
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
	return lastInsertId, affectedRows, nil
}

func (s *GWSession) setFields(data any) {
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if t.Kind() != reflect.Ptr {
		panic(errors.New("data must be a pointer to a struct"))
	}
	tElem := t.Elem()
	vElem := v.Elem()
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
		s.values = append(s.values, vElem.Field(i).Interface())
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
