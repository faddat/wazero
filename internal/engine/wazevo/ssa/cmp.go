package ssa

// IntegerCmpCond represents a condition for integer comparison.
type IntegerCmpCond byte

const (
	// IntegerCmpCondEqual represents "==".
	IntegerCmpCondEqual IntegerCmpCond = iota
	// IntegerCmpCondNotEqual represents "!=".
	IntegerCmpCondNotEqual
	// IntegerCmpCondSignedLessThan represents Signed "<".
	IntegerCmpCondSignedLessThan
	// IntegerCmpCondSignedGreaterThanOrEqual represents Signed ">=".
	IntegerCmpCondSignedGreaterThanOrEqual
	// IntegerCmpCondSignedGreaterThan represents Signed ">".
	IntegerCmpCondSignedGreaterThan
	// IntegerCmpCondSignedLessThanOrEqual represents Signed "<=".
	IntegerCmpCondSignedLessThanOrEqual
	// IntegerCmpCondUnsignedLessThan represents Unsigned "<".
	IntegerCmpCondUnsignedLessThan
	// IntegerCmpCondUnsignedGreaterThanOrEqual represents Unsigned ">=".
	IntegerCmpCondUnsignedGreaterThanOrEqual
	// IntegerCmpCondUnsignedGreaterThan represents Unsigned ">".
	IntegerCmpCondUnsignedGreaterThan
	// IntegerCmpCondUnsignedLessThanOrEqual represents Unsigned "<=".
	IntegerCmpCondUnsignedLessThanOrEqual
)

// String implements fmt.Stringer.
func (i IntegerCmpCond) String() string {
	switch i {
	case IntegerCmpCondEqual:
		return "eq"
	case IntegerCmpCondNotEqual:
		return "neq"
	case IntegerCmpCondSignedLessThan:
		return "lt_s"
	case IntegerCmpCondSignedGreaterThanOrEqual:
		return "ge_s"
	case IntegerCmpCondSignedGreaterThan:
		return "gt_s"
	case IntegerCmpCondSignedLessThanOrEqual:
		return "le_s"
	case IntegerCmpCondUnsignedLessThan:
		return "lt_u"
	case IntegerCmpCondUnsignedGreaterThanOrEqual:
		return "ge_u"
	case IntegerCmpCondUnsignedGreaterThan:
		return "gt_u"
	case IntegerCmpCondUnsignedLessThanOrEqual:
		return "le_u"
	default:
		panic("invalid integer comparison condition")
	}
}
