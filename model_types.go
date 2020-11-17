package journal

var modelTypes = make(map[int]ModelType, 64)

type journalModels int

func (m journalModels) ID() int { return int(m) }
func (m journalModels) String() string {
	switch m {
	case ModelTypeUnknown:
		return "unknown"
	case ModelTypeCrash:
		return "crash"
	default:
		return "unknown"
	}
}

const (
	ModelTypeUnknown journalModels = 0
	ModelTypeCrash   journalModels = 1
)

func init() {
	RegisterType(
		ModelTypeUnknown,
		ModelTypeCrash,
	)
}
