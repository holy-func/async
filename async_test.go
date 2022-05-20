package async

import (
	"fmt"
	"testing"
	"time"
)

type thenAble struct {
	name string
	age  int
}

func (t *thenAble) setThen(resolved bool, val int) promiseTask {
	return func(resolve, reject Handler) {
		if resolved {
			resolve(val)
		} else {
			reject(val)
		}
	}
}
func asset(test bool) {
	if !test {
		panic("error")
	}
}
func setTasks() *Plain {
	return &Plain{1, 2,
		Promise(func(resolve, reject Handler) {
			resolve(3)
		}),
		Promise(func(resolve, reject Handler) {
			reject(4)
		}),
		&thenAble{"hloy", 19},
	}
}
func TestAll(t *testing.T) {
	PROD()
	promise := Do(func(params ...interface{}) interface{} {
		asset(params[0].(int) == 1)
		return 1 + params[0].(int)
	}, 1).Then(func(ret interface{}) interface{} {
		fmt.Println(ret,1)
		asset(ret.(int) == 2)
		return Resolve(ret)
	}, nil).Then(func(ret interface{}) interface{} {
		asset(ret.(int) == 2)
		return Reject(ret)
	}, nil).Then(nil, func(ret interface{}) interface{} {
		asset(ret.(int) == 2)
		return Reject(ret)
	}).Catch(func(ret interface{}) interface{} {
		asset(ret.(int) == 2)
		return Promise((&thenAble{"holy-func", 19}).setThen(true, 2))
	}).Finally(func() {
		All(setTasks()).UnsafeAwait()
		AllSettled(setTasks()).UnsafeAwait()
		Any(setTasks()).UnsafeAwait()
		Race(setTasks()).UnsafeAwait()
		setTasks().All()
		setTasks().AllSettled()
		setTasks().Any()
		setTasks().Race()
	})
	promise.Then(func(i interface{}) interface{} {
		time.Sleep(time.Second*2)
		fmt.Println(i, 3)
		return 2
	}, nil)
	promise.Then(func(i interface{}) interface{} {
		fmt.Println(i, 2)
		return 3
	}, nil)
	fmt.Println(promise)
	Wait()
	fmt.Println(promise)
}
// Promise { <pending> }
// 2 1
// 2 2
// 2 3
// Promise { 2 }
// PASS
// coverage: 85.1% of statements
// ok      github.com/holy-func/async   2.597s