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

type Session[T any] struct {
	db        *GWDB
	tableName string

	sqlStr string
	args   []any

	tx        *sql.Tx
	isTxBegin bool // Indicates if a transaction has been started

	errStack []error
}

func Open() (*GWDB, error) {
	driverName := config.RootConfig.DataSource["driver"].(string)
	username := config.RootConfig.DataSource["username"].(string)
	password := config.RootConfig.DataSource["password"].(string)
	host := config.RootConfig.DataSource["host"].(string)
	port := config.RootConfig.DataSource["port"].(string)
	database := config.RootConfig.DataSource["database"].(string)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", username, password, host, port, database)
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetConnMaxIdleTime(time.Minute * 1)
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return &GWDB{
		db:     db,
		logger: log.GWLogger,
	}, nil
}

func (g *GWDB) SetMaxIdleConns(n int) {
	g.db.SetMaxIdleConns(n)
}

func (g *GWDB) SetMaxOpenConns(n int) {
	g.db.SetMaxOpenConns(n)
}

func (g *GWDB) SetConnMaxLifetime(d time.Duration) {
	g.db.SetConnMaxLifetime(d)
}

func (g *GWDB) SetConnMaxIdleTime(d time.Duration) {
	g.db.SetConnMaxIdleTime(d)
}

func (g *GWDB) Close() error {
	return g.db.Close()
}

func NewSession[T any](db *GWDB, tableName string) *Session[T] {
	return &Session[T]{
		db:        db,
		tableName: tableName,
		sqlStr:    "",
		args:      make([]any, 0),
		tx:        nil,
		isTxBegin: false,
		errStack:  make([]error, 0),
	}
}

func (s *Session[T]) Begin() error {
	if s.isTxBegin {
		return errors.New("transaction already started")
	}
	tx, err := s.db.db.Begin()
	if err != nil {
		return err
	}
	s.tx = tx
	s.isTxBegin = true
	s.db.logger.Info("Transaction started")
	return nil
}

func (s *Session[T]) Commit() error {
	if !s.isTxBegin {
		return errors.New("no transaction to commit")
	}
	if s.tx == nil {
		return errors.New("transaction not started")
	}
	err := s.tx.Commit()
	if err != nil {
		return err
	}
	s.isTxBegin = false
	s.tx = nil
	s.db.logger.Info("Transaction committed")
	return nil
}

func (s *Session[T]) Rollback() error {
	if !s.isTxBegin {
		return errors.New("no transaction to rollback")
	}
	if s.tx == nil {
		return errors.New("transaction not started")
	}
	err := s.tx.Rollback()
	if err != nil {
		return err
	}
	s.isTxBegin = false
	s.tx = nil
	s.db.logger.Info("Transaction rolled back")
	return nil
}

func (s *Session[T]) Exec() (int64, int64, error) {
	if len(s.errStack) > 0 {
		return -1, -1, errors.New(fmt.Sprintf("session has errors: %v", s.errStack))
	}
	if s.sqlStr == "" {
		return -1, -1, errors.New("no SQL to execute")
	}
	s.db.logger.Info(fmt.Sprintf("Executing SQL: %s", s.sqlStr))
	var stmt *sql.Stmt
	var err error
	if s.isTxBegin {
		if s.tx == nil {
			return -1, -1, errors.New("transaction not started")
		}
		stmt, err = s.tx.Prepare(s.sqlStr)
	} else {
		stmt, err = s.db.db.Prepare(s.sqlStr)
	}
	if err != nil {
		return -1, -1, err
	}
	res, err := stmt.Exec(s.args...)
	if err != nil {
		return -1, -1, err
	}
	lastInsertId, err := res.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	s.sqlStr = ""
	s.args = make([]any, 0) // Clear args after execution
	return lastInsertId, rowsAffected, nil
}

