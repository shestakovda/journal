package crash

import (
	"time"

	"github.com/shestakovda/fdbx/v2/mvcc"
)

// Номера индексов
const (
	IndexDate uint16 = 0x0001
	IndexCode uint16 = 0x0002
)

func NewFdbxFactory(tx mvcc.Tx, crashID uint16) Factory { return newFdbxFactory(tx, crashID) }

/*
	Provider - менеджер регистрации внешних ошибок системы.

	* Формирует модели, которые могут быть сериализованы в RFC7807 и хранятся в БД
	* Содержит внутри себя транслятор по ошибкам, поэтому должен быть глобальным на проект
*/
type Provider interface {
	/*
		Register - регистрирует новое соответствие внешней ошибки внутренним.

		* status - внешний http код, может быть только групп 4** или 5**
		* number - номер ошибки согласно приоритету
		* title - заголовок ошибки. Не может быть пустой или форматной строкой
		* triggers - список внутренних ошибок, на которые будет сформирована эта внешняя

		* В случае, если передан некорректный параметр, паникует
		* Может вызываться несколько раз. Если у двух разных ошибок указана одинаковая внутренняя, сработает первая
	*/
	Register(status, number int, title string, triggers ...error)

	/*
		Report - формирование новой внешней ошибки.

		* err - внутренняя ошибка, для поиска соответствующей внешней. В случае пустого err возвращает nil
	*/
	Report(err error) *Report
}

// Factory - поставщик моделей для работы в рамках транзакции
type Factory interface {
	/*
		New - конструктор новой модели для сохранения
	*/
	New() Model

	/*
		ByID - получение отчета об ошибке по идентификатору.

		* id не должен быть пустым и валидируется как core.UUID

		* Если не найден, ErrNotFound
		* Если что-то пошло не так, ErrSelect
	*/
	ByID(id string) (Model, error)

	/*
		ByDateCode - список ошибок по диапазону дат и коду (или его части)

		* Если не указывать код, тогда фильтрация только по дате
	*/
	ByDateCode(from, to time.Time, code string) ([]Model, error)
}

// Model - запись ошибки в БД
type Model interface {
	/*
		Import - копирование основного представления в модель и сохранение в БД.
	*/
	Import(*Report) error

	/*
		Export - основное представление отчета об ошибке
	*/
	Export() *Report

	/*
		ExportRFC - представление для сериализации в API и событиях.

		* Не выводит стек и детализацию для отладки
	*/
	ExportRFC() *RFC

	/*
		ExportMonitoring - представление для сериализации в мониторинге.
	*/
	ExportMonitoring() *ViewMonitoring
}
