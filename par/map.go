package par

import (
	"reflect"
	"runtime"
)

var defaultWorkers = runtime.NumCPU()

type MapFunc = func(key, value interface{}, emit func(x interface{}))

func Map(v interface{}, mapper MapFunc) <-chan interface{} {
	return MapN(v, defaultWorkers, mapper)
}

func MapN(v interface{}, workers int, mapper MapFunc) <-chan interface{} {
	type task struct {
		key   interface{}
		value interface{}
	}
	r := reflect.ValueOf(v)
	kind := r.Kind()
	if !(kind == reflect.Map || kind == reflect.Slice || kind == reflect.Array || kind == reflect.Chan) {
		panic("not a map, slice, array, or channel")
	}
	tasks := make(chan task)
	results := make(chan interface{})
	emit := func(x interface{}) {
		results <- x
	}
	done := make(chan struct{})
	// spawn workers
	for i := 0; i < workers; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for t := range tasks {
				mapper(t.key, t.value, emit)
			}
		}()
	}
	// feed workers
	go func() {
		switch kind {
		case reflect.Map:
			iter := r.MapRange()
			for iter.Next() {
				k := iter.Key()
				v := iter.Value()
				tasks <- task{k.Interface(), v.Interface()}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < r.Len(); i++ {
				tasks <- task{i, r.Index(i).Interface()}
			}
		case reflect.Chan:
			for i := 0; ; i++ {
				x, ok := r.Recv()
				if !ok {
					break
				}
				tasks <- task{i, x.Interface()}
			}
		}
		close(tasks)
	}()
	// close results when done
	go func() {
		ndone := 0
		for range done {
			ndone++
			if ndone >= workers {
				close(results)
				return
			}
		}
	}()
	return results
}
