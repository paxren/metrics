package resetexamples

// generate:reset
type SimpleStruct struct {
	IntField    int
	StringField string
	BoolField   bool
	FloatField  float64
}

// generate:reset
type ComplexStruct struct {
	IntSlice    []int
	StringSlice []string
	IntMap      map[string]int
	StringMap   map[string]string
	IntPtr      *int
	StringPtr   *string
	BoolPtr     *bool
	FloatPtr    *float64
}

// generate:reset
type NestedStruct struct {
	Simple  SimpleStruct
	Complex *ComplexStruct
	Mixed   interface{ Reset() }
}

// generate:reset
type AllTypesStruct struct {
	// Примитивные типы
	IntVal     int
	Int8Val    int8
	Int16Val   int16
	Int32Val   int32
	Int64Val   int64
	UintVal    uint
	Uint8Val   uint8
	Uint16Val  uint16
	Uint32Val  uint32
	Uint64Val  uint64
	Float32Val float32
	Float64Val float64
	StringVal  string
	BoolVal    bool

	// Сложные типы
	IntSliceVal    []int
	StringSliceVal []string
	IntMapVal      map[string]int
	StringMapVal   map[string]string

	// Указатели на примитивы
	IntPtrVal    *int
	StringPtrVal *string
	BoolPtrVal   *bool
	FloatPtrVal  *float64

	// Вложенные структуры
	SimpleStructVal SimpleStruct
	ComplexPtrVal   *ComplexStruct
}
