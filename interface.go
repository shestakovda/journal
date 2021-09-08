package journal

import (
	"time"

	"github.com/shestakovda/errx"
	"github.com/shestakovda/fdbx/v2/db"
	"github.com/shestakovda/fdbx/v2/mvcc"

	"github.com/shestakovda/journal/crash"
)

// Типы для моделей по-умолчанию
const (
	ModelTypeUnknown journalModels = 0
	ModelTypeCrash   journalModels = 1
)

// Константы индексов
const (
	IndexStart uint16 = 0x0001
	IndexModel uint16 = 0x0002
)

// NewFdbxFactory - конструктор фабрики для загрузки через fdbx/v2
func NewFdbxFactory(tx mvcc.Tx, journalID, crashID uint16) Factory {
	return newFdbxFactory(tx, journalID, crashID)
}

// NewFdbxDriver - конструктор драйвера для сохранения через fdbx/v2
func NewFdbxDriver(dbc db.Connection, journalID, crashID uint16) Driver {
	return newFdbxDriver(dbc, journalID, crashID)
}

// ModelType - абстрактный тип модели для логирования
type ModelType interface {
	ID() int
	String() string
}

// RegisterType - регистрация типа для корректной загрузки данных из БД
func RegisterType(types ...ModelType) { regType(types...) }

// Provider сборки и сохранения журнала
type Provider interface {
	/*
		Print - простая текстовая запись, для отладки или обозначения контрольной точки в процессе.

		* txt - текст записи, возможно форматная строка
		* args - аргументы форматной строки, могут отсутствовать, если текст не требует форматирования

		* Вызов функции создает новую отметку времени в цепочке
	*/
	Print(txt string, args ...interface{})

	/*
		Model - запись со ссылкой на какую-то модель в БД.

		* mtp - тип модели записи журнала, один из допустимых для логирования и поиска
		* mid - идентификатор модели, будет участвовать в поисковом индексе
		* txt - комментарий к записи, возможно форматная строка
		* args - аргументы форматной строки, могут отсутствовать, если текст не требует форматирования

		* Вызов функции создает новую отметку времени в цепочке
		* Если указан пустой идентификатор, то функция аналогична вызову Print
	*/
	Model(mtp ModelType, mid string, txt string, args ...interface{})

	/*
		Dump - сериализация одной или нескольких записей в json со ссылкой на какую-то модель в БД.

		* mtp - тип модели записи журнала, один из допустимых для логирования и поиска
		* mid - идентификатор модели, будет участвовать в поисковом индексе
		* items - одна или несколько моделей для записи

		* Сериализация выполняется через json.MarshalIndent с отступом \t
		* Если не удалось сериализовать в json, то вызовется Crash
	*/
	Dump(mtp ModelType, mid string, items ...interface{})

	/*
		Diff - логирование изменений между старыми данными и новыми со ссылкой на какую-то модель в БД.

		* mtp - тип модели записи журнала, один из допустимых для логирования и поиска
		* mid - идентификатор модели, будет участвовать в поисковом индексе
		* old - структура данных до изменения
		* new - структура данных после изменения

		* Если изменений нет, то ничего не логируется
	*/
	Diff(mtp ModelType, mid string, old, new interface{})

	/*
		Crash - логирование ошибки в журнал с формированием и записью отчета.

		* err - любая ошибка, которая будет возвращена как внешняя

		* Если err = nil, то возвращается тоже nil
		* Ошибка логируется как модель с идентификатором типа ModelTypeCrash
		* В текстовый комментарий к модели идет содержимое err.Error()
	*/
	Crash(err error) *crash.Report

	/*
		Close - закрытие модели, запись в glog и сохранение с помощью фабрики.

		* Обязательно требуется вызов этого метода, иначе все записи будут потеряны
	*/
	Close() *Entry

	/*
		Clone - создание нового чистого провайдера, с теми же параметрами.
	*/
	Clone() Provider

	/*
		SaveOnlyError - пометка о том, что сохранять запись в БД при закрытии нужно только в случае ошибки
	*/
	SaveOnlyError(opt bool)

	/*
		Debug - общая запись значений для дебага, который записывается в каждую ошибку.
		В записях журнала дебаг не сохраняется.

		* Старые значения из карты не удаляются.
		* Повторяющиеся - перезаписываются последними данными.
	*/
	Debug(dbg map[string]string)

	/*
		OnCrash - установка обработчика, который будет вызываться при Crash,
		когда возвращаемый crash.Report не равен nil
	*/
	OnCrash(handler CrashHandler)
}

// Driver - помощник сохранения журнала для провайдера
type Driver interface {
	/*
		InsertEntry - сохранение записи журнала в БД.
	*/
	InsertEntry(*Entry) error
}

// Factory - поставщик моделей для работы в рамках транзакции
type Factory interface {
	/*
		New - конструктор новой модели для сохранения
	*/
	New() Model

	/*
		ByID - получение записи журнала по идентификатору.

		* id не должен быть пустым и валидируется как core.UUID

		* Если не найден, ErrNotFound
		* Если что-то пошло не так, ErrSelect
	*/
	ByID(id string) (Model, error)

	/*
		ByModel - получение всех записей журнала по конкретной модели
	*/
	ByModel(mtp ModelType, mid string) ([]Model, error)

	/*
		Cursor - загрузка существующего курсора
	*/
	Cursor(id string) (_ Cursor, err error)

	/*
		ByDate - формирование курсора перебора по дате
	*/
	ByDate(from, to time.Time, page uint, services ...string) (_ Cursor, err error)

	/*
		ByModelDate - формирование курсора перебора по модели и дате
	*/
	ByModelDate(mtp ModelType, mid string, from, to time.Time, page uint, services ...string) (_ Cursor, err error)

	/*
		ImportEntries - массовая загрузка сразу нескольких моделей
	*/
	ImportEntries(...*Entry) error
}

// Model - запись журнала в БД
type Model interface {
	/*
		Import - копирование основного представления в модель и сохранение в БД.
	*/
	Import(*Entry) error

	/*
		Export - основное представление записи журнала.
	*/
	Export(withCrash bool) (*Entry, error)

	/*
		ExportAPI - представление для выдачи журнала в файлах и пакетах
	*/
	ExportAPI(log Provider) *API

	/*
		ExportMonitoring - представление для выдачи журнала в мониторинге
	*/
	ExportMonitoring(log Provider) *ViewMonitoring
}

// Cursor - модель для крупных выборок с постраничкой
type Cursor interface {
	ID() string
	Empty() bool

	// NextPage - Подгрузка следующей страницы (но, возможно, с изменением размера)
	NextPage(size uint, services ...string) ([]Model, error)
}

// Logger - обертка для записи журнала в консольку
type Logger interface {
	Print(tpl string, args ...interface{})
	Error(tpl string, args ...interface{})
}

// CrashHandler - обработчик вызываемый при Crash
type CrashHandler func(report *crash.Report, chain []*Stage)

// Ошибки реализаций
var (
	ErrSelect   = errx.New("Ошибка загрузки записи журнала").WithReason(errx.ErrInternal)
	ErrInsert   = errx.New("Ошибка сохранения записи журнала").WithReason(errx.ErrInternal)
	ErrNotFound = errx.New("Не найдены подходящие записи журнала").WithReason(errx.ErrNotFound)
	ErrValidate = errx.New("Ошибка валидации входных данных").WithReason(errx.ErrBadRequest)
)
