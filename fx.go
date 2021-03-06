package stream

type (

	// EmittableFunc defines a function that should be used with Start operator.
	// EmittableFunc can be used to wrap a blocking operation.
	EmittableFunc func() interface{}

	// MappableFunc defines a function that acts as a predicate to the Map operator.
	MappableFunc func(interface{}) interface{}

	// ScannableFunc defines a function that acts as a predicate to the Scan operator.
	ScannableFunc func(interface{}, interface{}) interface{}

	// FilterableFunc defines a func that should be passed to the Filter operator.
	FilterableFunc func(interface{}) bool

	// KeySelectorFunc defines a func that should be passed to the Distinct operator.
	KeySelectorFunc func(interface{}) interface{}

	// HandleFunc defines a func that should be passed to the Distinct operator.
	HandleFunc func(interface{}) (interface{}, error)

	ConsumeFunc func(interface{})

	ClassifyFunc func(interface{}) string

	CompareFunc func(interface{}, interface{}) int

	GroupHandleFunc map[interface{}]HandleFunc

	ErrorHandleFunc func(err error) error

	Max func(receiver interface{}, apply CompareFunc)

	Min func(receiver interface{}, apply CompareFunc)
)
