package ssa

type Type byte

const (
	// TypeI8 represents an integer type with 8 bits.
	TypeI8 Type = 1 + iota

	// TypeI16 represents an integer type with 16 bits.
	TypeI16

	// TypeI32 represents an integer type with 32 bits.
	TypeI32

	// TypeI64 represents an integer type with 64 bits.
	TypeI64

	// TypeF32 represents 32-bit floats in the IEEE 754.
	TypeF32

	// TypeF64 represents 64-bit floats in the IEEE 754.
	TypeF64

	// TODO: SIMD, ref types!
)

func (t Type) String() (ret string) {
	switch t {
	case TypeI8:
		return "i8"
	case TypeI16:
		return "i16"
	case TypeI32:
		return "i32"
	case TypeI64:
		return "i64"
	case TypeF32:
		return "f32"
	case TypeF64:
		return "f64"
	}
	return
}
