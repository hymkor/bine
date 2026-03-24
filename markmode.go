package bine

type markMode interface {
	Contains(pos int64, cursor int64) bool
	Toggle(pos int64) markMode
	Range(cursor int64) (from, to int64, ok bool)
}

type noMarking struct{}

func (noMarking) Contains(pos, cursor int64) bool {
	return false
}

func (noMarking) Toggle(pos int64) markMode {
	return marking{address: pos}
}

func (noMarking) Range(cursor int64) (from, to int64, ok bool) {
	return cursor, cursor, false
}

type marking struct {
	address int64
}

func (m marking) Contains(pos, cursor int64) bool {
	if m.address < cursor {
		return m.address <= pos && pos <= cursor
	}
	return cursor <= pos && pos <= m.address
}

func (marking) Toggle(_ int64) markMode {
	return noMarking{}
}

func (m marking) Range(cursor int64) (from, to int64, ok bool) {
	if m.address < cursor {
		return m.address, cursor, true
	}
	return cursor, m.address, true
}
