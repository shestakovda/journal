package journal

import (
	"net/http"
	"time"

	"github.com/shestakovda/errx"
	"github.com/shestakovda/fdbx"
	"github.com/shestakovda/journal/crash"
	"github.com/shestakovda/typex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FdbSuite struct {
	suite.Suite

	entry *Entry
}

func (s *FdbSuite) SetupTest() {
	failID := typex.NewUUID().Hex()
	s.entry = &Entry{
		ID:    "entryID",
		Start: time.Date(2020, 04, 13, 12, 02, 35, 0, time.UTC),
		Total: time.Minute,
		Chain: []*Stage{
			{
				EnID: "eventID",
				Text: "text1",
				Wait: time.Second,
				Verb: 2,
				Type: 42,
			},
			{
				EnID: "",
				Text: "text2",
				Wait: time.Minute,
				Verb: 1,
				Type: ModelTypeUnknown.ID(),
			},
			{
				EnID: failID,
				Text: assert.AnError.Error(),
				Wait: time.Millisecond,
				Verb: 1,
				Type: ModelTypeCrash.ID(),
				Fail: &crash.Report{
					ID:      failID,
					Link:    "Link",
					Title:   "Title",
					Status:  uint16(http.StatusNotFound),
					Created: time.Date(2020, 04, 12, 23, 32, 55, 0, time.UTC),
					Entries: []*crash.ReportEntry{
						{
							Text:   "text",
							Detail: "detail",
							Stack:  []string{"stack1", "stack2"},
							Debug:  map[string]string{"key": "value"},
						},
					},
				},
			},
		},
	}
}

func (s *FdbSuite) TestImport() {
	fdb := new(fdbx.MockConn)

	s.NoError(fdb.Tx(func(db fdbx.DB) error {
		fdb.FAt = func(id uint16) fdbx.DB {
			s.Equal(crash.DatabaseAPI, id)
			return db
		}

		fac := NewFactoryFDB(fdb, db)
		mod := fac.New()

		if err := mod.Import(s.entry); s.Error(err) {
			s.True(errx.Is(err, errx.ErrBadRequest))
		}

		s.entry.ID = typex.NewUUID().Hex()

		fdb.FSave = func(hdl fdbx.RecordHandler, recs ...fdbx.Record) error {
			s.Nil(hdl)
			s.Len(recs, 1)
			if _, ok := recs[0].(Model); ok {
				return assert.AnError
			}
			return nil
		}

		if err := mod.Import(s.entry); s.Error(err) {
			s.True(errx.Is(err, ErrInsert))
			s.True(errx.Is(err, assert.AnError))
		}

		fdb.FSave = func(hdl fdbx.RecordHandler, recs ...fdbx.Record) error {
			s.Nil(hdl)
			s.Len(recs, 1)
			if rec, ok := recs[0].(Model); ok {
				s.Equal(rec, recs[0])
			} else {
				s.Equal(s.entry.Chain[2].Fail, recs[0].(crash.Model).Export())
			}
			return nil
		}

		fdb.FLoad = func(hdl fdbx.RecordHandler, recs ...fdbx.Record) error {
			s.Nil(hdl)
			s.Len(recs, 1)
			if rec, ok := recs[0].(crash.Model); s.True(ok) {
				s.NoError(rec.Import(s.entry.Chain[2].Fail))
			}
			return nil
		}

		if err := mod.Import(s.entry); s.NoError(err) {
			if e, err := mod.Export(true); s.NoError(err) {
				s.Equal(s.entry, e)
			}
		}

		rec := mod.(*fdbModel)
		rec2 := fac.New().(*fdbModel)
		if buf, err := rec.FdbxMarshal(); s.NoError(err) {
			if err := rec2.FdbxUnmarshal(buf); s.NoError(err) {
				if e1, err := rec.Export(false); s.NoError(err) {
					if e2, err := rec.Export(false); s.NoError(err) {
						s.Equal(e1, e2)
					}
				}
			}
		}

		return nil
	}))
}

func (s *FdbSuite) TestByID() {
	fdb := new(fdbx.MockConn)

	s.NoError(fdb.Tx(func(db fdbx.DB) (err error) {
		fdb.FAt = func(id uint16) fdbx.DB {
			s.Equal(crash.DatabaseAPI, id)
			return db
		}

		fac := NewFactoryFDB(fdb, db)

		if _, err = fac.ByID(""); s.Error(err) {
			s.True(errx.Is(err, errx.ErrBadRequest))
		}

		if _, err = fac.ByID("unknown"); s.Error(err) {
			s.True(errx.Is(err, errx.ErrBadRequest))
		}

		uid := typex.NewUUID()

		fdb.FLoad = func(fdbx.RecordHandler, ...fdbx.Record) error { return assert.AnError }

		if _, err = fac.ByID(uid.Hex()); s.Error(err) {
			s.True(errx.Is(err, ErrSelect))
			s.True(errx.Is(err, assert.AnError))
		}

		fdb.FLoad = func(fdbx.RecordHandler, ...fdbx.Record) error { return fdbx.ErrRecordNotFound }

		if _, err = fac.ByID(uid.Hex()); s.Error(err) {
			s.True(errx.Is(err, ErrNotFound))
			s.True(errx.Is(err, fdbx.ErrRecordNotFound))
		}

		fdb.FLoad = func(fdbx.RecordHandler, ...fdbx.Record) error { return nil }

		var mod Model
		if mod, err = fac.ByID(uid.Hex()); s.NoError(err) {
			s.Equal(uid, mod.(*fdbModel).id)
		}

		return nil
	}))
}
