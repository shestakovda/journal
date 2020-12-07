package journal

import "strconv"

var modelTypes = make(map[int]ModelType, 64)

func getType(id int) ModelType {
	if mtp := modelTypes[id]; mtp != nil {
		return mtp
	}

	return new(journalModels)
}

func regType(types ...ModelType) {
	for _, mtp := range types {
		if mtp != nil {
			modelTypes[mtp.ID()] = mtp
		}
	}
}

type journalModels int

func (m journalModels) ID() int { return int(m) }
func (m journalModels) String() string {
	switch m {
	case ModelTypeUnknown:
		return "unknown"
	case ModelTypeCrash:
		return "crash"
	default:
		return strconv.Itoa(int(m))
	}
}

func init() {
	regType(
		ModelTypeUnknown,
		ModelTypeCrash,
	)
}