func (s *Session[T]) Query() ([]T, error) {
	if len(s.errStack) > 0 {
		return nil, errors.New(fmt.Sprintf("session has errors: %v", s.errStack))
	}
	if s.sqlStr == "" {
		return nil, errors.New("no SQL to query")
	}
	s.db.logger.Info(fmt.Sprintf("Executing Query SQL: %s", s.sqlStr))
	var stmt *sql.Stmt
	var err error
	if s.isTxBegin {
		if s.tx == nil {
			return nil, errors.New("transaction not started")
		}
		stmt, err = s.tx.Prepare(s.sqlStr)
	} else {
		stmt, err = s.db.db.Prepare(s.sqlStr)
	}
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(s.args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			return
		}
	}(rows)

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []T
	for rows.Next() {
		var item T
		scanAddr := make([]any, len(columns))
		values := make([]any, len(columns))
		for i := range scanAddr {
			scanAddr[i] = &values[i]
		}
		err = rows.Scan(scanAddr...)
		if err != nil {
			return nil, err
		}
		tElem := reflect.TypeOf(item).Elem()
		vElem := reflect.ValueOf(item).Elem()
		for i := 0; i < tElem.NumField(); i++ {
			name := tElem.Field(i).Name
			tag := tElem.Field(i).Tag.Get("orm")
			if tag == "" {
				tag = fieldToColumn(name)
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
		results = append(results, item)
	}
	s.sqlStr = ""
	s.args = make([]any, 0) // Clear args after execution
	return results, nil
}

func (s *Session[T]) QueryRow() (int64, error) {
	if len(s.errStack) > 0 {
		return -1, errors.New(fmt.Sprintf("session has errors: %v", s.errStack))
	}
	if s.sqlStr == "" {
		return -1, errors.New("no SQL to query")
	}
	s.db.logger.Info(fmt.Sprintf("Executing QueryRow SQL: %s", s.sqlStr))
	var stmt *sql.Stmt
	var err error
	if s.isTxBegin {
		if s.tx == nil {
			return -1, errors.New("transaction not started")
		}
		stmt, err = s.tx.Prepare(s.sqlStr)
	} else {
		stmt, err = s.db.db.Prepare(s.sqlStr)
	}
	if err != nil {
		return -1, err
	}
	var count int64
	err = stmt.QueryRow(s.args...).Scan(&count)
	if err != nil {
		return -1, err
	}
	s.sqlStr = ""
	s.args = make([]any, 0) // Clear args after execution
	return count, nil
}

func (s *Session[T]) Sql(sqlStr string, args ...any) *Session[T] {
	s.sqlStr = sqlStr
	s.args = append(s.args, args...)
	return s
}

func (s *Session[T]) Count() *Session[T] {
	// count(*) from tableName where condition
	s.sqlStr = fmt.Sprintf("SELECT COUNT(*) FROM %s", s.tableName)
	return s
}

func (s *Session[T]) Insert(data T) *Session[T] {
	// insert into tableName (field1, field2, ...) values (?, ?, ...)
	fields, placeholders, err := s.getInsertStr(data)
	if err != nil {
		s.errStack = append(s.errStack, err)
		return s
	}
	s.sqlStr = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", s.tableName, fields, placeholders)
	if err = s.setArgs(data); err != nil {
		s.errStack = append(s.errStack, err)
	}
	return s
}

func (s *Session[T]) InsertBatch(data []T) *Session[T] {
	// insert into tableName (field1, field2, ...) values (?, ?, ...), (?, ?, ...), ...
	if len(data) == 0 {
		s.errStack = append(s.errStack, errors.New("data slice is empty"))
		return s
	}
	fields, placeholder, err := s.getInsertStr(data[0])
	if err != nil {
		s.errStack = append(s.errStack, err)
		return s
	}
	phs := strings.Repeat("("+placeholder+"),", len(data)-1) + "(" + placeholder + ")"
	s.sqlStr = fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", s.tableName, fields, phs)
	for _, item := range data {
		if err = s.setArgs(item); err != nil {
			s.errStack = append(s.errStack, err)
			return s
		}
	}
	return s
}

func (s *Session[T]) Update(data T) *Session[T] {
	// update table tableName set field1 = ?, field2 = ? where condition
	fields, err := s.getUpdateStr(data)
	if err != nil {
		s.errStack = append(s.errStack, err)
		return s
	}
	s.sqlStr = fmt.Sprintf("UPDATE %s SET %s", s.tableName, fields)
	if err = s.setArgs(data); err != nil {
		s.errStack = append(s.errStack, err)
		return s
	}
	return s
}

func (s *Session[T]) Delete() *Session[T] {
	// delete from tableName where condition
	s.sqlStr = fmt.Sprintf("DELETE FROM %s", s.tableName)
	return s
}

func (s *Session[T]) Select(fields ...string) *Session[T] {
	// select field1, field2 from tableName where condition
	if len(fields) == 0 {
		s.sqlStr = fmt.Sprintf("SELECT * FROM %s", s.tableName)
	} else {
		s.sqlStr = fmt.Sprintf("SELECT %s FROM %s", strings.Join(fields, ", "), s.tableName)
	}
	return s
}

func (s *Session[T]) Eq(field string, value any) *Session[T] {
	// add where condition
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add condition"))
		return s
	}
	if strings.Contains(s.sqlStr, "WHERE") {
		s.sqlStr += fmt.Sprintf(" %s = ?", field)
	} else {
		s.sqlStr += fmt.Sprintf(" WHERE %s = ?", field)
	}
	s.args = append(s.args, value)
	return s
}

