// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

package marshal

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/attic-labs/noms/go/nomdl"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/testify/assert"
)

func TestMarshalTypeType(tt *testing.T) {
	vs := newTestValueStore()
	defer vs.Close()

	t := func(exp *types.Type, ptr interface{}) {
		p := reflect.ValueOf(ptr)
		assert.NotEqual(tt, reflect.Ptr, p.Type().Kind())
		actual, err := MarshalType(vs, p.Interface())
		assert.NoError(tt, err)
		assert.NotNil(tt, actual, "%#v", p.Interface())
		assert.True(tt, exp.Equals(actual))
	}

	t(types.NumberType, float32(0))
	t(types.NumberType, float64(0))
	t(types.NumberType, int(0))
	t(types.NumberType, int16(0))
	t(types.NumberType, int32(0))
	t(types.NumberType, int64(0))
	t(types.NumberType, int8(0))
	t(types.NumberType, uint(0))
	t(types.NumberType, uint16(0))
	t(types.NumberType, uint32(0))
	t(types.NumberType, uint64(0))
	t(types.NumberType, uint8(0))

	t(types.BoolType, true)
	t(types.StringType, "hi")

	var l []int
	t(types.MakeListType(types.NumberType), l)

	var m map[uint32]string
	t(types.MakeMapType(types.NumberType, types.StringType), m)

	t(types.MakeListType(types.ValueType), types.List{})
	t(types.MakeSetType(types.ValueType), types.Set{})
	t(types.MakeMapType(types.ValueType, types.ValueType), types.Map{})
	t(types.MakeRefType(types.ValueType), types.Ref{})

	type TestStruct struct {
		Str string
		Num float64
	}
	var str TestStruct
	t(types.MakeStructTypeFromFields("TestStruct", types.FieldMap{
		"str": types.StringType,
		"num": types.NumberType,
	}), str)

	// Same again to test caching
	t(types.MakeStructTypeFromFields("TestStruct", types.FieldMap{
		"str": types.StringType,
		"num": types.NumberType,
	}), str)

	anonStruct := struct {
		B bool
	}{
		true,
	}
	t(types.MakeStructTypeFromFields("", types.FieldMap{
		"b": types.BoolType,
	}), anonStruct)

	type TestNestedStruct struct {
		A []int16
		B TestStruct
		C float64
	}
	var nestedStruct TestNestedStruct
	t(types.MakeStructTypeFromFields("TestNestedStruct", types.FieldMap{
		"a": types.MakeListType(types.NumberType),
		"b": types.MakeStructTypeFromFields("TestStruct", types.FieldMap{
			"str": types.StringType,
			"num": types.NumberType,
		}),
		"c": types.NumberType,
	}), nestedStruct)

	type testStruct struct {
		Str string
		Num float64
	}
	var ts testStruct
	t(types.MakeStructTypeFromFields("TestStruct", types.FieldMap{
		"str": types.StringType,
		"num": types.NumberType,
	}), ts)
}

//
func assertMarshalTypeErrorMessage(t *testing.T, v interface{}, expectedMessage string) {
	vs := newTestValueStore()
	defer vs.Close()

	_, err := MarshalType(vs, v)
	assert.Error(t, err)
	assert.Equal(t, expectedMessage, err.Error())
}

func TestMarshalTypeInvalidTypes(t *testing.T) {
	assertMarshalTypeErrorMessage(t, make(chan int), "Type is not supported, type: chan int")
}

func TestMarshalTypeEmbeddedStruct(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type EmbeddedStruct struct {
		B bool
	}
	type TestStruct struct {
		EmbeddedStruct
		A int
	}

	var s TestStruct
	typ := MustMarshalType(vs, s)

	assert.True(types.MakeStructTypeFromFields("TestStruct", types.FieldMap{
		"a": types.NumberType,
		"b": types.BoolType,
	}).Equals(typ))
}

func TestMarshalTypeEmbeddedStructSkip(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type EmbeddedStruct struct {
		B bool
	}
	type TestStruct struct {
		EmbeddedStruct `noms:"-"`
		A              int
	}

	var s TestStruct
	typ := MustMarshalType(vs, s)

	assert.True(types.MakeStructTypeFromFields("TestStruct", types.FieldMap{
		"a": types.NumberType,
	}).Equals(typ))
}

