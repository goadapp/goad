package helpers

type Int64slice []int64

func (a Int64slice) Len() int           { return len(a) }
func (a Int64slice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Int64slice) Less(i, j int) bool { return a[i] < a[j] }
