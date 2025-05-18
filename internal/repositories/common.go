package repositories

type BatchResult[T any] struct {
	Value T
	Err   error
}

type BatchCreateArg struct {
	ShortIdentifier string
	URL             string
	VisitorUUID     *string
}
