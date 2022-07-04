package model

type Index = map[string]Pair
type Pair struct {
	Start uint32
	End   uint32
}
