package crash

import (
	"github.com/shestakovda/errx"
)

var DatabaseAPI uint16 = 1 // TODO: Deprecated

const BlankLink = "about:blank"
const EnvBaseURL = "ERR_BASE_URL"

const (
	ModelCrash        uint16 = 32
	IndexCrashCreated uint16 = 33
	IndexCrashMessage uint16 = 34
	IndexCrashCode    uint16 = 35
)

var UnknownErrMsg = "Ошибка обработки запроса"

var (
	ErrSelect     = errx.New("Ошибка загрузки отчета об ошибке")
	ErrInsert     = errx.New("Ошибка сохранения отчета об ошибке")
	ErrNotFound   = errx.New("Не найден подходящий отчет об ошибке")
	ErrIDValidate = errx.New("Некорректный идентификатор ошибки")
)
