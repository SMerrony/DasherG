package main

type Cell struct {
	charValue                                byte
	blink, dim, reverse, underscore, protect bool
}

func (cell *Cell) set(cv byte, bl, dm, rev, under, prot bool) {
	cell.charValue = cv
	cell.blink = bl
	cell.dim = dm
	cell.reverse = rev
	cell.underscore = under
	cell.protect = prot
}

func (cell *Cell) clearToSpace() {
	cell.charValue = ' '
	cell.blink = false
	cell.dim = false
	cell.reverse = false
	cell.underscore = false
	cell.protect = false
}

func (cell *Cell) clearToSpaceIfUnprotected() {
	if !cell.protect {
		cell.clearToSpace()
	}
}

func (cell *Cell) copy(fromCell *Cell) {
	cell.charValue = fromCell.charValue
	cell.blink = fromCell.blink
	cell.dim = fromCell.dim
	cell.reverse = fromCell.reverse
	cell.underscore = fromCell.underscore
	cell.protect = fromCell.protect
}
