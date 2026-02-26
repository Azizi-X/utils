package utils

import "sync"

func BasicEqual[T comparable](a, b T) bool { return a == b }

func HandleGroups(fns []func()) {
	wg := new(sync.WaitGroup)

	for _, fn := range fns {
		wg.Go(func() {
			fn()
		})
	}

	wg.Wait()
}
