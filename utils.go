<<<<<<< HEAD
package utils

import "sync"

func BasicEqual[T comparable](a, b T) bool { return a == b }

func HandleGroups(fns []func()) {
	wg := new(sync.WaitGroup)

	for _, fn := range fns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	wg.Wait()
}
=======
package utils

import "sync"

func BasicEqual[T comparable](a, b T) bool { return a == b }

func HandleGroups(fns []func()) {
	wg := new(sync.WaitGroup)

	for _, fn := range fns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	wg.Wait()
}
>>>>>>> 2330299ae49ffbbd1dbfff4a195c0169abca5303
