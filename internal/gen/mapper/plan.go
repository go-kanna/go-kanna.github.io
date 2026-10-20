package mapper

import (
	"fmt"
	"go/types"
	"strings"
)

// funcPlan describes one generated mapping function.
type funcPlan struct {
	name         string
	src          types.Type
	dst          types.Type
	returnsError bool
	fields       []fieldPlan
	dstOneofs    []dstOneof
	srcOneofs    []srcOneof
}

// fieldPlan describes how one destination field is populated.
type fieldPlan struct {
	dstName string
	dstType types.Type
	read    readAccess
	conv    op
}

// readAccess describes how the source value is read: a plain field access
// or a nil-safe getter call.
type readAccess struct {
	name   string
	getter bool
	typ    types.Type // type the read yields
}

// oneofArm is one implementation of a oneof interface: a struct that wraps the
// arm's value in its single field, as protoc-gen-go writes them.
type oneofArm struct {
	wrapper *types.Named
	field   *types.Var
}

// dstOneof fills a destination oneof field from the first of its arms whose
// source field is set.
type dstOneof struct {
	dstName string
	arms    []dstArm
}

// dstArm is one arm of a dstOneof: the source field that carries it, the
// wrapper it is stored in, and how the set value becomes the wrapper's field.
type dstArm struct {
	oneofArm

	read  readAccess
	deref bool // convert what the pointer read points at, not the pointer
	conv  op
}

// srcOneof fills destination fields from the arms of a source oneof field,
// one per arm the destination has a field for.
type srcOneof struct {
	via  *types.Var // the interface-typed source field
	read readAccess
	arms []srcArm
}

// srcArm is one arm of a srcOneof: the wrapper the value arrives in and the
// destination field it goes to.
type srcArm struct {
	oneofArm

	dstName string
	dstType types.Type
	conv    op
}

// op describes how a source value becomes a destination value. Ops nest
// for composite conversions (pointers, slices).
type op interface {
	mayFail() bool
}

// opDirect assigns the value as-is.
type opDirect struct{}

// opConvert calls a registered converter.
type opConvert struct{ conv converter }

// opMapper calls another generated mapping function.
type opMapper struct{ plan *funcPlan }

// opTypeConv applies a Go type conversion.
type opTypeConv struct{ dst types.Type }

// opDeref dereferences a source pointer, leaving the destination zero
// when the source is nil.
type opDeref struct{ elem op }

// opAddr stores the converted value in a temporary and takes its address.
type opAddr struct{ elem op }

// opNilGuard passes a source pointer on to elem only when it is not nil,
// leaving the destination nil otherwise.
type opNilGuard struct{ elem op }

// opSlice converts a slice element-wise; nil maps to nil.
type opSlice struct {
	dst  types.Type // destination slice type
	elem op
}

// Asserted on values, not pointers: every op is constructed as a value, and a
// pointer assertion would keep passing if a method moved to a pointer receiver
// while the construction sites stopped satisfying the interface.
var (
	_ op = opDirect{}
	_ op = opConvert{}
	_ op = opMapper{}
	_ op = opTypeConv{}
	_ op = opDeref{}
	_ op = opAddr{}
	_ op = opNilGuard{}
	_ op = opSlice{}
)

func (opDirect) mayFail() bool     { return false }
func (o opConvert) mayFail() bool  { return o.conv.hasErr }
func (o opMapper) mayFail() bool   { return o.plan.returnsError }
func (opTypeConv) mayFail() bool   { return false }
func (o opDeref) mayFail() bool    { return o.elem.mayFail() }
func (o opAddr) mayFail() bool     { return o.elem.mayFail() }
func (o opNilGuard) mayFail() bool { return o.elem.mayFail() }
func (o opSlice) mayFail() bool    { return o.elem.mayFail() }

// ops lists every conversion the function performs.
func (p *funcPlan) ops() []op {
	var ops []op
	for _, f := range p.fields {
		ops = append(ops, f.conv)
	}
	for _, o := range p.dstOneofs {
		for _, a := range o.arms {
			ops = append(ops, a.conv)
		}
	}
	for _, o := range p.srcOneofs {
		for _, a := range o.arms {
			ops = append(ops, a.conv)
		}
	}
	return ops
}

// describe renders a compact, human-readable form of the plan for tests
// and debug output.
func (p *funcPlan) describe() string {
	var b strings.Builder
	ret := typeLabel(p.dst)
	if p.returnsError {
		ret = "(" + ret + ", error)"
	}
	fmt.Fprintf(&b, "%s(%s) %s", p.name, typeLabel(p.src), ret)
	for _, f := range p.fields {
		fmt.Fprintf(&b, "\n  %s = %s %s", f.dstName, describeRead(f.read), describeOp(f.conv))
	}
	for _, o := range p.dstOneofs {
		fmt.Fprintf(&b, "\n  %s = oneof", o.dstName)
		for _, a := range o.arms {
			val := describeRead(a.read)
			if a.deref {
				val = "*" + val
			}
			fmt.Fprintf(&b, "\n    %s.%s = %s %s", a.wrapper.Obj().Name(), a.field.Name(), val, describeOp(a.conv))
		}
	}
	for _, o := range p.srcOneofs {
		fmt.Fprintf(&b, "\n  oneof %s", describeRead(o.read))
		for _, a := range o.arms {
			fmt.Fprintf(&b, "\n    %s = %s.%s %s", a.dstName, a.wrapper.Obj().Name(), a.field.Name(), describeOp(a.conv))
		}
	}
	return b.String()
}

func describeRead(r readAccess) string {
	if r.getter {
		return "." + r.name + "()"
	}
	return "." + r.name
}

func describeOp(o op) string {
	switch v := o.(type) {
	case opDirect:
		return "direct"
	case opConvert:
		kind := "conv"
		if v.conv.hasErr {
			kind = "convE"
		}
		return kind + ":" + v.conv.fn.Name()
	case opMapper:
		return "map:" + v.plan.name
	case opTypeConv:
		return "cast:" + typeLabel(v.dst)
	case opDeref:
		return "deref(" + describeOp(v.elem) + ")"
	case opAddr:
		return "addr(" + describeOp(v.elem) + ")"
	case opNilGuard:
		return "guard(" + describeOp(v.elem) + ")"
	case opSlice:
		return "slice(" + describeOp(v.elem) + ")"
	default:
		return fmt.Sprintf("unknown(%T)", o)
	}
}
