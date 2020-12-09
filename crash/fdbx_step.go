package crash

import "github.com/shestakovda/journal/models"

func fdbxNewStep(e *ReportEntry) *fdbxStep {
	return &fdbxStep{
		text:   e.Text,
		detail: e.Detail,
		stack:  e.Stack,
		debug:  e.Debug,
	}
}

func fdbxLoadStep(m *models.FdbxStepT) *fdbxStep {
	s := &fdbxStep{
		text:   m.Text,
		detail: m.Detail,
	}

	if len(m.Stack) > 0 {
		s.stack = m.Stack
	}

	if len(m.Debug) > 0 {
		s.debug = make(map[string]string, len(m.Debug))
		for i := range m.Debug {
			s.debug[m.Debug[i].Name] = m.Debug[i].Text
		}
	}

	return s
}

type fdbxStep struct {
	text   string
	detail string
	stack  []string
	debug  map[string]string
}

func (s *fdbxStep) Export() *ReportEntry {
	return &ReportEntry{
		Text:   s.text,
		Detail: s.detail,
		Stack:  s.stack,
		Debug:  s.debug,
	}
}

func (s *fdbxStep) ExportMonitoring() *ViewMonitoringEntry {
	return &ViewMonitoringEntry{
		Text:   s.text,
		Detail: s.detail,
		Stack:  s.stack,
		Debug:  s.debug,
	}
}

func (s *fdbxStep) dump() *models.FdbxStepT {
	mod := &models.FdbxStepT{
		Text:   s.text,
		Detail: s.detail,
		Stack:  s.stack,
		Debug:  make([]*models.FdbxDebugT, 0, len(s.debug)),
	}

	for name, text := range s.debug {
		mod.Debug = append(mod.Debug, &models.FdbxDebugT{
			Name: name,
			Text: text,
		})
	}

	return mod
}