func TestMarshalTypeEmbeddedStructNamed(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type EmbeddedStruct struct {
		B bool
	}
	type TestStruct struct {
		EmbeddedStruct `noms:"em"`
		A              int
	}

	var s TestStruct
	typ := MustMarshalType(vs, s)

	assert.True(types.MakeStructTypeFromFields("TestStruct", types.FieldMap{
		"a": types.NumberType,
		"em": types.MakeStructTypeFromFields("EmbeddedStruct", types.FieldMap{
			"b": types.BoolType,
		}),
	}).Equals(typ))
}

func TestMarshalTypeEncodeNonExportedField(t *testing.T) {
	type TestStruct struct {
		x int
	}
	assertMarshalTypeErrorMessage(t, TestStruct{1}, "Non exported fields are not supported, type: marshal.TestStruct")
}

func TestMarshalTypeEncodeTaggingSkip(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type S struct {
		Abc int `noms:"-"`
		Def bool
	}
	var s S
	typ, err := MarshalType(vs, s)
	assert.NoError(err)
	assert.True(types.MakeStructTypeFromFields("S", types.FieldMap{
		"def": types.BoolType,
	}).Equals(typ))
}

func TestMarshalTypeNamedFields(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type S struct {
		Aaa int  `noms:"a"`
		Bbb bool `noms:"B"`
		Ccc string
	}
	var s S
	typ, err := MarshalType(vs, s)
	assert.NoError(err)
	assert.True(types.MakeStructTypeFromFields("S", types.FieldMap{
		"a":   types.NumberType,
		"B":   types.BoolType,
		"ccc": types.StringType,
	}).Equals(typ))
}

func TestMarshalTypeInvalidNamedFields(t *testing.T) {
	type S struct {
		A int `noms:"1a"`
	}
	var s S
	assertMarshalTypeErrorMessage(t, s, "Invalid struct field name: 1a")
}

func TestMarshalTypeOmitEmpty(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type S struct {
		String string `noms:",omitempty"`
	}
	var s S
	typ, err := MarshalType(vs, s)
	assert.NoError(err)
	assert.True(types.MakeStructType("S", types.StructField{"string", types.StringType, true}).Equals(typ))
}

func ExampleMarshalType() {
	vs := newTestValueStore()
	defer vs.Close()

	type Person struct {
		Given  string
		Female bool
	}
	var person Person
	personNomsType, err := MarshalType(vs, person)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(personNomsType.Describe())
	// Output: struct Person {
	//   female: Bool,
	//   given: String,
	// }
}

func TestMarshalTypeSlice(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	s := []string{"a", "b", "c"}
	typ, err := MarshalType(vs, s)
	assert.NoError(err)
	assert.True(types.MakeListType(types.StringType).Equals(typ))
}

func TestMarshalTypeArray(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	a := [3]int{1, 2, 3}
	typ, err := MarshalType(vs, a)
	assert.NoError(err)
	assert.True(types.MakeListType(types.NumberType).Equals(typ))
}

func TestMarshalTypeStructWithSlice(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type S struct {
		List []int
	}
	var s S
	typ, err := MarshalType(vs, s)
	assert.NoError(err)
	assert.True(types.MakeStructTypeFromFields("S", types.FieldMap{
		"list": types.MakeListType(types.NumberType),
	}).Equals(typ))
}

func TestMarshalTypeRecursive(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type Node struct {
		Value    int
		Children []Node
	}
	var n Node
	typ, err := MarshalType(vs, n)
	assert.NoError(err)

	typ2 := types.MakeStructType("Node",
		types.StructField{
			Name: "children",
			Type: types.MakeListType(types.MakeCycleType("Node")),
		},
		types.StructField{
			Name: "value",
			Type: types.NumberType,
		},
	)
	assert.True(typ2.Equals(typ))
}

