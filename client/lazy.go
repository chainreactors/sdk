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

func (l *lazy[T]) isLoaded() bool {
	return l.loaded
}

func (l *lazy[T]) close() error {
	if !l.loaded {
		return nil
	}
	var err error
	if any(l.val) != nil {
		if c, ok := any(l.val).(io.Closer); ok {
			err = c.Close()
		}
	}
	var zero T
	l.val = zero
	l.loaded = false
	return err
}
