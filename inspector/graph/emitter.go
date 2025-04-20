package graph

// Emitter represents generator
type Emitter interface {
	Emit(file *File) ([]byte, error)
}