func TestMarshalTypeMap(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	var m map[string]int
	typ, err := MarshalType(vs, m)
	assert.NoError(err)
	assert.True(types.MakeMapType(types.StringType, types.NumberType).Equals(typ))

	type S struct {
		N string
	}

	var m2 map[S]bool
	typ, err = MarshalType(vs, m2)
	assert.NoError(err)
	assert.True(types.MakeMapType(
		types.MakeStructTypeFromFields("S", types.FieldMap{
			"n": types.StringType,
		}),
		types.BoolType).Equals(typ))
}

func TestMarshalTypeSet(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type S struct {
		A map[int]struct{} `noms:",set"`
		B map[int]struct{}
		C map[int]string      `noms:",set"`
		D map[string]struct{} `noms:",set"`
		E map[string]struct{}
		F map[string]int `noms:",set"`
		G []int          `noms:",set"`
		H string         `noms:",set"`
	}
	var s S
	typ, err := MarshalType(vs, s)
	assert.NoError(err)

	emptyStructType := types.MakeStructTypeFromFields("", types.FieldMap{})

	assert.True(types.MakeStructTypeFromFields("S", types.FieldMap{
		"a": types.MakeSetType(types.NumberType),
		"b": types.MakeMapType(types.NumberType, emptyStructType),
		"c": types.MakeMapType(types.NumberType, types.StringType),
		"d": types.MakeSetType(types.StringType),
		"e": types.MakeMapType(types.StringType, emptyStructType),
		"f": types.MakeMapType(types.StringType, types.NumberType),
		"g": types.MakeSetType(types.NumberType),
		"h": types.StringType,
	}).Equals(typ))
}

func TestEncodeTypeOpt(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	tc := []struct {
		in       interface{}
		opt      Opt
		wantType *types.Type
	}{
		{
			[]string{},
			Opt{},
			types.MakeListType(types.StringType),
		},
		{
			[]string{},
			Opt{Set: true},
			types.MakeSetType(types.StringType),
		},
		{
			map[string]struct{}{},
			Opt{},
			types.MakeMapType(types.StringType, types.MakeStructType("")),
		},
		{
			map[string]struct{}{},
			Opt{Set: true},
			types.MakeSetType(types.StringType),
		},
	}

	for _, t := range tc {
		r, err := MarshalTypeOpt(vs, t.in, t.opt)
		assert.True(t.wantType.Equals(r))
		assert.Nil(err)
	}
}

func TestMarshalTypeSetWithTags(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type S struct {
		A map[int]struct{} `noms:"foo,set"`
		B map[int]struct{} `noms:",omitempty,set"`
		C map[int]struct{} `noms:"bar,omitempty,set"`
	}

	var s S
	typ, err := MarshalType(vs, s)
	assert.NoError(err)
	assert.True(types.MakeStructType("S",
		types.StructField{"foo", types.MakeSetType(types.NumberType), false},
		types.StructField{"b", types.MakeSetType(types.NumberType), true},
		types.StructField{"bar", types.MakeSetType(types.NumberType), true},
	).Equals(typ))
}

func TestMarshalTypeInvalidTag(t *testing.T) {
	vs := newTestValueStore()
	defer vs.Close()

	type S struct {
		F string `noms:",omitEmpty"`
	}
	var s S
	_, err := MarshalType(vs, s)
	assert.Error(t, err)
	assert.Equal(t, `Unrecognized tag: omitEmpty`, err.Error())
}

func TestMarshalTypeCanSkipUnexportedField(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type S struct {
		Abc         int
		notExported bool `noms:"-"`
	}
	var s S
	typ, err := MarshalType(vs, s)
	assert.NoError(err)
	assert.True(types.MakeStructTypeFromFields("S", types.FieldMap{
		"abc": types.NumberType,
	}).Equals(typ))
}

func TestMarshalTypeOriginal(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type S struct {
		Foo int          `noms:",omitempty"`
		Bar types.Struct `noms:",original"`
	}

	var s S
	typ, err := MarshalType(vs, s)
	assert.NoError(err)
	assert.True(types.MakeStructType("S",
		types.StructField{"foo", types.NumberType, true},
	).Equals(typ))
}

func TestMarshalTypeNomsTypes(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	type S struct {
		Blob   types.Blob
		Bool   types.Bool
		Number types.Number
		String types.String
		Type   *types.Type
	}
	var s S
	assert.True(MustMarshalType(vs, s).Equals(
		types.MakeStructTypeFromFields("S", types.FieldMap{
			"blob":   types.BlobType,
			"bool":   types.BoolType,
			"number": types.NumberType,
			"string": types.StringType,
			"type":   types.TypeType,
		}),
	))
}

