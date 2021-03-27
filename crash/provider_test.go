package crash

import (
	"net/http"

	"github.com/shestakovda/errx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ProviderSuite struct {
	suite.Suite
}

//nolint:funlen
func (s *ProviderSuite) TestReport() {
	// Накидываем ошибок
	someError1 := errx.New("some error 1").WithReason(assert.AnError)
	someError2 := errx.New("some error 2").WithReason(assert.AnError).WithDetail("Ашипке: %d", 42)
	someError3 := errx.New("some error 3").WithDebug(errx.Debug{"id": 42})

	// Регистрируем обработчики
	prv := NewTestProvider()
	prv.Register(http.StatusForbidden, 3, "title1", someError1)
	prv.Register(http.StatusUnauthorized, 5, "title2", assert.AnError)

	// Заглушка, если нет ошибки
	s.Nil(prv.Report(nil))

	// Что указываем, то и получаем, должна быть title2
	if rep := prv.Report(assert.AnError); s.NotNil(rep) {
		s.Equal("#testing4015", rep.Link)
		s.Equal("title2", rep.Title)
		s.Equal(http.StatusUnauthorized, int(rep.Status))
		s.Len(rep.Entries, 1)
		s.Equal(assert.AnError.Error(), rep.Entries[0].Text)
		s.Empty(rep.Entries[0].Detail)
		s.Empty(rep.Entries[0].Stack)
		s.Empty(rep.Entries[0].Debug)
	}

	// Поскольку someError1 указана раньше, то должна быть title1
	if rep := prv.Report(someError1); s.NotNil(rep) {
		s.Equal("#testing4033", rep.Link)
		s.Equal("title1", rep.Title)
		s.Equal(http.StatusForbidden, int(rep.Status))
		s.Len(rep.Entries, 2)
		s.Equal("some error 1", rep.Entries[0].Text)
		s.Equal(assert.AnError.Error(), rep.Entries[1].Text)
		s.Empty(rep.Entries[0].Detail)
		s.Empty(rep.Entries[0].Debug)
		s.Len(rep.Entries[0].Stack, 7)
	}

	// Поскольку someError2 наследуется, то хотя она не зарегана, все равно должна быть title2 по наследнику
	if rep := prv.Report(someError2); s.NotNil(rep) {
		s.Equal("#testing4015", rep.Link)
		s.Equal("title2", rep.Title)
		s.Equal(http.StatusUnauthorized, int(rep.Status))
		s.Len(rep.Entries, 2)
		s.Equal("some error 2", rep.Entries[0].Text)
		s.Equal(assert.AnError.Error(), rep.Entries[1].Text)
		s.Equal("Ашипке: 42", rep.Entries[0].Detail)
		s.Empty(rep.Entries[0].Debug)
		s.Len(rep.Entries[0].Stack, 7)
	}

	// Поскольку someError3 не наследуется и не зарегана, то ошибка по-умолчанию
	if rep := prv.Report(someError3); s.NotNil(rep) {
		s.Equal("#testing5009", rep.Link)
		s.Equal(UnknownErrMsg, rep.Title)
		s.Equal(http.StatusInternalServerError, int(rep.Status))
		s.Len(rep.Entries, 1)
		s.Equal("some error 3", rep.Entries[0].Text)
		s.Equal(map[string]string{"id": "42"}, rep.Entries[0].Debug)
		s.Empty(rep.Entries[0].Detail)
		s.Len(rep.Entries[0].Stack, 7)
	}
}
