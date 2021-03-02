package par

import (
	"reflect"
	"runtime"
	"sync"
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
	var wg sync.WaitGroup
	// spawn workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
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
		// wait for workers, close results when done
		wg.Wait()
		close(results)
	}()
	return results
}
