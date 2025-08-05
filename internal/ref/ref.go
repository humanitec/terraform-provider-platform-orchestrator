package ref

// DerefOr returns the dereferenced value of the pointer v if it is not nil, otherwise returns the default value d.
func DerefOr[T any](v *T, d T) T {
	if v == nil {
		return d
	}
	return *v
}

// RefStringEmptyNil returns a pointer to the string if it is not empty, otherwise returns nil.
func RefStringEmptyNil(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// Ref returns a pointer to the value of type a.
func Ref[a any](i a) *a {
	return &i
}

// ReplaceStringOrNil returns nil if the input string is nil, otherwise returns a pointer to the replacement string.
func ReplaceStringOrNil(s *string, r string) *string {
	if s == nil {
		return nil
	} else {
		return &r
	}
}
