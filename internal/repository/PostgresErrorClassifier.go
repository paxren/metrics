package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"

	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/jackc/pgerrcode"
)

// PGErrorClassification тип для классификации ошибок PostgreSQL.
//
// Определяет, можно ли повторить операцию после получения ошибки.
type PGErrorClassification int

const (
	// NonRetriable - операцию не следует повторять, ошибка является постоянной
	NonRetriable PGErrorClassification = iota

	// Retriable - операцию можно повторить, ошибка является временной
	Retriable
)

// PostgresErrorClassifier классификатор ошибок PostgreSQL.
//
// Используется для определения, можно ли повторить операцию базы данных
// после получения ошибки. Анализирует коды ошибок PostgreSQL
// и классифицирует их как временные или постоянные.
type PostgresErrorClassifier struct{}

// NewPostgresErrorClassifier создаёт новый экземпляр классификатора ошибок.
//
// Возвращает:
//   - *PostgresErrorClassifier: указатель на созданный классификатор
func NewPostgresErrorClassifier() *PostgresErrorClassifier {
	return &PostgresErrorClassifier{}
}

// Classify классифицирует ошибку и определяет, можно ли повторить операцию.
//
// Проверяет, является ли ошибка ошибкой PostgreSQL, и классифицирует её.
// Если ошибка не является ошибкой PostgreSQL, по умолчанию считается неповторяемой.
//
// Параметры:
//   - err: ошибка для классификации
//
// Возвращает:
//   - PGErrorClassification: Retriable для временных ошибок, NonRetriable для постоянных
func (c *PostgresErrorClassifier) Classify(err error) PGErrorClassification {
	if err == nil {
		return NonRetriable
	}

	// Проверяем и конвертируем в pgconn.PgError, если это возможно
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return СlassifyPgError(pgErr)
	}

	// По умолчанию считаем ошибку неповторяемой
	return NonRetriable
}

// СlassifyPgError классифицирует ошибку PostgreSQL по её коду.
//
// Использует официальные коды ошибок PostgreSQL для определения,
// является ли ошибка временной (повторяемой) или постоянной.
//
// Справочник кодов ошибок: https://www.postgresql.org/docs/current/errcodes-appendix.html
//
// Параметры:
//   - pgErr: ошибка PostgreSQL для классификации
//
// Возвращает:
//   - PGErrorClassification: Retriable для временных ошибок, NonRetriable для постоянных
func СlassifyPgError(pgErr *pgconn.PgError) PGErrorClassification {
	// Коды ошибок PostgreSQL: https://www.postgresql.org/docs/current/errcodes-appendix.html

	switch pgErr.Code {
	// Класс 08 - Ошибки соединения (временные)
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure:
		return Retriable

	// Класс 40 - Откат транзакции (временные)
	case pgerrcode.TransactionRollback, // 40000
		pgerrcode.SerializationFailure, // 40001
		pgerrcode.DeadlockDetected:     // 40P01
		return Retriable

	// Класс 57 - Ошибка оператора (временные)
	case pgerrcode.CannotConnectNow: // 57P03
		return Retriable
	}

	// Можно добавить более конкретные проверки с использованием констант pgerrcode
	switch pgErr.Code {
	// Класс 22 - Ошибки данных (постоянные)
	case pgerrcode.DataException,
		pgerrcode.NullValueNotAllowedDataException:
		return NonRetriable

	// Класс 23 - Нарушение ограничений целостности (постоянные)
	case pgerrcode.IntegrityConstraintViolation,
		pgerrcode.RestrictViolation,
		pgerrcode.NotNullViolation,
		pgerrcode.ForeignKeyViolation,
		pgerrcode.UniqueViolation,
		pgerrcode.CheckViolation:
		return NonRetriable

	// Класс 42 - Синтаксические ошибки (постоянные)
	case pgerrcode.SyntaxErrorOrAccessRuleViolation,
		pgerrcode.SyntaxError,
		pgerrcode.UndefinedColumn,
		pgerrcode.UndefinedTable,
		pgerrcode.UndefinedFunction:
		return NonRetriable
	}

	// По умолчанию считаем ошибку неповторяемой
	return NonRetriable
}
