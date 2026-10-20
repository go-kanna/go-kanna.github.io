package mapper_test

import (
	"go/types"
	"slices"
	"strings"
	"testing"

	"github.com/go-kanna/kanna/internal/gen/mapper"
	"github.com/go-kanna/kanna/internal/packages"
)

func namedType(t *testing.T, pkg *packages.Package, name string) types.Type {
	t.Helper()
	obj := pkg.Types.Scope().Lookup(name)
	if obj == nil {
		t.Fatalf("type %s not found in %s", name, pkg.PkgPath)
	}
	return obj.Type()
}

func employeePairs(t *testing.T) ([]mapper.PairSpec, mapper.ConverterTable, *packages.Package) {
	t.Helper()
	model := fixture(t, "model")
	protolike := fixture(t, "protolike")
	conv := fixture(t, "conv")

	table, err := mapper.ExtractConverters([]*packages.Package{conv}, "example.com/output")
	if err != nil {
		t.Fatalf("extract converters: %v", err)
	}
	pairs := []mapper.PairSpec{
		{Src: namedType(t, model, "Employee"), Dst: types.NewPointer(namedType(t, protolike, "Employee"))},
		{Src: namedType(t, model, "Address"), Dst: types.NewPointer(namedType(t, protolike, "Address"))},
	}
	return pairs, table, model
}

// eventPairs adds the oneof-carrying pair to employeePairs; its arms resolve
// through the Employee and Address pairs.
func eventPairs(t *testing.T) ([]mapper.PairSpec, mapper.ConverterTable, *packages.Package) {
	t.Helper()
	pairs, table, model := employeePairs(t)
	protolike := fixture(t, "protolike")
	pairs = append(pairs, mapper.PairSpec{
		Src: namedType(t, model, "Event"), Dst: types.NewPointer(namedType(t, protolike, "Event")),
	})
	return pairs, table, model
}

