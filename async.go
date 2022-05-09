package async

import (
	"fmt"
	"sync"
)

const Pending State = "pending"
const Resolved State = "resolved"
const Rejected State = "rejected"
const dev = "development"
const prod = "production"

var wg sync.WaitGroup
var env = dev
var tools = map[string]func(*Tasks) *GoPromise{"Race": race, "All": all, "Any": any, "AllSettled": allSettled}

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
type Tasks []*GoPromise
type Plain []interface{}
type counter struct {
	count int
	m     sync.Mutex
}
type Settled struct {
	Status State
	Value  interface{}
}

func (p *Plain) AllSettled() *GoPromise {
	return AllSettled(p)
}
func (s *Settled) String() string {
	return fmt.Sprintf("{ Status: %s, Value: %s }", s.Status, s.Value)
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
func (c *counter) do(action func(*counter)) {
	c.m.Lock()
	action(c)
	c.m.Unlock()
}
func (c *counter) value() (ret int) {
	c.m.Lock()
	ret = c.count
	c.m.Unlock()
	return
}
func (p *Plain) toPromise() *Tasks {
	ret := make(Tasks, len(*p))
	for i, v := range *p {
		ret[i] = Resolve(v)
	}
	return &ret
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
	close(p.async)
	p.callback()
	clearTask()
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
	panic("uncatched error " + l.String())
}
func (l *lock) uncatchedRejected() {
	if env == prod {
		return
	}
	panic("uncatched rejected " + l.String())
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
func (p *GoPromise) resolve(v interface{}) {
	if !p.settled() {
		ret, err := deepAwait(v, v, nil)
		if err != nil {
			p.reject(err)
		} else {
			p.state = Resolved
			p.ret = ret
		}
		p.endTask()
	}
}
func (p *GoPromise) reject(v interface{}) {
	if !p.settled() {
		p.ret = v
		p.state = Rejected
		p.endTask()
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
func (l *lock) resolved() bool {
	return l.state == Resolved
}
func (l *lock) rejected() bool {
	return l.state == Rejected
}
func Resolve(v interface{}) *GoPromise {
	if then, ok := isThenable(v); ok {
		return Promise(then.Then)
	} else if promise, ok := isPromise(v); ok {
		return promise
	} else {
		return &GoPromise{&lock{make(chan int), v, Resolved, false}, nil}
	}
}
func Reject(v interface{}) *GoPromise {
	return &GoPromise{&lock{make(chan int), v, Rejected, false}, nil}
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
		p.err = true
		p.reject(err)
	}
}
func (l *lock) String() string {
	if l.rejected() {
		return fmt.Sprintf("Promise { <%v> %v }", l.state, l.ret)
	} else if l.resolved() {
		return fmt.Sprintf("Promise { %v }", l.ret)
	} else {
		return fmt.Sprintf("Promise { <%v> }", l.state)
	}
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

func funcCallWithErrorIntercept(tasks *Tasks, funcName string) *GoPromise {
	if len(*tasks) == 0 {
		meaninglessError()
		return nil
	}
	return tools[funcName](tasks)
}

func All(plain *Plain) *GoPromise {
	return funcCallWithErrorIntercept(plain.toPromise(), "All")
}
func Race(plain *Plain) *GoPromise {
	return funcCallWithErrorIntercept(plain.toPromise(), "Race")
}
func AllSettled(plain *Plain) *GoPromise {
	return funcCallWithErrorIntercept(plain.toPromise(), "AllSettled")
}
func Any(plain *Plain) *GoPromise {
	return funcCallWithErrorIntercept(plain.toPromise(), "Any")
}
func Wait() {
	wg.Wait()
}
func race(promises *Tasks) *GoPromise {
	var newPromise *GoPromise
	newPromise = Promise(func(resolve, reject Handler) {
		for _, promise := range *promises {
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
			}(promise)
		}
	})
	return newPromise
}
func all(promises *Tasks) *GoPromise {
	num := len(*promises)
	result := make([]interface{}, num)
	count := safeAct()
	var newPromise *GoPromise
	newPromise = Promise(func(resolve, reject Handler) {
		for i, promise := range *promises {
			wg.Add(1)
			go func(i int, promise *GoPromise) {
				ret, err := promise.UnsafeAwait()
				if newPromise.settled() {
					return
				}
				if err == nil {
					count.do(func(c *counter) {
						result[i] = ret
						c.count++
					})
					if count.value() == num {
						resolve(result)
					}
				} else {
					reject(err)
				}
			}(i, promise)
		}
	})
	return newPromise
}
func allSettled(promises *Tasks) *GoPromise {
	return Promise(func(resolve, reject Handler) {
		result := make([]*Settled, len(*promises))
		for i, promise := range *promises {
			ret, err := promise.UnsafeAwait()
			if err == nil {
				result[i] = &Settled{Resolved, ret}
			} else {
				result[i] = &Settled{Rejected, err}
			}
		}
		resolve(result)
	})
}
func safeAct() *counter {
	return &counter{0, sync.Mutex{}}
}
func any(promises *Tasks) *GoPromise {
	num := len(*promises)
	result := make([]interface{}, num)
	count := safeAct()
	var newPromise *GoPromise
	newPromise = Promise(func(resolve, reject Handler) {
		for i, promise := range *promises {
			go func(i int, promise *GoPromise) {
				ret, err := promise.UnsafeAwait()
				if newPromise.settled() {
					return
				}
				if err != nil {
					count.do(func(c *counter) {
						result[i] = err
						c.count++
					})
					if count.value() == num {
						reject(result)
					}
				} else {
					resolve(ret)
				}
			}(i, promise)
		}
	})
	return newPromise
}
