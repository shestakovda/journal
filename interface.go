package journal

import (
	"time"

	"github.com/shestakovda/journal/crash"
)

// ModelType - абстрактный тип модели для логирования
type ModelType interface {
	ID() int
	String() string
}

// RegisterType - регистрация типа для корректной загрузки данных из БД
func RegisterType(types ...ModelType) {
	for _, mtp := range types {
		if mtp != nil {
			modelTypes[mtp.ID()] = mtp
		}
	}
}

// Provider сборки и сохранения журнала
type Provider interface {
	/*
		V - проверка уровня логирования.

		* Принимает необходимый уровень, с которым планируется запись
		* Разрешает запись, если указанный уровень меньше или равен максимальному, заданному в конструкторе
		* Имеет сайдэффект - устанавливает указанный уровень как текущий на одну (!) следующую операцию
	*/
	V(int) bool

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
		Crash - логирование ошибки в журнал с формированием и записью отчета.

		* err - любая ошибка, которая будет возвращена как внешняя

		* Если err = nil, то возвращается тоже nil
		* Ошибка логируется как модель с идентификатором типа core.ModelTypeCrash
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

	Cursor(id string) (_ Cursor, err error)
	ByDate(from, to time.Time, page uint, services ...string) (_ Cursor, err error)
	ByModelDate(mtp ModelType, mid string, from, to time.Time, page uint, services ...string) (_ Cursor, err error)
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

	// Подгрузка следующей страницы (но, возможно, с изменением размера)
	NextPage(size uint, services ...string) ([]Model, error)
}

// Logger - обертка для записи журнала в консольку
type Logger interface {
	Print(tpl string, args ...interface{})
	Error(tpl string, args ...interface{})
}