func (t primitiveType) MarshalNomsType() (*types.Type, error) {
	return types.NumberType, nil
}

func TestTypeMarshalerPrimitiveType(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	var u primitiveType
	typ := MustMarshalType(vs, u)
	assert.Equal(types.NumberType, typ)
}

func (u primitiveSliceType) MarshalNomsType() (*types.Type, error) {
	return types.StringType, nil
}

func TestTypeMarshalerPrimitiveSliceType(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	var u primitiveSliceType
	typ := MustMarshalType(vs, u)
	assert.Equal(types.StringType, typ)
}

func (u primitiveMapType) MarshalNomsType() (*types.Type, error) {
	return types.MakeSetType(types.StringType), nil
}

func TestTypeMarshalerPrimitiveMapType(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	var u primitiveMapType
	typ := MustMarshalType(vs, u)
	assert.Equal(types.MakeSetType(types.StringType), typ)
}

func TestTypeMarshalerPrimitiveStructTypeNoMarshalNomsType(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	var u primitiveStructType
	_, err := MarshalType(vs, u)
	assert.Error(err)
	assert.Equal("Cannot marshal type which implements marshal.Marshaler, perhaps implement marshal.TypeMarshaler for marshal.primitiveStructType", err.Error())
}

func (u builtinType) MarshalNomsType() (*types.Type, error) {
	return types.StringType, nil
}

func TestTypeMarshalerBuiltinType(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	var u builtinType
	typ := MustMarshalType(vs, u)
	assert.Equal(types.StringType, typ)
}

func (u wrappedMarshalerType) MarshalNomsType() (*types.Type, error) {
	return types.NumberType, nil
}

func TestTypeMarshalerWrapperMarshalerType(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	var u wrappedMarshalerType
	typ := MustMarshalType(vs, u)
	assert.Equal(types.NumberType, typ)
}

func (u returnsMarshalerError) MarshalNomsType() (*types.Type, error) {
	return nil, errors.New("expected error")
}

func (u returnsMarshalerNil) MarshalNomsType() (*types.Type, error) {
	return nil, nil
}

func (u panicsMarshaler) MarshalNomsType() (*types.Type, error) {
	panic("panic")
}

func TestTypeMarshalerErrors(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	expErr := errors.New("expected error")
	var m1 returnsMarshalerError
	_, actErr := MarshalType(vs, m1)
	assert.Equal(expErr, actErr)

	var m2 returnsMarshalerNil
	assert.Panics(func() { MarshalType(vs, m2) })

	var m3 panicsMarshaler
	assert.Panics(func() { MarshalType(vs, m3) })
}

func TestMarshalTypeStructName(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	var ts TestStructWithNameImpl
	typ := MustMarshalType(vs, ts)
	assert.True(types.MakeStructType("A", types.StructField{"x", types.NumberType, false}).Equals(typ), typ.Describe())
}

func TestMarshalTypeStructName2(t *testing.T) {
	assert := assert.New(t)

	vs := newTestValueStore()
	defer vs.Close()

	var ts TestStructWithNameImpl2
	typ := MustMarshalType(vs, ts)
	assert.True(types.MakeStructType("", types.StructField{"x", types.NumberType, false}).Equals(typ), typ.Describe())
}

type OutPhoto struct {
	Faces             []OutFace `noms:",set"`
	SomeOtherFacesSet []OutFace `noms:",set"`
}

type OutFace struct {
	Blob types.Ref
}

func (f OutFace) MarshalNomsStructName() string {
	return "Face"
}

func TestMarshalTypeOutface(t *testing.T) {
	vs := newTestValueStore()
	defer vs.Close()

	typ := MustMarshalType(vs, OutPhoto{})
	expectedType := nomdl.MustParseType(`struct OutPhoto {
          faces: Set<struct Face {
            blob: Ref<Value>,
          }>,
          someOtherFacesSet: Set<Cycle<Face>>,
        }`)
	assert.True(t, typ.Equals(expectedType))
}
