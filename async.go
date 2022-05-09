package async

import (
	"fmt"
)

func (p *Plain) AllSettled() *GoPromise {
	return AllSettled(p)
}
func (s *Settled) String() string {
	return fmt.Sprintf("{ Status: %s, Value: %s }", s.Status, s.Value)
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
func (l *lock) resolved() bool {
	return l.state == Resolved
}
func (l *lock) rejected() bool {
	return l.state == Rejected
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
