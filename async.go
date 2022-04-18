package async

import (
	"reflect"
)

var asyncFuncs = []*GoPromise{}
var promiseTasks = []*GoPromise{}
var Pending State = "Pending"
var Resolved State = "Resolved"
var Rejected State = "Rejected"
var dev = "development"
var prod = "production"
var env = dev
var promiseType = &GoPromise{}

type lock struct {
	async chan int
	ret   interface{}
	state State
}
type State string
type Handler func(interface{})
type promiseTask func(Handler, Handler)
type CallBack func(interface{}) interface{}
type finally func()
type AsyncTask func(...interface{}) interface{}

func defaultResolvedThenHandler(ret interface{}) interface{} {
	return ret
}
func handleException(err *interface{}, reject Handler) {
	if *err != nil {
		reject(*err)
	}
}

type Async struct {
	Await func() (interface{}, interface{})
}
type GoPromise struct {
	*lock
	then    [][]CallBack
	finally []finally
	catched bool
}

func DEV() {
	env = dev
}
func PROD() {
	env = prod
}
func catchError(err *interface{}, catched bool, p *GoPromise) {
	if env == prod || catched || (p != nil && p.settled()) {
		*err = recover()
	}
}
func (l *lock) runError(err interface{}) {
	l.state = Rejected
	l.ret = err
}
func (l *lock) execAsync(task AsyncTask, params []interface{}) {
	go l.end()
	var err interface{}
	defer l.start()
	defer handleException(&err, l.runError)
	defer catchError(&err, false, nil)
	l.ret = task(params...)
}

func (l *lock) Await() (ret interface{}, err interface{}) {
	l.end()
	if l.state == Resolved {
		err = nil
		ret = l.ret
	} else if l.state == Rejected {
		err = l.ret
		ret = nil
	}
	return
}
func (p *GoPromise) Await() (ret interface{}, err interface{}) {
	ret, err = p.lock.Await()
	if reflect.TypeOf(ret) == reflect.TypeOf(promiseType) {
		promise := ret.(*GoPromise)
		ret, err = promise.Await()
	}
	return
}
func (l *lock) start() {
	close(l.async)
}
func (l *lock) end() {
	if l.settled() {
		return
	}
	<-l.async
	l.state = Resolved
}
func (l *lock) settled() bool {
	return l.state == Rejected || l.state == Resolved
}
func (p *GoPromise) start() {
	p.thenHandler()
	p.finallyHandler()
	p.lock.start()
}
func (p *GoPromise) Then(callbacks ...CallBack) *GoPromise {
	if len(callbacks) > 0 {
		p.then = append(p.then, callbacks)
		if len(callbacks) == 2 {
			p.catched = true
		}
	}
	return p
}
func (p *GoPromise) Catch(callback CallBack) *GoPromise {
	p.catched = true
	p.then = append(p.then, []CallBack{defaultResolvedThenHandler, callback})
	return p
}
func setExecption(p *GoPromise, excption bool) {
	p.catched = excption
}
func (p *GoPromise) NoExecption() *GoPromise {
	setExecption(p, true)
	return p
}
func (p *GoPromise) Finally(finally func()) *GoPromise {
	p.finally = append(p.finally, finally)
	p.then = append(p.then, nil)
	return p
}
func (p *GoPromise) finallyHandler() {
}
func (p *GoPromise) thenHandler() {
	current := p
	finallyIndex := 0
	for _, callback := range p.then {
		if callback == nil {
			func() {
				var err interface{}
				defer handleException(&err, current.unexpected)
				defer catchError(&err, true, nil)
				p.finally[finallyIndex]()
				finallyIndex++
			}()
			continue
		}
		if reflect.TypeOf(current.ret) == reflect.TypeOf(promiseType) {
			current = current.ret.(*GoPromise)
			func() {
				var err interface{}
				defer handleException(&err, current.reject)
				defer catchError(&err, true, nil)
				current.Await()
			}()
		}
		func() {
			var err interface{}
			defer handleException(&err, current.unexpected)
			defer catchError(&err, true, nil)
			if current.state == Resolved && len(callback) > 0 {
				current.ret = callback[0](current.ret)
			} else if current.state == Rejected && len(callback) == 2 {
				current.ret = callback[1](current.ret)
				current.state = Resolved
			}
		}()

	}
	if current.state == Rejected {
		panic("error not catch2")
		// 这里错误没有处理进行报错
	}
	if reflect.TypeOf(current.ret) == reflect.TypeOf(promiseType) {
		excption := current.catched
		current = current.ret.(*GoPromise)
		setExecption(current, excption)
		current.Await()
	}
}
func (p *GoPromise) execPromise(task promiseTask) {
	go p.end()
	var err interface{}
	defer p.start()
	defer handleException(&err, p.reject)
	defer catchError(&err, p.catched, p)
	task(p.resolve, p.reject)
}
func (p *GoPromise) resolve(ret interface{}) {
	if p.settled() {
		return
	}
	p.ret = ret
	p.state = Resolved
}
func (p *GoPromise) reject(ret interface{}) {
	if p.settled() {
		return
	}
	p.ret = ret
	p.state = Rejected
}
func (p *GoPromise) unexpected(ret interface{}) {
	p.state = Rejected
	p.ret = ret
}
func Do(task AsyncTask, params ...interface{}) *Async {
	return executeAsyncTask(task, &asyncFuncs, params)
}
func executeAsyncTask(task AsyncTask, store *[]*GoPromise, params []interface{}) *Async {
	async := gAsync()
	*store = append(*store, async)
	go async.execAsync(task, params)
	return &Async{async.Await}
}
func executePromiseTask(task promiseTask, store *[]*GoPromise) *GoPromise {
	async := gPromise(nil, Pending)
	*store = append(*store, async)
	go async.execPromise(task)
	return async
}
func Wait() {
	for _, async := range asyncFuncs {
		async.Await()
	}
	for _, async := range promiseTasks {
		async.Await()
	}
}
func Promise(task promiseTask) *GoPromise {
	return executePromiseTask(task, &promiseTasks)
}
func Resolve(v interface{}) *GoPromise {
	return gPromise(v, Resolved)
}
func gPromise(v interface{}, state State) *GoPromise {
	if reflect.TypeOf(v) == reflect.TypeOf(promiseType) {
		return v.(*GoPromise)
	}
	return &GoPromise{&lock{make(chan int), v, state}, [][]CallBack{}, []finally{}, false}
}
func gAsync() *GoPromise {
	return &GoPromise{&lock{make(chan int), nil, Pending}, nil, nil, false}
}
func Reject(v interface{}) *GoPromise {
	return gPromise(v, Rejected)
}
