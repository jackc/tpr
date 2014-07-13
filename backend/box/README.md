box
===

Use a box around values that can be undefined, unknown, empty, or present.

As Go does not have generics or templates, a box.go.erb is processed to produce box.go with specialized box types for each underlying type. Currently, boxes are defined for bool, float32, float64, int8, int16, int32, int64, string, time.Time, uint8, uint16, uint32, and uint64 types.

If you need additional types, you may want to fork this project into a sub-package and customize the type list in box.go.erb.
