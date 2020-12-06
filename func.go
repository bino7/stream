package stream

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

/*can't tell stream.HandleFunc is nil correctly*/
func FistNotNil(vs ...interface{}) interface{} {
	for _, v := range vs {
		if v != nil {
			return v
		}
	}
	return nil
}

func FuncName(f interface{}) (string, string, error) {
	m := reflect.ValueOf(f)
	if m.Kind() != reflect.Func {
		return "", "", fmt.Errorf("input is not a func")
	}
	nameFull := runtime.FuncForPC(m.Pointer()).Name()
	nameEnd := filepath.Ext(nameFull)
	pkg := strings.TrimSuffix(nameFull, nameEnd)
	name := strings.TrimSuffix(strings.TrimPrefix(nameEnd, "."), "-fm")
	return pkg, name, nil
}
