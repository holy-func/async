package async

import (
	"sync"
)

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
