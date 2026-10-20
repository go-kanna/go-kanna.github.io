// Package errcases holds type pairs that must fail plan resolution, one
// pair per rule.
package errcases

import "time"

// UnmappedSrc lacks a counterpart for UnmappedDst.Bar.
type UnmappedSrc struct{ Foo string }

// UnmappedDst has a field with no source.
type UnmappedDst struct{ Foo, Bar string }

// TagMissingSrc has no field named Nope.
type TagMissingSrc struct{ X string }

// TagMissingDst tags its field to a nonexistent counterpart.
type TagMissingDst struct {
	X string `map:"Nope"`
}

// ConflictSrc tags A to Out while ConflictDst.Out names B.
type ConflictSrc struct {
	A string `map:"Out"`
	B string
}

// ConflictDst declares the other half of the tag conflict.
type ConflictDst struct {
	Out string `map:"B"`
}

// AmbiguousSrc fold-matches AmbiguousDst.Ident twice.
type AmbiguousSrc struct{ IDent, IdEnt string }

// AmbiguousDst is the ambiguous match target.
type AmbiguousDst struct{ Ident string }

// OneofSrc pairs with OneofDst.
type OneofSrc struct{ Payload string }

// OneofDst has an interface field like a protobuf oneof, but nothing
// implements it, so it is not one.
type OneofDst struct{ Payload isPayload }

type isPayload interface{ isPayload() }

// ArmlessSrc has no field for the arm of ArmlessDst.Kind.
type ArmlessSrc struct{ Other string }

// ArmlessDst is a oneof whose arm has no source.
type ArmlessDst struct{ Kind isArmless }

type isArmless interface{ isArmless() }

// Armless_X is the arm of ArmlessDst.Kind.
type Armless_X struct{ X *string }

func (*Armless_X) isArmless() {}

// ValueArmSrc has the arm's field as a value, which cannot say whether it is
// set.
type ValueArmSrc struct{ X string }

// ValueArmDst is a oneof whose arm's source is not nilable.
type ValueArmDst struct{ Kind isValueArm }

type isValueArm interface{ isValueArm() }

// ValueArm_X is the arm of ValueArmDst.Kind.
type ValueArm_X struct{ X string }

func (*ValueArm_X) isValueArm() {}

// BothSrc is a oneof whose arm matches an arm of BothDst, which is not
// supported.
type BothSrc struct{ Kind isBothSrc }

type isBothSrc interface{ isBothSrc() }

// BothSrc_X is the arm of BothSrc.Kind.
type BothSrc_X struct{ X *string }

func (*BothSrc_X) isBothSrc() {}

// BothDst is the oneof BothSrc would fill.
type BothDst struct{ Kind isBothDst }

type isBothDst interface{ isBothDst() }

// BothDst_X is the arm of BothDst.Kind.
type BothDst_X struct{ X *string }

func (*BothDst_X) isBothDst() {}

// OpaqueSrc pairs with OpaqueDst.
type OpaqueSrc struct{ Id string }

// OpaqueDst mimics the protobuf opaque API: no exported fields, getters
// only.
type OpaqueDst struct{ id string }

// GetId returns the hidden field.
func (o *OpaqueDst) GetId() string { return o.id }

// BadTagSrc pairs with BadTagDst.
type BadTagSrc struct{ X string }

// BadTagDst carries an invalid map tag value.
type BadTagDst struct {
	X string `map:"9x"`
}

// LossySrc pairs with LossyDst; float64 to int must not auto-convert.
type LossySrc struct{ V float64 }

// LossyDst is the lossy conversion target.
type LossyDst struct{ V int }

// TagUnexportedSrc tags its field to an unexported counterpart, which
// can never be mapped.
type TagUnexportedSrc struct {
	A string `map:"hidden"`
}

// TagUnexportedDst has only the unexported field the tag points at.
type TagUnexportedDst struct{ hidden string }

// NarrowSrc pairs with NarrowDst; int64 to int32 narrows and must not
// auto-convert.
type NarrowSrc struct{ V int64 }

// NarrowDst is the narrowing target.
type NarrowDst struct{ V int32 }

// SignSrc pairs with SignDst; int to uint changes sign and must not
// auto-convert.
type SignSrc struct{ V int }

// SignDst is the sign-change target.
type SignDst struct{ V uint }

// NeedConvSrc pairs with NeedConvDst; time.Time to string requires a
// converter.
type NeedConvSrc struct{ When time.Time }

// NeedConvDst is the suggestion-message target.
type NeedConvDst struct{ When string }

// PromotedBase carries a field its author excluded from mapping.
type PromotedBase struct {
	Secret string `map:"-"`
}

// PromotedSrc promotes Secret through embedding, where the tag has to keep
// applying.
type PromotedSrc struct {
	PromotedBase
	Name string
}

// PromotedDst has the field the excluded one would otherwise fill.
type PromotedDst struct {
	Secret string
	Name   string
}
