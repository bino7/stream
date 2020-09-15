package stream

import "time"

type Streamer interface {
	Source() <-chan interface{}
	Play(time.Duration) error
	Stop() error
	Pause() error
	Set(int) error
	IsOver() bool
}
