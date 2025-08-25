package mig

import "net/http"

// Response wraps http.ResponseWriter and tracks status and bytes written.
// It implements http.ResponseWriter and can be used anywhere a ResponseWriter is expected.
type Response struct {
	http.ResponseWriter
	status  int
	written int
}

func (w *Response) WriteHeader(code int) {
	// Only set once to mirror net/http behavior
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *Response) Write(b []byte) (int, error) {
	// Implicit 200 if Write called before WriteHeader
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.written += n
	return n, err
}

func (w *Response) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *Response) Written() int {
	return w.written
}
