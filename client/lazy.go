package client

import "io"

type lazy[T any] struct {
	val    T
	loaded bool
	initFn func() (T, error)
}

func (l *lazy[T]) get() (T, error) {
	if l.loaded {
		return l.val, nil
	}
	v, err := l.initFn()
	if err != nil {
		var zero T
		return zero, err
	}
	l.val = v
	l.loaded = true
	return v, nil
}

func (l *lazy[T]) set(v T) {
	l.val = v
	l.loaded = true
}

func (l *lazy[T]) isLoaded() bool {
	return l.loaded
}

func (l *lazy[T]) reset() {
	var zero T
	l.val = zero
	l.loaded = false
}

func (l *lazy[T]) close() error {
	if !l.loaded {
		return nil
	}
	if any(l.val) != nil {
		if c, ok := any(l.val).(io.Closer); ok {
			err := c.Close()
			l.reset()
			return err
		}
	}
	l.reset()
	return nil
}
