// Package render provides default implementations of the
// mig.Renderer interface for Go's standard template engines.
package render

// FuncMap is a convenience type for template functions. By using this,
// users of the render package don't need to import the underlying
// html/template or text/template packages directly.
type FuncMap map[string]any

// Must panics if err is not nil. It is intended for use in variable
// initialization during startup, such as `var renderer = render.Must(render.NewHTML(...))`.
func Must[T any](val T, err error) T {
	if err != nil {
		panic(err)
	}
	return val
}
