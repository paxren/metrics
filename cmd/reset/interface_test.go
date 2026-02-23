package main

import "testing"

// TestInterfaceImplementation проверяет, что *ExampleStruct реализует интерфейс Resetter
func TestInterfaceImplementation(t *testing.T) {
	var _ Resetter = (*ExampleStruct)(nil)

	obj := &ExampleStruct{}
	obj.Reset()

	if obj.ID != 0 || obj.Name != "" || obj.Active != false || len(obj.Data) != 0 {
		t.Errorf("Reset() didn't work properly: %+v", obj)
	}
}
