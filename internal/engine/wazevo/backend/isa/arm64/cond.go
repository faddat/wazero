package arm64

// cond represents a condition for conditional branches.
type cond uint8

const (
	eq cond = iota // Eq represents "equal"
	ne             // Ne represents "not equal"
	hs             // Hs represents "higher or same"
	lo             // Lo represents "lower"
	mi             // Mi represents "minus or negative result"
	pl             // Pl represents "plus or positive result"
	vs             // Vs represents "overflow set"
	vc             // Vc represents "overflow clear"
	hi             // Hi represents "higher"
	ls             // Ls represents "lower or same"
	ge             // Ge represents "greater or equal"
	lt             // Lt represents "less than"
	gt             // Gt represents "greater than"
	le             // Le represents "less than or equal"
	al             // Al represents "always"
	nv             // Nv represents "never"
)

// invert returns the inverted condition.
func (c cond) invert() cond {
	switch c {
	case eq:
		return ne
	case ne:
		return eq
	case hs:
		return lo
	case lo:
		return hs
	case mi:
		return pl
	case pl:
		return mi
	case vs:
		return vc
	case vc:
		return vs
	case hi:
		return ls
	case ls:
		return hi
	case ge:
		return lt
	case lt:
		return ge
	case gt:
		return le
	case le:
		return gt
	case al:
		return nv
	case nv:
		return al
	default:
		panic(c)
	}
}
