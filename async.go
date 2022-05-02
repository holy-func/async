package async

import (
	"fmt"
	"sync"
)

const Pending State = "Pending"
const Resolved State = "Resolved"
const Rejected State = "Rejected"
const dev = "development"
const prod = "production"

var wg sync.WaitGroup
var env = dev

type Thenable interface {
	Then(Handler, Handler)
}

type lock struct {
	async chan int
	ret   interface{}
	state State
	err   bool
}
type State string
type Handler func(interface{})
type promiseTask func(Handler, Handler)
type CallBack func(interface{}) interface{}
type finallyHandler func()
type AsyncTask func(...interface{}) interface{}
type Settled struct {
	Status State
	value  interface{}
}
type then struct {
	success CallBack
	fail    CallBack
	*GoPromise
}
type catch struct {
	handler CallBack
	*GoPromise
}
type finally struct {
	handler finallyHandler
	*GoPromise
}
type prototype struct {
	*then
	*catch
	*finally
}

type GoPromise struct {
	*lock
	*prototype
}

func DEV() {
	env = dev
}
func PROD() {
	env = prod
}
func pendingERROR() {
	if env == prod {
		return
	}
	panic("promise will be pending forever")
}
func (p *GoPromise) callback() {
	if p.prototype != nil {
		defer func() {
			p.prototype = nil
		}()
		if p.prototype.then != nil {
			v := p.prototype.then
			if p.resolved() {
				go v.execPromise(func(resolve, reject Handler) {
					if v.success != nil {
						resolve(v.success(p.ret))
					} else {
						resolve(p.ret)
					}
				})
			} else if p.rejected() {
				go v.execPromise(func(resolve, reject Handler) {
					if v.fail != nil {
						p.err = false
						resolve(v.fail(p.ret))
					} else {
						reject(p.ret)
					}
				})
			}
		} else if p.prototype.catch != nil {
			v := p.prototype.catch
			if v.handler != nil {
				p.err = false
			}
			go v.execPromise(func(resolve, reject Handler) {
				if p.rejected() {
					if v.handler != nil {
						resolve(v.handler(p.ret))
					} else {
						reject(p.ret)
					}
				} else {
					resolve(p.ret)
				}
			})
		} else if p.prototype.finally != nil {
			v := p.prototype.finally
			go v.execPromise(func(resolve, reject Handler) {
				v.handler()
				resolve(p.ret)
			})
		}
	}
}
func (p *GoPromise) endTask() {
	if p.settled() {
		close(p.async)
		p.callback()
		clearTask()
	} else {
		pendingERROR()
	}
}
func (l *lock) waitTask() {
	if l.settled() {
		return
	}
	<-l.async
}
func (l *lock) uncatchedError() {
	if env == prod {
		return
	}
	fmt.Print("uncatched error ")
	panic(l.ret)
}
func (l *lock) uncatchedRejected() {
	if env == prod {
		return
	}
	fmt.Print("uncatched rejected ")
	panic(l.ret)
}
func meaninglessError() {
	if env == prod {
		return
	}
	panic("this func call is meaningless")
}
func (l *lock) settled() bool {
	return l.state == Rejected || l.state == Resolved
}
func (l *lock) resolve(v interface{}) {
	if !l.settled() {
		ret, err := deepAwait(v, v, nil)
		if err != nil {
			l.reject(err)
		} else {
			l.state = Resolved
			l.ret = ret
		}
	}
}
func (l *lock) reject(v interface{}) {
	if !l.settled() {
		l.ret = v
		l.state = Rejected
	}
}
func (l *lock) Await() (ret interface{}, err interface{}) {
	l.waitTask()
	if l.state == Resolved {
		err = nil
		ret = l.ret
		ret, err = deepAwait(ret, ret, err)
	} else {
		if l.err {
			l.uncatchedError()
		} else {
			l.uncatchedRejected()
		}
		err = l.ret
		ret = nil
	}
	return
}
func collectTask() {
	wg.Add(1)
}
func clearTask() {
	wg.Done()
}
func (l *lock) UnsafeAwait() (ret interface{}, err interface{}) {
	l.waitTask()
	if l.state == Resolved {
		err = nil
		ret = l.ret
		ret, err = deepAwait(ret, ret, err)
	} else {
		err = l.ret
		ret = nil
	}
	return
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
func (p *GoPromise) resolved() bool {
	return p.state == Resolved
}
func (p *GoPromise) rejected() bool {
	return p.state == Rejected
}
func Resolve(v interface{}) *GoPromise {
	_, ok1 := isThenable(v)
	_, ok2 := isPromise(v)
	if ok1 || ok2 {
		ret, err := deepAwait(v, v, nil)
		if err == nil {
			return Resolve(ret)
		} else {
			return Reject(err)
		}
	} else {
		return Promise(func(resolve, reject Handler) {
			resolve(v)
		})
	}
}
func Reject(v interface{}) *GoPromise {
	return Promise(func(resolve, reject Handler) {
		reject(v)
	})
}
func gPromise() *GoPromise {
	return &GoPromise{&lock{make(chan int), nil, Pending, false}, nil}
}
func Do(task AsyncTask, params ...interface{}) *GoPromise {
	return Promise(func(resolve, reject Handler) {
		resolve(task(params...))
	})
}
func catchError(p *GoPromise) {
	err := recover()
	if p.settled() {
		err = nil
	} else if err != nil {
		p.reject(err)
		p.err = true
	}
	p.endTask()
}

func (p *GoPromise) execPromise(task promiseTask) {
	defer catchError(p)
	task(p.resolve, p.reject)
}
func Promise(task promiseTask) *GoPromise {
	promise := gPromise()
	collectTask()
	go promise.execPromise(task)
	return promise
}
func (p *GoPromise) handleCallback() {
	if p.settled() {
		p.callback()
	}
}
func (p *GoPromise) Then(success, fail CallBack) *GoPromise {
	promise := gPromise()
	collectTask()
	p.prototype = &prototype{&then{success, fail, promise}, nil, nil}
	go p.handleCallback()
	return promise
}
func (p *GoPromise) Catch(handler CallBack) *GoPromise {
	promise := gPromise()
	collectTask()
	p.prototype = &prototype{nil, &catch{handler, promise}, nil}
	go p.handleCallback()
	return promise
}
func (p *GoPromise) Finally(handler finallyHandler) *GoPromise {
	promise := gPromise()
	collectTask()
	p.prototype = &prototype{nil, nil, &finally{handler, promise}}
	go p.handleCallback()
	return promise
}
func Wait() {
	wg.Wait()
}
func Race(promises ...*GoPromise) *GoPromise {
	if len(promises) == 0 {
		meaninglessError()
		return nil
	}
	wait := make(chan int)
	var newPromise *GoPromise
	newPromise = Promise(func(resolve, reject Handler) {
		for _, promise := range promises {
			go func(promise *GoPromise) {
				ret, err := promise.UnsafeAwait()
				if newPromise.settled() {
					return
				}
				if err == nil {
					resolve(ret)
				} else {
					reject(err)
				}
				close(wait)
			}(promise)
		}
		<-wait
	})
	return newPromise
}
func All(promises ...*GoPromise) *GoPromise {
	if len(promises) == 0 {
		meaninglessError()
		return nil
	}
	result := make([]interface{}, len(promises))
	var wg sync.WaitGroup
	count := 0
	var newPromise *GoPromise
	newPromise = Promise(func(resolve, reject Handler) {
		for i, promise := range promises {
			wg.Add(1)
			go func(i int, promise *GoPromise) {
				ret, err := promise.UnsafeAwait()
				if newPromise.settled() {
					return
				}
				if err == nil {
					result[i] = ret
					count++
					wg.Done()
				} else {
					reject(err)
					for i := count; i < len(promises); i++ {
						wg.Done()
					}
				}
			}(i, promise)
		}
		wg.Wait()
		resolve(result)
	})
	return newPromise
}
func AllSettled(promises ...*GoPromise) *GoPromise {
	if len(promises) == 0 {
		meaninglessError()
		return nil
	}
	return Promise(func(resolve, reject Handler) {
		result := make([]Settled, len(promises))
		for i, promise := range promises {
			ret, err := promise.UnsafeAwait()
			if err == nil {
				result[i] = Settled{Resolved, ret}
			} else {
				result[i] = Settled{Rejected, err}
			}
		}
		resolve(result)
	})
}
func Any(promises ...*GoPromise) *GoPromise {
	if len(promises) == 0 {
		meaninglessError()
		return nil
	}
	result := make([]interface{}, len(promises))
	var wg sync.WaitGroup
	count := 0
	var newPromise *GoPromise
	newPromise = Promise(func(resolve, reject Handler) {
		for i, promise := range promises {
			wg.Add(1)
			go func(i int, promise *GoPromise) {
				ret, err := promise.UnsafeAwait()
				if newPromise.settled() {
					return
				}
				if err != nil {
					result[i] = err
					count++
					wg.Done()
				} else {
					resolve(ret)
					for i := count; i < len(promises); i++ {
						wg.Done()
					}
				}
			}(i, promise)
		}
		wg.Wait()
		reject(result)
	})
	return newPromise
}
