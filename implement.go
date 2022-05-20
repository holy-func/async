package async

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
