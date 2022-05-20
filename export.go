package async

func DEV() {
	env = dev
}
func PROD() {
	env = prod
}
func Resolve(v interface{}) *GoPromise {
	if then, ok := isThenable(v); ok {
		return Promise(then.Then)
	} else if promise, ok := isPromise(v); ok {
		return promise
	} else {
		return &GoPromise{&lock{make(chan int), v, Resolved, false, nil}, nil}
	}
}
func Reject(v interface{}) *GoPromise {
	return &GoPromise{&lock{make(chan int), v, Rejected, false, nil}, nil}
}
func Do(task AsyncTask, params ...interface{}) *GoPromise {
	return Promise(func(resolve, reject Handler) {
		resolve(task(params...))
	})
}
func Promise(task promiseTask) *GoPromise {
	promise := gPromise()
	collectTask()
	go promise.execPromise(task)
	return promise
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
