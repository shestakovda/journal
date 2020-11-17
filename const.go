package journal

import "github.com/shestakovda/errx"

const verJournalV1 = 1

const (
	ModelJournal       uint16 = 36
	IndexJournalStart  uint16 = 37
	IndexJournalEntity uint16 = 38
)

// Ошибки реализаций
var (
	ErrSelect   = errx.New("Ошибка загрузки записи журнала").WithReason(errx.ErrInternal)
	ErrInsert   = errx.New("Ошибка сохранения записи журнала").WithReason(errx.ErrInternal)
	ErrNotFound = errx.New("Не найдены подходящие записи журнала").WithReason(errx.ErrNotFound)
	ErrValidate = errx.New("Ошибка валидации входных данных").WithReason(errx.ErrBadRequest)
)
