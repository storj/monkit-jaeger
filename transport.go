package jaeger

type Transport interface {
	Write([]byte) error
	Close()
}
