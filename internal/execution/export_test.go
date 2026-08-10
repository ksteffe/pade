package execution

// NewSecretRedactorForTest exposes the streaming redactor for unit tests.
func NewSecretRedactorForTest(dst interface {
	Write([]byte) (int, error)
}, secrets []string) interface {
	Write([]byte) (int, error)
	Close() error
} {
	return newSecretRedactor(dst, secrets)
}
