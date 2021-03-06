package journal_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/shestakovda/errx"
	fdbv1 "github.com/shestakovda/fdbx"
	"github.com/shestakovda/fdbx/v2/db"
	"github.com/shestakovda/fdbx/v2/mvcc"
	"github.com/shestakovda/journal"
	"github.com/shestakovda/journal/crash"
	"github.com/shestakovda/typex"
	"github.com/stretchr/testify/suite"
)

func TestFactory(t *testing.T) {
	suite.Run(t, new(journal.FdbSuite))
}

func TestProvider(t *testing.T) {
	suite.Run(t, new(journal.ProviderSuite))
}

func TestInterface(t *testing.T) {
	suite.Run(t, new(InterfaceSuite))
}

type InterfaceSuite struct {
	suite.Suite

	mt  *mType
	crp crash.Provider

	entry  *journal.Entry
	entry2 *journal.Entry
	entry3 *journal.Entry
}

func (s *InterfaceSuite) SetupTest() {
	s.mt = &mType{
		id:   36,
		name: "event",
	}

	s.crp = crash.NewTestProvider()
	s.crp.Register(http.StatusForbidden, journal.TestNum, journal.TestTitle, errx.ErrForbidden)
}

func (s *InterfaceSuite) TestWorkflowFDB() {
	var cid string

	fdb, err := fdbv1.NewConn(crash.DatabaseAPI, fdbv1.ConnVersion610)
	s.Require().NoError(err)
	s.Require().NoError(fdb.ClearDB())

	// Сохраняем данные по логам
	log, rep := s.saveEntries(journal.NewDriverFDB(fdb))

	// Где-то в другом месте его можно получить по айдишке
	s.Require().NoError(fdb.Tx(func(db fdbv1.DB) error {
		cid = s.checkSaved(journal.NewFactoryFDB(fdb, db), log, rep)
		return nil
	}))

	// В след. раз загружаем этот курсор и смотрим, чот там есть
	s.Require().NoError(fdb.Tx(func(db fdbv1.DB) (exp error) {
		s.checkCursor(journal.NewFactoryFDB(fdb, db), cid)
		return nil
	}))
}

func (s *InterfaceSuite) TestWorkflowFdbx() {
	dbc, err := db.ConnectV610(0x10)
	s.Require().NoError(err)
	s.Require().NoError(dbc.Clear())

	tx, err := mvcc.Begin(dbc)
	s.Require().NoError(err)
	defer tx.Cancel()

	drv := journal.NewFdbxDriver(dbc, 0x1234, 0x4321)
	fac := journal.NewFdbxFactory(tx, 0x1234, 0x4321)

	log, rep := s.saveEntries(drv)

	// Сохраняем данные по логам
	// Где-то в другом месте его можно получить по айдишке
	// В след. раз загружаем этот курсор и смотрим, чот там есть
	s.checkCursor(fac, s.checkSaved(fac, log, rep))
}

func (s *InterfaceSuite) saveEntries(drv journal.Driver) (journal.Provider, *crash.Report) {
	log := journal.NewProvider(1, s.crp, drv, nil, "")
	log2 := log.Clone()
	log3 := log.Clone()

	// Должны записать строку
	log.Print("ololo %s %d", "test1", 41)
	log2.Print("ololo %s %d", "test2", 42)
	log3.Print("ololo %s %d", "test3", 43)

	journal.RegisterType(s.mt)

	// Должны записать данные парочке моделей
	log.Model(s.mt, "", "empty %s", "id")
	log.Model(s.mt, "eventID", "some %s", "comment1")
	log2.Model(s.mt, "eventID", "some %s", "comment2")
	log3.Model(s.mt, "eventID", "some %s", "comment3")

	// Должны записать данные о модели с ошибкой
	rep := log.Crash(journal.ErrTest.WithReason(errx.ErrForbidden))

	// Должны записать в лог и вызвать сохранение, несколько раз желательно, чтобы курсор работал
	s.entry = log.Close()
	s.entry2 = log2.Close()
	s.entry3 = log3.Close()

	return log, rep
}

func (s *InterfaceSuite) checkSaved(fac journal.Factory, log journal.Provider, rep *crash.Report) string {
	var exp error
	var cur journal.Cursor

	mod, err := fac.ByID(s.entry.ID)
	s.Require().NoError(err)

	// По крайней мере, их внешние представления должны совпадать
	if row, err := mod.Export(true); s.NoError(err) {
		s.Require().Equal(s.entry, row)

		// Сравним, как это выгружается в формате API
		api := mod.ExportAPI(log)
		s.Equal(row.ID, api.ID)
		s.Equal(row.Total, api.Total)
		s.Equal("ololo test1 41", api.Name)

		// Сравним, как это выгружается в формате мониторинга
		mon := mod.ExportMonitoring(log)
		s.Equal(row.ID, mon.ID)
		s.Equal(row.Total.String(), mon.Total)
		s.Equal("ololo test1 41", mon.Name)
		s.Equal("", mon.Stages[0].Type)
		s.Equal("", mon.Stages[0].EnID)
		s.Equal("event", mon.Stages[2].Type)
		s.Equal("eventID", mon.Stages[2].EnID)
		s.Equal("crash", mon.Stages[3].Type)
		s.Equal(rep.ID, mon.Stages[3].EnID)
	}

	// Попробуем найти по модели ошибки
	if mods, exp := fac.ByModel(journal.ModelTypeCrash, rep.ID); s.NoError(exp) && s.Len(mods, 1) {
		if row, err := mods[0].Export(true); s.NoError(err) {
			s.Equal(s.entry, row)
		}
	}

	// Попробуем найти по дате
	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)
	if cur, exp = fac.ByDate(from, to, 10); s.NoError(exp) {
		if mods, exp := cur.NextPage(10); s.NoError(exp) && s.Len(mods, 3) {
			if row, err := mods[0].Export(true); s.NoError(err) {
				s.Equal(s.entry3, row)
			}
			if row, err := mods[1].Export(true); s.NoError(err) {
				s.Equal(s.entry2, row)
			}
			if row, err := mods[2].Export(true); s.NoError(err) {
				s.Equal(s.entry, row)
			}
		}
	}

	// Попробуем найти по модели
	if cur, exp = fac.ByModelDate(s.mt, "eventID", from, to, 10); s.NoError(exp) {
		if mods, exp := cur.NextPage(1); s.NoError(exp) && s.Len(mods, 1) {
			if row, err := mods[0].Export(true); s.NoError(err) {
				s.Equal(s.entry3, row)
			}
		}
	}

	s.False(cur.Empty())

	return cur.ID()
}

func (s *InterfaceSuite) checkCursor(fac journal.Factory, cid string) {
	var exp error
	var cur journal.Cursor

	if _, exp = fac.Cursor(typex.NewUUID().Hex()); s.Error(exp) {
		s.True(errors.Is(exp, errx.ErrNotFound))
	}

	if cur, exp = fac.Cursor(cid); s.NoError(exp) {
		if mods, exp := cur.NextPage(5); s.NoError(exp) && s.Len(mods, 2) {
			if row, err := mods[0].Export(true); s.NoError(err) {
				s.Equal(s.entry2, row)
			}

			if row, err := mods[1].Export(true); s.NoError(err) {
				s.Equal(s.entry, row)
			}
		}
	}

	s.True(cur.Empty())
}

type mType struct {
	id   int
	name string
}

func (mt mType) ID() int        { return mt.id }
func (mt mType) String() string { return mt.name }
