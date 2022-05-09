package async

import (
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
