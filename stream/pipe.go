package stream

type Pipe struct {
	in  Stream
	out Stream
}

func (p *Pipe) In() chan <-interface{} {
	return p.in
}

func (p *Pipe) Out() <- chan interface{} {
	return p.out
}
