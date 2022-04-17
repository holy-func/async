# async
一个模仿javascript中的async和promise的库，以同步的方式书写并发代码
### async.Do()
这个函数接受一个想要异步调用的函数asyncTask和它的参数，返回一个含有Await方法的结构体,Await方法可以用来阻塞调用该方法的函数执行并一直等待到asyncTask执行完成,并返回asyncTask函数的返回结果
### async.Promise()
这个函数接受一个想要异步调用的函数promiseTask,在调用时会传入resolve和reject方法用来控制该promise的状态，状态一旦settled(resolved or rejected)便不可以再改变,返回一个*async.promise结构体
#### *promise
##### Await()
等待promise settled 并返回结果
##### Then()
promise settled 后的回调函数 返回*promise
##### Catch() 
promise rejected 后的回调函数 返回*promise
##### Finally() 
promise settled后一定会执行的函数 返回*promise
### async.Wait()
阻塞调用该方法的函数并等待使用async包发起的所有asyncTask和promiseTask的完成

```
package main

import (
	"fmt"
	"github.com/holy-func/async"
	"time"
)

func action(params ...interface{}) interface{} {
	a, b := params[0].(int), params[1].(string)
	time.Sleep(time.Second * 2)
	fmt.Println(a, b)
	return 1
}
func promise(resolve, reject async.Handler) {
	fmt.Println("promise start!!!")
	time.Sleep(time.Second * 1)
	resolve(100)
	fmt.Println("promise end!!!")
	reject(-100)
}
func Then(i interface{}) interface{} {
	fmt.Println(i)
	return i.(int) + 1
}
func main() {
	task := async.Do(action, 10, "hi~")
	async.Do(action, 20, "ha~")
	c := async.Promise(promise)
	fmt.Println(c.Then(Then).Then(Then, Then).Finally(func() { fmt.Println("promise finally") }).Await(), "promise settled")
	fmt.Println(task.Await(), "task settled")
	async.Wait()
	fmt.Println("all tasks settled")
}
```
### 输出结果
```
promise start!!!
promise end!!!
100
101
promise finally
102 promise settled
20 ha~
10 hi~
1 task settled
all tasks settled
```
