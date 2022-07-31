package main

type Limiter struct {
	limiter chan interface{}
}

func NewLimiter(number int) *Limiter {
	l := Limiter{
		limiter: make(chan interface{}, number),
	}

	return &l
}

func (l *Limiter) Start() {
	l.limiter <- nil
}

func (l *Limiter) Done() {
	<-l.limiter
}
