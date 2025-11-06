package utils

import (
	"log"
	"reflect"
	"sync"
)

type WaitGroupWrapper struct {
	wg            sync.WaitGroup
	processChan   chan int
	maxProcessCnt int
}

func NewWG(maxProcessCnt int) *WaitGroupWrapper {
	self := new(WaitGroupWrapper)
	if maxProcessCnt > 0 {
		self.maxProcessCnt = maxProcessCnt
		self.processChan = make(chan int, maxProcessCnt)
	}
	return self
}

func (w *WaitGroupWrapper) Add(delta int) {
	if w.maxProcessCnt > 0 {
		for i := 0; i < delta; i++ {
			w.processChan <- 1
		}
	}

	w.wg.Add(delta)
}

func (w *WaitGroupWrapper) Done() {
	if w.maxProcessCnt > 0 {
		<-w.processChan
	}
	w.wg.Done()
}

func (w *WaitGroupWrapper) Wait() {
	w.wg.Wait()
}

func (w *WaitGroupWrapper) Wrap(cb func()) {
	w.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("go routine run error: %+v, trace: %s\n", err, string(Stack(3)))
			}
		}()

		cb()

		w.Done()
	}()
}

// WrapFunc wrap with args, do not use this func in performance sensitive scenarios
func (w *WaitGroupWrapper) WrapFunc(fn interface{}, args ...interface{}) {
	vFn := reflect.ValueOf(fn)
	if vFn.Kind() == reflect.Func {
		w.Add(1)

		var vArgs []reflect.Value = nil
		if len(args) > 0 {
			vArgs = make([]reflect.Value, len(args))
			for idx, arg := range args {
				vArgs[idx] = reflect.ValueOf(arg)
			}
		}

		go func(fn reflect.Value, args []reflect.Value) {
			defer w.Done()
			defer func() {
				if err := recover(); err != nil {
					log.Printf("go routine run error: %+v, trace: %+v\n", err, string(Stack(3)))
				}
			}()

			fn.Call(args)

		}(vFn, vArgs)
	}
}
