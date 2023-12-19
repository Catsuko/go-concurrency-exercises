//////////////////////////////////////////////////////////////////////
//
// Given is some code to cache key-value pairs from a database into
// the main memory (to reduce access time). Note that golang's map are
// not entirely thread safe. Multiple readers are fine, but multiple
// writers are not. Change the code to make this thread safe.
//

package main

import (
	"container/list"
	"testing"
)

// CacheSize determines how big the cache can grow
const CacheSize = 100

// KeyStoreCacheLoader is an interface for the KeyStoreCache
type KeyStoreCacheLoader interface {
	// Load implements a function where the cache should gets it's content from
	Load(string) string
}

type page struct {
	Key   string
	Value string
}

// KeyStoreCache is a LRU cache for string key-value pairs
type KeyStoreCache struct {
	cache map[string]*list.Element
	pages chan *list.List
	load  func(string) string
}

// New creates a new KeyStoreCache
func New(load KeyStoreCacheLoader) *KeyStoreCache {
	pagesChannel := make(chan *list.List, 1)
	pagesChannel <- list.New()
	return &KeyStoreCache{
		load:  load.Load,
		cache: make(map[string]*list.Element),
		pages: pagesChannel,
	}
}

func (k *KeyStoreCache) PageLength() int {
	pages := <-k.pages
	length := pages.Len()
	k.pages <- pages
	return length
}

// TODO: Currently access to the cache is competely serialized.
// 			 Learn how this can be made concurrent whilst still supporting LRU
//			 https://www.openmymind.net/High-Concurrency-LRU-Caching/

// Get gets the key from cache, loads it from the source if needed
func (k *KeyStoreCache) Get(key string) string {
	pages := <-k.pages

	if e, ok := k.cache[key]; ok {
		pages.MoveToFront(e)
		k.pages <- pages
		return e.Value.(page).Value
	}
	// Miss - load from database and save it in cache

	// Return if another thread has loaded it in the meantime
	if e, ok := k.cache[key]; ok {
		k.pages <- pages
		return e.Value.(page).Value
	}

	p := page{key, k.load(key)}
	// if cache is full remove the least used item
	if len(k.cache) >= CacheSize {
		end := pages.Back()
		// remove from map
		delete(k.cache, end.Value.(page).Key)
		// remove from list
		pages.Remove(end)
	}
	pages.PushFront(p)
	k.cache[key] = pages.Front()
	k.pages <- pages
	return p.Value
}

// Loader implements KeyStoreLoader
type Loader struct {
	DB *MockDB
}

// Load gets the data from the database
func (l *Loader) Load(key string) string {
	val, err := l.DB.Get(key)
	if err != nil {
		panic(err)
	}

	return val
}

func run(t *testing.T) (*KeyStoreCache, *MockDB) {
	loader := Loader{
		DB: GetMockDB(),
	}
	cache := New(&loader)

	RunMockServer(cache, t)

	return cache, loader.DB
}

func main() {
	run(nil)
}
