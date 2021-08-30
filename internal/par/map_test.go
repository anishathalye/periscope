package par

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func TestMapMapBasic(t *testing.T) {
	m := map[int]string{
		1: "hello",
		3: "map",
		5: "skip",
	}
	c := Map(m, func(k, v interface{}, emit func(x interface{})) {
		if v.(string) == "skip" {
			return
		}
		emit(fmt.Sprintf("%d-%d", k.(int), len(v.(string))))
	})
	var got []string
	for v := range c {
		got = append(got, v.(string))
	}
	sort.Strings(got)
	expected := []string{"1-5", "3-3"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestMapMapMultipleEmit(t *testing.T) {
	m := map[string]string{
		"lorem":      "ipsum",
		"dolor":      "sit",
		"amet":       "consectetur",
		"adipiscing": "elit",
	}
	mapper := func(k, v interface{}, emit func(x interface{})) {
		emit(len(k.(string)))
		emit(len(v.(string)))
	}
	var got []int
	for v := range MapN(m, 4, mapper) {
		got = append(got, v.(int))
	}
	sort.Ints(got)
	expected := []int{3, 4, 4, 5, 5, 5, 10, 11}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}

func TestSliceMapBasic(t *testing.T) {
	s := []int{3, 19, 256, 10}
	mapper := func(i, v interface{}, emit func(x interface{})) {
		emit(fmt.Sprintf("%d", i.(int)))
		emit(fmt.Sprintf("0x%x", v.(int)))
	}
	var got []string
	for v := range Map(s, mapper) {
		got = append(got, v.(string))
	}
	sort.Strings(got)
	expected := []string{"0", "0x100", "0x13", "0x3", "0xa", "1", "2", "3"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}

}

func TestMapChanBasic(t *testing.T) {
	c := make(chan string)
	go func() {
		c <- "the"
		c <- "quick"
		c <- "brown"
		c <- "fox"
		close(c)
	}()
	mapper := func(i, v interface{}, emit func(x interface{})) {
		emit(i.(int) + len(v.(string)))
	}
	var got []int
	for v := range Map(c, mapper) {
		got = append(got, v.(int))
	}
	sort.Ints(got)
	expected := []int{3, 6, 6, 7}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
}
