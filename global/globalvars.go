package global

import (
	"sync"

	"labix.org/v2/mgo"
)

var ETHUSD float64
var Wg sync.WaitGroup
var MongoSession *mgo.Session