func (s *Session[T]) NotEq(field string, value any) *Session[T] {
	// add where condition for not equal
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add condition"))
		return s
	}
	if strings.Contains(s.sqlStr, "WHERE") {
		s.sqlStr += fmt.Sprintf(" %s != ?", field)
	} else {
		s.sqlStr += fmt.Sprintf(" WHERE %s != ?", field)
	}
	s.args = append(s.args, value)
	return s
}

func (s *Session[T]) Lt(field string, value any) *Session[T] {
	// add where condition for less than
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add condition"))
		return s
	}
	if strings.Contains(s.sqlStr, "WHERE") {
		s.sqlStr += fmt.Sprintf(" %s < ?", field)
	} else {
		s.sqlStr += fmt.Sprintf(" WHERE %s < ?", field)
	}
	s.args = append(s.args, value)
	return s
}

func (s *Session[T]) Gt(field string, value any) *Session[T] {
	// add where condition for greater than
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add condition"))
		return s
	}
	if strings.Contains(s.sqlStr, "WHERE") {
		s.sqlStr += fmt.Sprintf(" %s > ?", field)
	} else {
		s.sqlStr += fmt.Sprintf(" WHERE %s > ?", field)
	}
	s.args = append(s.args, value)
	return s
}

func (s *Session[T]) Ge(field string, value any) *Session[T] {
	// add where condition for greater than or equal
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add condition"))
		return s
	}
	if strings.Contains(s.sqlStr, "WHERE") {
		s.sqlStr += fmt.Sprintf(" %s >= ?", field)
	} else {
		s.sqlStr += fmt.Sprintf(" WHERE %s >= ?", field)
	}
	s.args = append(s.args, value)
	return s
}

func (s *Session[T]) Le(field string, value any) *Session[T] {
	// add where condition for less than or equal
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add condition"))
		return s
	}
	if strings.Contains(s.sqlStr, "WHERE") {
		s.sqlStr += fmt.Sprintf(" %s <= ?", field)
	} else {
		s.sqlStr += fmt.Sprintf(" WHERE %s <= ?", field)
	}
	s.args = append(s.args, value)
	return s
}

func (s *Session[T]) Like(field string, value any) *Session[T] {
	// add where condition for like
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add condition"))
		return s
	}
	if strings.Contains(s.sqlStr, "WHERE") {
		s.sqlStr += fmt.Sprintf(" %s LIKE ?", field)
	} else {
		s.sqlStr += fmt.Sprintf(" WHERE %s LIKE ?", field)
	}
	s.args = append(s.args, "%"+value.(string)+"%")
	return s
}

