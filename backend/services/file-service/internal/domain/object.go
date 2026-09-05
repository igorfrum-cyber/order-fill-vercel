package domain

type Object struct {
	ID          string
	Key         string
	Name        string
	ContentType string
	Size        int64
	Body        []byte
}
