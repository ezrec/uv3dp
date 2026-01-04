//
// Copyright (c) 2020 Jason S. McMullan <jason.mcmullan@gmail.com>
//

package uv3dp

type Progressor interface {
	Show(percent float32)
	Stop()
}

type nilProgress struct{}

func (np *nilProgress) Show(float32) {}
func (np *nilProgress) Stop()        {}

var defaultProgress = Progressor(&nilProgress{})

func SetProgress(prog Progressor) {
	if prog == Progressor(nil) {
		prog = &nilProgress{}
	}
	defaultProgress = prog
}

type Progress struct {
	Progressor
	Completed chan struct{}
	Done      chan struct{}
}

func NewProgress(total int) (prog *Progress) {
	prog = &Progress{
		Progressor: defaultProgress,
		Completed:  make(chan struct{}, total),
		Done:       make(chan struct{}),
	}

	go func(prog *Progress) {
		for completion := 0; completion < total; completion++ {
			prog.Show(float32(completion) * 100.0 / float32(total))
			<-prog.Completed
		}
		prog.Show(100.0)
		prog.Stop()
		close(prog.Done)
	}(prog)

	return
}

func (prog *Progress) Indicate() {
	prog.Completed <- struct{}{}
}

func (prog *Progress) Close() {
	<-prog.Done
	close(prog.Completed)
}
