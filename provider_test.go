package journal

import (
	"net/http"

	"github.com/shestakovda/errx"
	"github.com/shestakovda/journal/crash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	TestNum   = 4
	TestTitle = "Доступ запрещен"
)

var ErrTest = errx.New("some test err")

type ProviderSuite struct {
	suite.Suite

	prv Provider
	log *StrLogger
	drv *MockDriver
	crp crash.Provider
}

func (s *ProviderSuite) SetupTest() {
	s.log = new(StrLogger)
	s.drv = new(MockDriver)
	s.crp = crash.NewTestProvider()
	s.prv = NewProvider(1, s.crp, s.drv, s.log, "")
	s.crp.Register(http.StatusForbidden, TestNum, TestTitle, errx.ErrForbidden)
}

func (s *ProviderSuite) TearDownTest() {
	s.drv.AssertExpectations(s.T())
}

func (s *ProviderSuite) TestPrint() {

	// Не должны сюда попасть
	if s.prv.V(2) {
		s.True(false)
	}

	// Должны попасть и записать строку
	if s.prv.V(1) {
		s.prv.Print("ololo %s %d", "test", 42)
	}

	// Должны записать данные парочке моделей
	if s.prv.V(0) {
		s.prv.Model(42, "", "empty %s", "id")
		s.prv.Model(42, "eventID", "some %s", "comment")
	}

	// Должны записать данные о модели с ошибкой
	s.prv.Crash(ErrTest.WithReason(errx.ErrForbidden))

	// Пусть сохранение пройдет с ошибкой
	s.drv.On("InsertEntry", mock.Anything).Return(assert.AnError).Once()

	// Должны записать в лог и вызвать сохранение
	s.prv.Close()

	// Поскольку у нас была ошибка, должен быть сделан Flush
	// Проверим, какой лог получился, но только кусками, потому что много случайных данных
	s.Contains(s.log.Result, "eventID (42)")
	s.Contains(s.log.Result, "ololo test 42")
	s.Contains(s.log.Result, "empty id")
	s.Contains(s.log.Result, "some comment")
	s.Contains(s.log.Result, "[ 403 ] Доступ запрещен")
	s.Contains(s.log.Result, "|-> some test err")
	s.Contains(s.log.Result, "|-> 403 Forbidden")
	s.Contains(s.log.Result, "[ 500 ] Ошибка обработки запроса")
	s.Contains(s.log.Result, assert.AnError.Error())

	// Если хочется посмотреть на лог вживую
	// glog.Errorf(s.log.Result)
	// s.True(false)
}

type MockDriver struct{ mock.Mock }

func (m *MockDriver) InsertEntry(e *Entry) error { return m.Called(e).Error(0) }
