package main

import (
	"math/rand"
)

func ChooseWeighted[Key comparable](r *rand.Rand, weights map[Key]int) Key {
	var sum int
	for _, v := range weights {
		sum += v
	}
	i := r.Intn(sum)
	var s int
	for k, w := range weights {
		s += w
		if s > i {
			return k
		}
	}
	panic("shouldn't get here")
}