func (s *Session[T]) LikeLeft(field string, value any) *Session[T] {
	// add where condition for like left
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add condition"))
		return s
	}
	if strings.Contains(s.sqlStr, "WHERE") {
		s.sqlStr += fmt.Sprintf(" %s LIKE ?", field)
	} else {
		s.sqlStr += fmt.Sprintf(" WHERE %s LIKE ?", field)
	}
	s.args = append(s.args, "%"+value.(string))
	return s
}

func (s *Session[T]) LikeRight(field string, value any) *Session[T] {
	// add where condition for like right
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add condition"))
		return s
	}
	if strings.Contains(s.sqlStr, "WHERE") {
		s.sqlStr += fmt.Sprintf(" %s LIKE ?", field)
	} else {
		s.sqlStr += fmt.Sprintf(" WHERE %s LIKE ?", field)
	}
	s.args = append(s.args, value.(string)+"%")
	return s
}

func (s *Session[T]) GroupBy(field string) *Session[T] {
	// add GROUP BY clause
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add GROUP BY condition"))
		return s
	}
	s.sqlStr += fmt.Sprintf(" GROUP BY %s", field)
	return s
}

func (s *Session[T]) OrderBy(field string, asc bool) *Session[T] {
	// add ORDER BY clause
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add ORDER BY condition"))
		return s
	}
	order := "ASC"
	if !asc {
		order = "DESC"
	}
	s.sqlStr += fmt.Sprintf(" ORDER BY %s %s", field, order)
	return s
}

func (s *Session[T]) And() *Session[T] {
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add AND condition"))
		return s
	}
	s.sqlStr += " AND"
	return s
}

func (s *Session[T]) Or() *Session[T] {
	if s.sqlStr == "" {
		s.errStack = append(s.errStack, errors.New("no SQL to add OR condition"))
		return s
	}
	s.sqlStr += " OR"
	return s
}

func (s *Session[T]) setArgs(data T) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr {
		return errors.New("data must be a pointer to struct")
	}
	v := reflect.ValueOf(data).Elem()
	for i := 0; i < t.Elem().NumField(); i++ {
		s.args = append(s.args, v.Field(i).Interface())
	}
	return nil
}

func (s *Session[T]) getInsertStr(data T) (string, string, error) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr {
		return "", "", errors.New("data must be a pointer to struct")
	}
	var fields []string
	var placeholders []string
	tElem := t.Elem()
	for i := 0; i < tElem.NumField(); i++ {
		name := tElem.Field(i).Name
		tag := tElem.Field(i).Tag.Get("orm")
		if tag == "" {
			tag = fieldToColumn(name)
		} else {
			if strings.Contains(tag, "auto_increment") {
				continue
			}
			if strings.Contains(tag, ",") {
				tag = tag[:strings.Index(tag, ",")]
			}
		}
		fields = append(fields, name)
		placeholders = append(placeholders, "?")
	}
	return strings.Join(fields, ","), strings.Join(placeholders, ","), nil
}

func (s *Session[T]) getUpdateStr(data T) (string, error) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr {
		return "", errors.New("data must be a pointer to struct")
	}
	var fields []string
	tElem := t.Elem()
	for i := 0; i < tElem.NumField(); i++ {
		name := tElem.Field(i).Name
		tag := tElem.Field(i).Tag.Get("orm")
		if tag == "" {
			tag = fieldToColumn(name)
		} else {
			if strings.Contains(tag, "auto_increment") {
				continue
			}
			if strings.Contains(tag, ",") {
				tag = tag[:strings.Index(tag, ",")]
			}
		}
		fields = append(fields, fmt.Sprintf("%s = ?", tag))
	}
	return strings.Join(fields, ","), nil
}

func fieldToColumn(field string) string {
	var nameSlice = field[:]
	var sb strings.Builder
	lastIndex := 0
	for i, v := range nameSlice {
		if v >= 'A' && v <= 'Z' {
			if i == 0 {
				continue
			}
			sb.WriteString(field[lastIndex:i])
			sb.WriteByte('_')
			lastIndex = i
		}
	}
	sb.WriteString(field[lastIndex:])
	return strings.ToLower(sb.String())
}
