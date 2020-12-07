package journal

import (
	"time"

	"github.com/shestakovda/journal/models"
)

func newFdbxStage(s *Stage) *fdbxStage {
	return &fdbxStage{
		dur: s.Wait,
		msg: s.Text,
		mid: s.EnID,
		mtp: getType(s.Type),
	}
}

func loadFdbxStage(s *models.FdbxStageT) *fdbxStage {
	return &fdbxStage{
		msg: s.Msg,
		mid: s.Mid,
		mtp: getType(int(s.Mtp)),
		dur: time.Duration(s.Dur),
	}
}

type fdbxStage struct {
	dur time.Duration
	mtp ModelType
	mid string
	msg string
}

func (s *fdbxStage) Export() *Stage {
	return &Stage{
		Wait: s.dur,
		Text: s.msg,
		EnID: s.mid,
		Type: s.mtp.ID(),
	}
}

func (s *fdbxStage) ExportAPI() *StageAPI {
	return &StageAPI{
		Name: s.msg,
		Wait: s.dur,
	}
}

func (s *fdbxStage) ExportMonitoring() *StageMonitoring {
	v := &StageMonitoring{
		Name: s.msg,
		Time: uint64(s.dur),
		Wait: s.dur.String(),
	}

	if s.mid != "" {
		v.EnID = s.mid
		v.Type = s.mtp.String()
	}

	return v
}

func (s *fdbxStage) dump() *models.FdbxStageT {
	return &models.FdbxStageT{
		Msg: s.msg,
		Mid: s.mid,
		Dur: int64(s.dur),
		Mtp: int32(s.mtp.ID()),
	}
}
