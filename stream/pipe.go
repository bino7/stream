package stream

type Pipe struct {
	in  Stream
	out Stream
}

func NewPipe(in,out Stream) *Pipe{
	return &Pipe{in,out}
}

func (p *Pipe) In() chan <-interface{} {
	return p.in
}

func (p *Pipe) Out() <- chan interface{} {
	return p.out
}
