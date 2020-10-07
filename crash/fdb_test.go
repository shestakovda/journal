package crash

import (
	"fmt"
	"net/http"
	"time"

	"github.com/shestakovda/errx"
	"github.com/shestakovda/fdbx"
	"github.com/shestakovda/typex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FdbSuite struct {
	suite.Suite

	rep *Report
}

func (s *FdbSuite) SetupTest() {
	s.rep = &Report{
		ID:      "ID",
		Link:    "Link",
		Title:   "Title",
		Status:  uint16(http.StatusNotFound),
		Created: time.Date(2020, 04, 12, 23, 32, 55, 0, time.UTC),
		Entries: []*ReportEntry{
			{
				Text:   "text 1",
				Detail: "detail 1",
				Stack:  []string{"stack 1-1", "stack 1-2"},
				Debug: map[string]string{
					"key 1": "value 1",
				},
			}, {
				Text:  "text 2",
				Stack: []string{"stack 2-1", "stack 2-2"},
				Debug: map[string]string{
					"key 2": "value 2",
				},
			}, {
				Text:   "text 3",
				Detail: "detail 3",
				Stack:  []string{"stack 3-1", "stack 3-2"},
				Debug: map[string]string{
					"key 3": "value 3",
				},
			},
		},
	}
}

func (s *FdbSuite) TestImport() {
	fdb := new(fdbx.MockConn)

	s.NoError(fdb.Tx(func(db fdbx.DB) error {
		fdb.FAt = func(id uint16) fdbx.DB {
			s.Equal(DatabaseAPI, id)
			return db
		}

		mod := NewFactoryFDB(db).New()

		fdb.FSave = func(fdbx.RecordHandler, ...fdbx.Record) error { return assert.AnError }

		if err := mod.Import(s.rep); s.Error(err) {
			s.True(errx.Is(err, ErrInsert))
			s.True(errx.Is(err, assert.AnError))
		}

		fdb.FSave = func(hdl fdbx.RecordHandler, recs ...fdbx.Record) error {
			s.Nil(hdl)
			s.Len(recs, 1)
			s.Equal(mod, recs[0])
			return nil
		}

		if err := mod.Import(s.rep); s.NoError(err) {
			s.Equal(s.rep.AsRFC(), mod.ExportRFC())
		}

		s.rep.Link = ""

		if err := mod.Import(s.rep); s.NoError(err) {
			s.Equal(s.rep, mod.Export())
			s.Equal(s.rep.AsRFC(), mod.ExportRFC())
		}

		return nil
	}))
}

func (s *FdbSuite) TestByID() {
	fdb := new(fdbx.MockConn)

	s.NoError(fdb.Tx(func(db fdbx.DB) (err error) {
		fdb.FAt = func(id uint16) fdbx.DB {
			s.Equal(DatabaseAPI, id)
			return db
		}

		fac := NewFactoryFDB(db)

		if _, err = fac.ByID(""); s.Error(err) {
			s.True(errx.Is(err, typex.ErrUUIDEmpty))
		}

		if _, err = fac.ByID("unknown"); s.Error(err) {
			s.True(errx.Is(err, typex.ErrUUIDInvalid))
		}

		uid := typex.NewUUID().Hex()

		fdb.FLoad = func(fdbx.RecordHandler, ...fdbx.Record) error { return assert.AnError }

		if _, err = fac.ByID(uid); s.Error(err) {
			s.True(errx.Is(err, ErrSelect))
			s.True(errx.Is(err, assert.AnError))
		}

		fdb.FLoad = func(fdbx.RecordHandler, ...fdbx.Record) error { return fdbx.ErrRecordNotFound }

		if _, err = fac.ByID(uid); s.Error(err) {
			s.True(errx.Is(err, ErrNotFound))
			s.True(errx.Is(err, fdbx.ErrRecordNotFound))
		}

		fdb.FLoad = func(fdbx.RecordHandler, ...fdbx.Record) error { return nil }

		var mod Model
		if mod, err = fac.ByID(uid); s.NoError(err) {
			s.Equal(uid, mod.ExportRFC().ID)
		}

		return nil
	}))
}

func (s *FdbSuite) TestAsRFC() {
	rfc := s.rep.AsRFC()
	s.Equal(s.rep.ID, rfc.ID)
	s.Equal(s.rep.Link, rfc.Link)
	s.Equal(s.rep.Title, rfc.Title)
	s.Equal(s.rep.Status, rfc.Status)
	s.Equal("detail 3 => detail 1", rfc.Detail)

	s.rep.Link = ""
	s.Equal(BlankLink, s.rep.AsRFC().Link)
}

func (s *FdbSuite) TestFormat() {
	s.Equal(`
[ 404 ] Title
`, fmt.Sprintf("\n%s\n", s.rep))

	s.Equal(`
[ 404 ] Title
|-> text 1 (detail 1)
|   key 1: value 1
|-> text 2
|   key 2: value 2
|-> text 3 (detail 3)
|   key 3: value 3
`, fmt.Sprintf("\n%v\n", s.rep))

	s.Equal(`
[ 404 ] Title
|-> text 1 (detail 1)
|   key 1: value 1
|       stack 1-1
|       stack 1-2
|-> text 2
|   key 2: value 2
|       stack 2-1
|       stack 2-2
|-> text 3 (detail 3)
|   key 3: value 3
|       stack 3-1
|       stack 3-2
`, fmt.Sprintf("\n%+v\n", s.rep))

}
