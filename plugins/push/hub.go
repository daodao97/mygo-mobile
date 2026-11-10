package push

import (
	"sync"

	"github.com/egoist/mygo/internal/pushhub"
)

// The listeners of this package, which mygo reaches through pushhub's
// hooks, set as the package initializes.

func init() {
	pushhub.Received = received
	pushhub.Registered = registered
}

// list is a list of listeners, safe from any goroutine, told in the order
// they were added.
type list[T any] struct {
	mu   sync.Mutex
	next int
	ids  []int
	fns  []T
}

func (l *list[T]) add(fn T) (off func()) {
	l.mu.Lock()
	defer l.mu.Unlock()
	id := l.next
	l.next++
	l.ids, l.fns = append(l.ids, id), append(l.fns, fn)
	return func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		for i, v := range l.ids {
			if v == id {
				l.ids = append(l.ids[:i:i], l.ids[i+1:]...)
				l.fns = append(l.fns[:i:i], l.fns[i+1:]...)
				return
			}
		}
	}
}

// each calls call with every listener, outside the lock, and reports
// whether there was one.
func (l *list[T]) each(call func(T)) bool {
	l.mu.Lock()
	fns := append([]T(nil), l.fns...)
	l.mu.Unlock()
	for _, fn := range fns {
		call(fn)
	}
	return len(fns) > 0
}

var (
	tokens   list[func(string)]
	errs     list[func(error)]
	messages list[func(pushhub.Event)]
	opened   list[func(pushhub.Event)]

	mu      sync.Mutex
	pending []pushhub.Event          // taps before any OnOpened
	present func(pushhub.Event) bool // SetPresentation's
)

func received(e pushhub.Event) bool {
	if !e.Remote {
		return shows(e)
	}
	if e.Clicked {
		if !opened.each(func(fn func(pushhub.Event)) { fn(e) }) {
			mu.Lock()
			pending = append(pending, e)
			mu.Unlock()
		}
		return true
	}
	show := shows(e)
	messages.each(func(fn func(pushhub.Event)) { fn(e) })
	return show
}

func shows(e pushhub.Event) bool {
	if e.Clicked {
		return true
	}
	mu.Lock()
	fn := present
	mu.Unlock()
	return fn == nil || fn(e)
}

func registered(token string, err error) {
	if err != nil {
		errs.each(func(fn func(error)) { fn(err) })
		return
	}
	tokens.each(func(fn func(string)) { fn(token) })
}

func onOpened(fn func(pushhub.Event)) (off func()) {
	off = opened.add(fn)
	mu.Lock()
	events := pending
	pending = nil
	mu.Unlock()
	for _, e := range events {
		fn(e)
	}
	return off
}
