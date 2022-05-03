[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/holy-func/async?tab=doc) 

介绍
------------

async是一个模仿javascript中的async和Promise的库,能够使开发者在Go语言中以同步的方式书写并发代码


安装
------------

##### Windows/macOS/Linux

```powershell
go get github.com/holy-func/async
```

文档
------------
### async.Do()
接收一个想要异步调用的函数asyncTask和它的参数,返回*async.GoPromise

### async.Promise()
接收一个想要异步调用的函数PromiseTask,在调用时会传入resolve和reject方法用来控制该Promise的状态,状态一旦settled(resolved or rejected)便不可以再改变,返回一个*async.GoPromise 与JavaScript中的[Promise](https://developer.mozilla.org/zh-CN/docs/Web/JavaScript/Reference/Global_Objects/Promise "Javascript Promise MDN")不同这里传入的回调函数不会立即执行即除非调用泛wait方法,即不会阻塞当前函数执行

```golang
func main() {
	promise := async.Promise(func(resolve, reject async.Handler) {
		resolve(100)
	})
	fmt.Println(promise)
	promise.Await()
	fmt.Println(promise)
}
```

### 输出结果
```
Promise { <pending> }
Promise { 100 }
```
#### *GoPromise

##### Await()
等待Promise settled 并返回结果P和错误信息,若reject未处理则panic
##### UnsafeAwait()
等待Promise settled 并返回结果和错误信息,无论是否rejected都可以获取到
##### Then()
Promise settled 后的回调函数 返回一个新的 *async.GOPromise
##### Catch() 
Promise rejected 后的回调函数 返回一个新的 *async.GOPromise
##### Finally() 
Promise settled后一定会执行的函数 返回一个新的 *async.GOPromise
### async.Wait()
阻塞调用该方法的函数并等待使用async包发起的所有asyncTask和PromiseTask的完成
### async.All()
接收n个 *async.GOPromise, 返回的 *async.GoPromise 在任意一个task rejected或者全部resolved的时候resolve
### async.Any()
接收n个 *async.GOPromise, 返回的 *async.GoPromise 在任意一个task resolved或者全部rejected的时候resolve
### async.Race()
接收n个 *async.GOPromise, 返回的 *async.GoPromise 在任意一个task resolved或者rejected的时候resolve
### async.AllSettled()
接收n个 *async.GOPromise, 返回的 *async.GoPromise 在所有task resolved或者rejected的时候resolve
### async.PROD()
开启生产环境,此时Await()与UnsafeAwait()等价,未处理的reject不会panic
### async.DEV()
开启开发环境,默认开启,未处理的reject会报错
### async.Resolve()
返回一个resolved的 *async.GoPromise并追踪终态
### async.Reject()
返回一个rejected的 *async.GoPromise不追踪终态

代码示例
------------

```golang
package main

import (
	"fmt"
	"time"
	"github.com/holy-func/async"
)

func action(b ...interface{}) interface{} {
	fmt.Println(b[0].(int) + b[1].(int))
	return 100
}
func actionA(resolve, reject async.Handler) {
	fmt.Println("A")
	reject(async.Do(action, 1, 2))
}
func actionB(resolve, reject async.Handler) {
	fmt.Println("B")
	reject(2)
}
func actionC(resolve, reject async.Handler) {
	fmt.Println("C")
	time.Sleep(time.Second)
	resolve(async.Do(action, 3, 4))
	resolve(3)
}

type then struct {
	name string
	age  int
}

func (t *then) Then(resolve, reject async.Handler) {
	fmt.Println("姓名:", t.name)
	fmt.Println("年龄:", t.age)
	fmt.Println("introduce over!!!")
	resolve(async.Promise(actionC))
}
func main() {
	a := async.Promise(actionA)
	b := async.Promise(actionB)
	c := async.Promise(actionC)
	a.Then(func(v interface{}) interface{} {
		fmt.Println("A resolved", v)
		return "AOK"
	}, func(err interface{}) interface{} {
		fmt.Println("A rejected", err)
		return "A!OK"
	}).Catch(func(err interface{}) interface{} {
		fmt.Println("这行代码不会执行因为错误已经被捕获了")
		return "无人问津的代码"
	}).Then(func(v interface{}) interface{} {
		fmt.Println(v, "task A")
		return 101
	}, nil).Finally(func() {
		fmt.Println("task A end!!!")
	})
	b.Catch(func(err interface{}) interface{} {
		fmt.Println("B rejected and catched", err)
		return "B failed"
	}).Then(func(v interface{}) interface{} {
		fmt.Println("resolved----", v)
		return "BOK"
	}, func(err interface{}) interface{} {
		fmt.Println("B rejected,错误已经被捕获这行代码不会被执行", err)
		return "B!OK"
	})
	c.Then(func(v interface{}) interface{} {
		fmt.Println("C resolved", v)
		fmt.Println("resolve 的Promise或*async.Thenable会采取最终态")
		return "COK"
	}, func(err interface{}) interface{} {
		fmt.Println("C rejected", err)
		return "C!OK"
	}).Await()
	async.Resolve(&then{"holy-func", 19}).Then(func(v interface{}) interface{} {
		fmt.Println("跟随resolve会得到最终的值为100 ----", v)
		return 1
	}, nil)
	async.Wait()
}
```
### 输出结果
```
B
B rejected and catched 2
resolved---- B failed
A
C
A rejected &{0xc00007a6c0 <nil>}
A!OK task A
task A end!!!
3
7
C resolved 100
resolve 的 Promise或 *async.Thenable 会采取最终态
姓名: holy-func
年龄: 19
introduce over!!!
C
7
跟随resolve会得到最终的值为100 ---- 100
```
