package async

var asyncFuncs = []*promise{}
var promiseTasks = []*promise{}
var Pending = "0"
var Resolved = "1"
var Rejected = "-1"

type lock struct {
	async chan int
	ret   interface{}
	state string
}
type Handler func(interface{})
type promiseTask func(Handler, Handler)
type CallBack func(interface{}) interface{}
type finally func()
type AsyncTask func(...interface{}) interface{}

func defaultResolvedThenHandler(ret interface{}) interface{} {
	return ret
}

type Async struct {
	Await func() interface{}
}
type promise struct {
	*lock
	then    [][]CallBack
	finally []finally
}

func (l *lock) execAsync(task AsyncTask, params []interface{}) {
	go l.end()
	l.ret = task(params...)
	l.start()
}

func (l *lock) Await() interface{} {
	if l.state != Pending {
		return l.ret
	}
	l.end()
	return l.ret
}
func (l *lock) start() {
	close(l.async)
}
func (l *lock) end() {
	if l.state != Pending {
		return
	}
	<-l.async
	l.state = Resolved
}
func (p *promise) start() {
	p.thenHandler()
	p.finallyHandler()
	p.lock.start()
}
func (p *promise) Then(callbacks ...CallBack) *promise {
	p.then = append(p.then, callbacks)
	return p
}
func (p *promise) Catch(callback CallBack) *promise {
	p.then = append(p.then, []CallBack{defaultResolvedThenHandler, callback})
	return p
}
func (p *promise) Finally(finally func()) *promise {
	p.finally = append(p.finally, finally)
	return p
}
func (p *promise) finallyHandler() {
	for _, callback := range p.finally {
		callback()
	}
}
func (p *promise) thenHandler() {
	for _, callback := range p.then {
		if p.state == Resolved && len(callback) > 0 {
			p.ret = callback[0](p.ret)
		} else if p.state == Rejected && len(callback) == 2 {
			p.ret = callback[1](p.ret)
		}
	}
}
func (p *promise) execpromise(task promiseTask) {
	go p.end()
	task(p.Resolve, p.Reject)
	p.start()
}
func (p *promise) Resolve(ret interface{}) {
	if p.state != Pending {
		return
	}
	p.ret = ret
	p.state = Resolved
}
func (p *promise) Reject(ret interface{}) {
	if p.state != Pending {
		return
	}
	p.ret = ret
	p.state = Rejected
}
func Do(task AsyncTask, params ...interface{}) *Async {
	return executeAsyncTask(task, &asyncFuncs, params)
}
func executeAsyncTask(task AsyncTask, store *[]*promise, params []interface{}) *Async {
	async := &promise{&lock{make(chan int), nil, Pending}, nil, nil}
	*store = append(*store, async)
	go async.execAsync(task, params)
	return &Async{async.Await}
}
func executepromiseTask(task promiseTask, store *[]*promise) *promise {
	async := &promise{&lock{make(chan int), nil, Pending}, [][]CallBack{}, []finally{}}
	*store = append(*store, async)
	go async.execpromise(task)
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
func Promise(task promiseTask) *promise {
	return executepromiseTask(task, &promiseTasks)
}
