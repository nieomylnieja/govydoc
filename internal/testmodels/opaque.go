package testmodels

import "fmt"

// OpaqueScalar encodes a numeric value and unit as one string.
type OpaqueScalar struct {
	value int
	unit  string
}

// Unit returns the suffix of the encoded value.
func (s OpaqueScalar) Unit() string { return s.unit }

// Value returns the numeric component of the encoded value.
func (s OpaqueScalar) Value() int { return s.value }

// MarshalText encodes the value and its unit, including when used as a map key.
func (s OpaqueScalar) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%d%s", s.value, s.unit), nil
}

// StructWithOpaqueScalars contains scalar values at different property paths.
type StructWithOpaqueScalars struct {
	// Scalar is optional even when its internal components are required.
	Scalar  OpaqueScalar                  `json:"scalar"`
	Scalars OpaqueScalar                  `json:"scalars"`
	Pointer *OpaqueScalar                 `json:"pointer"`
	Nested  NestedOpaqueScalar            `json:"nested"`
	Items   []OpaqueScalar                `json:"items"`
	Mapping map[OpaqueScalar]OpaqueScalar `json:"mapping"`
	Escaped OpaqueScalar                  `json:"scalar.value"`
}

// NestedOpaqueScalar places an opaque scalar below another struct.
type NestedOpaqueScalar struct {
	Scalar OpaqueScalar `json:"scalar"`
}
