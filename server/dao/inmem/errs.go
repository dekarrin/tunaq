package inmem

type Error string

func (e Error) Error() string {
	return string(e)
}

var (
	ErrConstraintViolation Error = "A uniqueness constraint was violated"
	ErrNotFound            Error = "The requested resource was not found"
)
