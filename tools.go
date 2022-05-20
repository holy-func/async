package async

import (
	"sync"
)

func catchError(p *GoPromise) {
	err := recover()
	if p.settled() {
		err = nil
	} else if err != nil {
		p.err = true
		p.reject(err)
	}
}
func meaninglessError() {
	if env == prod {
		return
	}
	panic("this func call is meaningless")
}
func (p *GoPromise) execPromise(task promiseTask) {
	defer catchError(p)
	task(p.resolve, p.reject)
}
func collectTask() {
	wg.Add(1)
}
func clearTask() {
	wg.Done()
}
func gPromise() *GoPromise {
	return &GoPromise{&lock{make(chan int), nil, Pending, false, &sync.Once{}}, nil}
}
func isPromise(v interface{}) (*GoPromise, bool) {
	obj, ok := v.(*GoPromise)
	return obj, ok
}
func isThenable(v interface{}) (Thenable, bool) {
	obj, ok := v.(Thenable)
	return obj, ok
}
func deepAwait(promise, ret, err interface{}) (interface{}, interface{}) {
	newRet, newErr := ret, err
	if v, ok := isPromise(promise); ok {
		newRet, newErr = v.Await()
	} else if thenableStruct, isThenable := isThenable(promise); isThenable {
		promise := Promise(thenableStruct.Then)
		newRet, newErr = promise.Await()
	}
	return newRet, newErr
}
func funcCallWithErrorIntercept(tasks *Tasks, funcName string) *GoPromise {
	if len(*tasks) == 0 {
		meaninglessError()
		return nil
	}
	return tools[funcName](tasks)
}
func safeAct() *counter {
	return &counter{0, sync.Mutex{}}
}
