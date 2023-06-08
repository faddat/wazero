package ssa

const poolPageSize = 128

type pool[T any] struct {
	pages            []*[poolPageSize]T
	allocated, index int
}

func newPool[T any]() pool[T] {
	var ret pool[T]
	ret.reset()
	return ret
}

func (p *pool[T]) allocate() *T {
	if p.index == poolPageSize {
		if len(p.pages) == cap(p.pages) {
			p.pages = append(p.pages, new([poolPageSize]T))
		} else {
			i := len(p.pages)
			p.pages = p.pages[:i+1]
			if p.pages[i] == nil {
				p.pages[i] = new([poolPageSize]T)
			}
		}
		p.index = 0
	}
	ret := &p.pages[len(p.pages)-1][p.index]
	p.index++
	p.allocated++
	return ret
}

func (p *pool[T]) view(i int) *T {
	page, index := i/poolPageSize, i%poolPageSize
	return &p.pages[page][index]
}

func (p *pool[T]) reset() {
	for _, ns := range p.pages {
		pages := ns[:]
		for i := range pages {
			var v T
			pages[i] = v
		}
	}
	p.pages = p.pages[:0]
	p.index = poolPageSize
	p.allocated = 0
}