// plansNamed keeps the plans whose function name has one of the prefixes.
func plansNamed(plans []*mapper.FuncPlan, prefixes ...string) []*mapper.FuncPlan {
	var out []*mapper.FuncPlan
	for _, p := range plans {
		for _, prefix := range prefixes {
			if strings.HasPrefix(mapper.PlanName(p), prefix) {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

func TestResolvePlans(t *testing.T) {
	t.Parallel()

	pairs, table, model := employeePairs(t)
	plans, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset:  model.Fset,
		Pairs: mapper.WithDirection(pairs, mapper.DirectionBoth),
		Conv:  table,
		Ignores: map[mapper.FieldKey]bool{
			{PkgPath: model.PkgPath, Type: "Employee", Field: "CreatedAt"}: true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		`EmployeeToProtolike(model.Employee) *protolike.Employee
  Id = .ID conv:FormatUserID
  Name = .EmployeeName direct
  Age = .Age direct
  HiredAt = .HiredAt conv:ToDate
  Address = .Address map:AddressToProtolike
  Tags = .Tags slice(cast:string)
  Subordinates = .Subordinates slice(map:EmployeeToProtolike)
  Note = .Note addr(direct)`,
		`EmployeeFromProtolike(*protolike.Employee) (model.Employee, error)
  ID = .GetId() convE:ParseUserID
  EmployeeName = .GetName() direct
  Age = .GetAge() direct
  HiredAt = .GetHiredAt() conv:ToTime
  Address = .GetAddress() map:AddressFromProtolike
  Tags = .GetTags() slice(cast:model.Tag)
  Subordinates = .GetSubordinates() slice(map:EmployeeFromProtolike)
  Note = .GetNote() direct`,
		`AddressToProtolike(model.Address) *protolike.Address
  City = .City direct
  Street = .Street direct`,
		`AddressFromProtolike(*protolike.Address) model.Address
  City = .GetCity() direct
  Street = .GetStreet() direct`,
	}
	if len(plans) != len(want) {
		t.Fatalf("got %d plans, want %d", len(plans), len(want))
	}
	for i, plan := range plans {
		if got := mapper.DescribePlan(plan); got != want[i] {
			t.Errorf("plan %d:\ngot:\n%s\nwant:\n%s", i, got, want[i])
		}
	}
}

// A destination oneof takes the first set source pointer, converted through
// what it points at; a source oneof hands each arm to the destination field of
// the arm's name, with the nil-safety any pointer gets.
func TestResolvePlansOneof(t *testing.T) {
	t.Parallel()

	pairs, table, model := eventPairs(t)
	plans, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset:  model.Fset,
		Pairs: mapper.WithDirection(pairs, mapper.DirectionBoth),
		Conv:  table,
		Ignores: map[mapper.FieldKey]bool{
			{PkgPath: model.PkgPath, Type: "Employee", Field: "CreatedAt"}: true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		`EventToProtolike(model.Event) *protolike.Event
  Version = .Version direct
  Kind = oneof
    Event_Hired.Hired = *.Hired map:EmployeeToProtolike
    Event_Moved.Moved = *.Moved map:AddressToProtolike
    Event_Note.Note = *.Note direct`,
		`EventFromProtolike(*protolike.Event) (model.Event, error)
  Version = .GetVersion() direct
  oneof .GetKind()
    Hired = Event_Hired.Hired guard(addr(map:EmployeeFromProtolike))
    Moved = Event_Moved.Moved guard(addr(map:AddressFromProtolike))
    Note = Event_Note.Note addr(direct)`,
	}
	got := plansNamed(plans, "Event")
	if len(got) != len(want) {
		t.Fatalf("got %d Event plans, want %d", len(got), len(want))
	}
	for i, plan := range got {
		if got := mapper.DescribePlan(plan); got != want[i] {
			t.Errorf("plan %d:\ngot:\n%s\nwant:\n%s", i, got, want[i])
		}
	}
}

// -exclude names an arm by its wrapper, as TYPE.FIELD always does, and the
// arm drops out of the chain.
func TestResolvePlansOneofExcludesArm(t *testing.T) {
	t.Parallel()

	pairs, table, model := eventPairs(t)
	protolike := fixture(t, "protolike")
	plans, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset:  model.Fset,
		Pairs: mapper.WithDirection(pairs, mapper.DirectionTo),
		Conv:  table,
		Ignores: map[mapper.FieldKey]bool{
			{PkgPath: protolike.PkgPath, Type: "Event_Note", Field: "Note"}: true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `EventToProtolike(model.Event) *protolike.Event
  Version = .Version direct
  Kind = oneof
    Event_Hired.Hired = *.Hired map:EmployeeToProtolike
    Event_Moved.Moved = *.Moved map:AddressToProtolike`
	got := plansNamed(plans, "Event")
	if len(got) != 1 {
		t.Fatalf("got %d Event plans, want 1", len(got))
	}
	if got := mapper.DescribePlan(got[0]); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestResolvePlansDirectionTo(t *testing.T) {
	t.Parallel()

	pairs, table, model := employeePairs(t)
	plans, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset:  model.Fset,
		Pairs: mapper.WithDirection(pairs, mapper.DirectionTo),
		Conv:  table,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("got %d plans, want 2", len(plans))
	}
	for i, name := range []string{"EmployeeToProtolike", "AddressToProtolike"} {
		if got := mapper.DescribePlan(plans[i]); !strings.HasPrefix(got, name+"(") {
			t.Errorf("plan %d = %q, want prefix %q", i, got, name)
		}
	}
}

func TestResolvePlansPromotedField(t *testing.T) {
	t.Parallel()

	model := fixture(t, "model")
	protolike := fixture(t, "protolike")
	pairs := []mapper.PairSpec{
		{Src: namedType(t, model, "WithBase"), Dst: namedType(t, protolike, "Flat")},
	}
	plans, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset:  model.Fset,
		Pairs: mapper.WithDirection(pairs, mapper.DirectionTo),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `WithBaseToFlat(model.WithBase) protolike.Flat
  Code = .Code direct
  Name = .Name direct`
	if got := mapper.DescribePlan(plans[0]); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestResolvePlansEmbeddedDstError(t *testing.T) {
	t.Parallel()

	model := fixture(t, "model")
	protolike := fixture(t, "protolike")
	pairs := []mapper.PairSpec{
		{Src: namedType(t, model, "WithBase"), Dst: namedType(t, protolike, "Flat")},
	}
	_, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset:  model.Fset,
		Pairs: mapper.WithDirection(pairs, mapper.DirectionFrom),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "embedded fields are not flattened") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// A getter returning a value must not be preferred over the pointer field it
// wraps, whether or not a converter sits in between: reading through it turns a
// nil pointer into a pointer to a zero, which is the distinction the pointer was
// carrying. A converter taking the pointer itself is guarded for the same
// reason.
func TestResolvePlansPrefersTheNilableRead(t *testing.T) {
	t.Parallel()

	opt := fixture(t, "optional")
	table, err := mapper.ExtractConverters([]*packages.Package{opt}, "example.com/output")
	if err != nil {
		t.Fatalf("extract converters: %v", err)
	}
	plans, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset: opt.Fset,
		Pairs: mapper.WithDirection([]mapper.PairSpec{
			{Src: types.NewPointer(namedType(t, opt, "Wire")), Dst: namedType(t, opt, "Domain")},
		}, mapper.DirectionTo),
		Conv: table,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `WireToDomain(*optional.Wire) (optional.Domain, error)
  Note = .Note direct
  Count = .Count deref(addr(convE:Atoi))
  Seen = .Seen guard(addr(conv:StampSeconds))
  Days = .Days conv:StampDays`
	if got := mapper.DescribePlan(plans[0]); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// A tag on a promoted field keeps applying. Matching by name regardless would
// copy a field the author excluded, with nothing to say it happened.
func TestResolvePlansPromotedFieldHonorsTags(t *testing.T) {
	t.Parallel()

	errcases := fixture(t, "errcases")
	_, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset: errcases.Fset,
		Pairs: mapper.WithDirection([]mapper.PairSpec{
			{Src: namedType(t, errcases, "PromotedSrc"), Dst: namedType(t, errcases, "PromotedDst")},
		}, mapper.DirectionTo),
	})
	if err == nil {
		t.Fatal("expected the excluded field to be reported as unmapped")
	}
	if !strings.Contains(err.Error(), "PromotedDst.Secret") {
		t.Errorf("error %q does not name the excluded field", err)
	}
}

func TestResolvePlansErrors(t *testing.T) {
	t.Parallel()

	errcases := fixture(t, "errcases")
	tests := []struct {
		name     string
		src, dst string
		wantErr  []string
	}{
		{
			name: "unmapped destination field",
			src:  "UnmappedSrc", dst: "UnmappedDst",
			wantErr: []string{"no source field", "UnmappedDst.Bar", `map:"-"`},
		},
		{
			name: "dst tag names missing source field",
			src:  "TagMissingSrc", dst: "TagMissingDst",
			wantErr: []string{`map tag "Nope" names a source field that does not exist`},
		},
		{
			name: "src tag names missing destination field",
			src:  "TagMissingDst", dst: "TagMissingSrc",
			wantErr: []string{`map tag "Nope" names a field that does not exist in errcases.TagMissingSrc`},
		},
		{
			name: "src tag names unexported destination field",
			src:  "TagUnexportedSrc", dst: "TagUnexportedDst",
			wantErr: []string{`map tag "hidden" names a field that does not exist in errcases.TagUnexportedDst`},
		},
		{
			name: "conflicting tags",
			src:  "ConflictSrc", dst: "ConflictDst",
			wantErr: []string{"conflicting map tags"},
		},
		{
			name: "ambiguous fold match",
			src:  "AmbiguousSrc", dst: "AmbiguousDst",
			wantErr: []string{"ambiguous"},
		},
		{
			name: "interface field without implementations",
			src:  "OneofSrc", dst: "OneofDst",
			wantErr: []string{"cannot map", "oneof", "this one has none"},
		},
		{
			name: "oneof arm without a source",
			src:  "ArmlessSrc", dst: "ArmlessDst",
			wantErr: []string{"no source field", "errcases.Armless_X.X (arm of errcases.ArmlessDst.Kind)"},
		},
		{
			name: "oneof arm from a value",
			src:  "ValueArmSrc", dst: "ValueArmDst",
			wantErr: []string{"ValueArmSrc.X (string)", "set only when its source is"},
		},
		{
			name: "oneof arm from another oneof",
			src:  "BothSrc", dst: "BothDst",
			wantErr: []string{"errcases.BothSrc_X.X", "not filled from another oneof"},
		},
		{
			name: "opaque destination",
			src:  "OpaqueSrc", dst: "OpaqueDst",
			wantErr: []string{"opaque API is not supported"},
		},
		{
			name: "invalid tag value",
			src:  "BadTagSrc", dst: "BadTagDst",
			wantErr: []string{`invalid map tag "9x"`},
		},
		{
			name: "lossy conversion needs converter",
			src:  "LossySrc", dst: "LossyDst",
			wantErr: []string{"cannot map", "mapper.Register(func(float64) int { ... })"},
		},
		{
			name: "narrowing integer needs converter",
			src:  "NarrowSrc", dst: "NarrowDst",
			wantErr: []string{"cannot map", "mapper.Register(func(int64) int32 { ... })"},
		},
		{
			name: "sign change needs converter",
			src:  "SignSrc", dst: "SignDst",
			wantErr: []string{"cannot map", "mapper.Register(func(int) uint { ... })"},
		},
		{
			name: "converter suggestion in message",
			src:  "NeedConvSrc", dst: "NeedConvDst",
			wantErr: []string{"mapper.Register(func(time.Time) string { ... })"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pairs := []mapper.PairSpec{
				{Src: namedType(t, errcases, tt.src), Dst: namedType(t, errcases, tt.dst)},
			}
			_, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
				Fset:  errcases.Fset,
				Pairs: mapper.WithDirection(pairs, mapper.DirectionTo),
			})
			if err == nil {
				t.Fatal("expected error")
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
		})
	}
}

func TestResolvePlansReportsBothInvalidPairTypes(t *testing.T) {
	t.Parallel()

	model := fixture(t, "model")
	pairs := []mapper.PairSpec{
		{Src: namedType(t, model, "UserID"), Dst: namedType(t, model, "Tag")},
	}
	_, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset:  model.Fset,
		Pairs: mapper.WithDirection(pairs, mapper.DirectionBoth),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"model.UserID", "model.Tag"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not report %s", err, want)
		}
	}
}

func TestTypeConvertible(t *testing.T) {
	t.Parallel()

	byteSlice := types.NewSlice(types.Typ[types.Byte])
	tests := []struct {
		name string
		src  types.Type
		dst  types.Type
		want bool
	}{
		{"widen signed", types.Typ[types.Int32], types.Typ[types.Int64], true},
		{"narrow signed", types.Typ[types.Int64], types.Typ[types.Int32], false},
		{"int widens to int64", types.Typ[types.Int], types.Typ[types.Int64], true},
		{"int64 may not fit int", types.Typ[types.Int64], types.Typ[types.Int], false},
		{"int32 fits int", types.Typ[types.Int32], types.Typ[types.Int], true},
		{"int may not fit int32", types.Typ[types.Int], types.Typ[types.Int32], false},
		{"signed to unsigned", types.Typ[types.Int8], types.Typ[types.Uint8], false},
		{"unsigned widens into signed", types.Typ[types.Uint8], types.Typ[types.Int16], true},
		{"unsigned needs extra bit", types.Typ[types.Uint32], types.Typ[types.Int32], false},
		{"uint32 fits int64", types.Typ[types.Uint32], types.Typ[types.Int64], true},
		{"uint may not fit int64", types.Typ[types.Uint], types.Typ[types.Int64], false},
		{"widen unsigned", types.Typ[types.Uint], types.Typ[types.Uint64], true},
		{"uintptr always needs converter", types.Typ[types.Uintptr], types.Typ[types.Uint64], false},
		{"float widens", types.Typ[types.Float32], types.Typ[types.Float64], true},
		{"float narrows", types.Typ[types.Float64], types.Typ[types.Float32], false},
		{"int32 exact in float64", types.Typ[types.Int32], types.Typ[types.Float64], true},
		{"int64 loses precision in float64", types.Typ[types.Int64], types.Typ[types.Float64], false},
		{"int32 loses precision in float32", types.Typ[types.Int32], types.Typ[types.Float32], false},
		{"int16 exact in float32", types.Typ[types.Int16], types.Typ[types.Float32], true},
		{"float to integer", types.Typ[types.Float64], types.Typ[types.Int64], false},
		{"numeric to string", types.Typ[types.Int], types.Typ[types.String], false},
		{"string to byte slice", types.Typ[types.String], byteSlice, true},
		{"byte slice to string", byteSlice, types.Typ[types.String], true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := mapper.TypeConvertible(tt.src, tt.dst); got != tt.want {
				t.Errorf("typeConvertible(%s, %s) = %v, want %v", tt.src, tt.dst, got, tt.want)
			}
		})
	}
}

func TestResolvePlansUnusedIgnore(t *testing.T) {
	t.Parallel()

	errcases := fixture(t, "errcases")
	pairs := []mapper.PairSpec{
		{Src: namedType(t, errcases, "UnmappedSrc"), Dst: namedType(t, errcases, "UnmappedDst")},
	}
	_, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset:  errcases.Fset,
		Pairs: mapper.WithDirection(pairs, mapper.DirectionTo),
		Ignores: map[mapper.FieldKey]bool{
			{PkgPath: errcases.PkgPath, Type: "UnmappedDst", Field: "Bar"}:  true,
			{PkgPath: errcases.PkgPath, Type: "UnmappedDst", Field: "Nope"}: true,
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "matched nothing") {
		t.Errorf("unexpected error message: %v", err)
	}
	if strings.Contains(err.Error(), "no source field") {
		t.Errorf("ignored field Bar should not be reported: %v", err)
	}
}

// A value and a pointer to it are different pairs that end up with one name.
func TestResolvePlansDuplicateName(t *testing.T) {
	t.Parallel()

	errcases := fixture(t, "errcases")
	src := namedType(t, errcases, "UnmappedSrc")
	_, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset: errcases.Fset,
		Pairs: mapper.WithDirection([]mapper.PairSpec{
			{Src: src, Dst: src},
			{Src: src, Dst: types.NewPointer(src)},
		}, mapper.DirectionTo),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "duplicate function name") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// Each pair keeps the direction it was declared with, so one run can map one
// type in and another out.
func TestResolvePlansPerPairDirection(t *testing.T) {
	t.Parallel()

	pairs, table, model := employeePairs(t)
	pairs[0].Direction = mapper.DirectionTo
	pairs[1].Direction = mapper.DirectionBoth
	plans, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset:  model.Fset,
		Pairs: pairs,
		Conv:  table,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := planNames(plans)
	want := []string{"EmployeeToProtolike", "AddressToProtolike", "AddressFromProtolike"}
	if !slices.Equal(names, want) {
		t.Errorf("plans = %v, want %v", names, want)
	}
}

// Declaring a pair once in each direction asks for both, not for a duplicate.
func TestResolvePlansMergesDirections(t *testing.T) {
	t.Parallel()

	pairs, table, model := employeePairs(t)
	to, from := pairs[1], pairs[1]
	to.Direction = mapper.DirectionTo
	from.Direction = mapper.DirectionFrom
	plans, _, err := mapper.ResolvePlans(mapper.ResolveConfig{
		Fset:  model.Fset,
		Pairs: []mapper.PairSpec{to, from},
		Conv:  table,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	names := planNames(plans)
	want := []string{"AddressToProtolike", "AddressFromProtolike"}
	if !slices.Equal(names, want) {
		t.Errorf("plans = %v, want %v", names, want)
	}
}

func planNames(plans []*mapper.FuncPlan) []string {
	names := make([]string, 0, len(plans))
	for _, p := range plans {
		names = append(names, mapper.PlanName(p))
	}
	return names
}
