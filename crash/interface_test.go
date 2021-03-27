package crash_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/shestakovda/errx"
	"github.com/shestakovda/fdbx"
	"github.com/stretchr/testify/suite"

	"github.com/shestakovda/journal/crash"
)

const (
	testNum   = 4
	testTitle = "Доступ запрещен"
)

var ErrTest = errx.New("some test err")
var ErrForbidden = errx.New("forbidden")

func TestFactory(t *testing.T) {
	suite.Run(t, new(crash.FdbSuite))
}

func TestProvider(t *testing.T) {
	suite.Run(t, new(crash.ProviderSuite))
}

func TestInterface(t *testing.T) {
	suite.Run(t, new(InterfaceSuite))
}

type InterfaceSuite struct {
	suite.Suite

	fdb fdbx.Conn
	prv crash.Provider
}

func (s *InterfaceSuite) SetupTest() {
	var err error

	if testing.Short() {
		s.T().SkipNow()
	}

	s.prv = crash.NewTestProvider()
	s.fdb, err = fdbx.NewConn(crash.DatabaseAPI, fdbx.ConnVersion610)
	s.Require().NoError(err)
	s.Require().NoError(s.fdb.ClearDB())

	s.prv.Register(http.StatusForbidden, testNum, testTitle, ErrForbidden)
}

//nolint:funlen
func (s *InterfaceSuite) TestWorkflow() {
	const msg = "Тебе сюда нельзя"

	// Где-то в глубинах кода мы получили ошибку
	err := ErrTest.WithReason(ErrForbidden).WithDetail(msg).WithDebug(errx.Debug{
		"Code": 42,
	})

	// Формируем отчет об ошибке
	report := s.prv.Report(err)
	report2 := s.prv.Report(err)

	// Теперь можно сохранить его где-то дальше в какой-то транзакции
	s.Require().NoError(s.fdb.Tx(func(db fdbx.DB) error {
		return crash.NewFactoryFDB(db).New().Import(report)
	}))

	// Где-то в другом месте его можно получить по айдишке
	// Попутно сохраним еще одну ошибку, чтобы их в списке было две
	var mod crash.Model
	s.Require().NoError(s.fdb.Tx(func(db fdbx.DB) (exp error) {
		fac := crash.NewFactoryFDB(db)

		if mod, exp = fac.ByID(report.ID); s.NoError(exp) {
			return fac.New().Import(report2)
		}

		return exp
	}))

	// По крайней мере, их внешние представления должны совпадать
	s.Equal(report.AsRFC(), mod.ExportRFC())

	// Должно быть можно получить по коду (в т.ч. его части) и дате
	s.Require().NoError(s.fdb.Tx(func(db fdbx.DB) (exp error) {
		code := "testing4034"
		from := time.Now().Add(-time.Hour)
		to := time.Now().Add(time.Hour)
		fac := crash.NewFactoryFDB(db)

		// Не пройдет по дате
		if list, exp := fac.ByDateCode(from, from, code); s.NoError(exp) {
			s.Empty(list)
		}

		// Не пройдет по дате
		if list, exp := fac.ByDateCode(to, to, code); s.NoError(exp) {
			s.Empty(list)
		}

		// Не пройдет по коду
		if list, exp := fac.ByDateCode(from, to, "lox"); s.NoError(exp) {
			s.Empty(list)
		}

		// Выборка только по дате, без кода
		if list, exp := fac.ByDateCode(from, to, ""); s.NoError(exp) && s.Len(list, 2) {
			s.Equal(report.AsRFC(), list[0].ExportRFC())
			s.Equal(report2.AsRFC(), list[1].ExportRFC())
		}

		// Выборка по части кода
		if list, exp := fac.ByDateCode(from, to, "test"); s.NoError(exp) && s.Len(list, 2) {
			s.Equal(report.AsRFC(), list[0].ExportRFC())
			s.Equal(report2.AsRFC(), list[1].ExportRFC())
		}

		// Выборка с полным кодом
		if list, exp := fac.ByDateCode(from, to, code); s.NoError(exp) && s.Len(list, 2) {
			s.Equal(report.AsRFC(), list[0].ExportRFC())
			s.Equal(report2.AsRFC(), list[1].ExportRFC())
		}

		return nil
	}))
}
