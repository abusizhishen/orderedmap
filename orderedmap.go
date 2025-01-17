package orderedmap

import (
	"bytes"
	"container/list"
	"encoding/gob"
	"encoding/json"
	"sync"
	"errors"
)

type orderedMapElement struct {
	key, value interface{}
}

type OrderedMap struct {
	kv map[interface{}]*list.Element
	ll *list.List
	sync.RWMutex
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		kv: make(map[interface{}]*list.Element),
		ll: list.New(),
	}
}

// Get returns the value for a key. If the key does not exist, the second return
// parameter will be false and the value will be nil.
func (m *OrderedMap) Get(key interface{}) (interface{}, bool) {
	m.RLock()
	defer m.RUnlock()
	value, ok := m.kv[key]
	if ok {
		return value.Value.(*orderedMapElement).value, true
	}

	return nil, false
}

// Set will set (or replace) a value for a key. If the key was new, then true
// will be returned. The returned value will be false if the value was replaced
// (even if the value was the same).
func (m *OrderedMap) Set(key, value interface{}) bool {
	m.Lock()
	defer m.Unlock()
	_, didExist := m.kv[key]

	if !didExist {
		element := m.ll.PushBack(&orderedMapElement{key, value})
		m.kv[key] = element
	} else {
		m.kv[key].Value.(*orderedMapElement).value = value
	}

	return !didExist
}

// GetOrDefault returns the value for a key. If the key does not exist, returns
// the default value instead.
func (m *OrderedMap) GetOrDefault(key, defaultValue interface{}) interface{} {
	m.RLock()
	defer m.RUnlock()
	if value, ok := m.kv[key]; ok {
		return value.Value.(*orderedMapElement).value
	}

	return defaultValue
}

// Len returns the number of elements in the map.
func (m *OrderedMap) Len() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.kv)
}

// Keys returns all of the keys in the order they were inserted. If a key was
// replaced it will retain the same position. To ensure most recently set keys
// are always at the end you must always Delete before Set.
func (m *OrderedMap) Keys() (keys []interface{}) {
	m.RLock()
	defer m.RUnlock()
	keys = make([]interface{}, m.Len())

	element := m.ll.Front()
	for i := 0; element != nil; i++ {
		keys[i] = element.Value.(*orderedMapElement).key
		element = element.Next()
	}

	return keys
}

// Delete will remove a key from the map. It will return true if the key was
// removed (the key did exist).
func (m *OrderedMap) Delete(key interface{}) (didDelete bool) {
	m.Lock()
	defer m.Unlock()
	element, ok := m.kv[key]
	if ok {
		m.ll.Remove(element)
		delete(m.kv, key)
	}

	return ok
}

// Front will return the element that is the first (oldest Set element). If
// there are no elements this will return nil.
func (m *OrderedMap) Front() *Element {
	m.RLock()
	defer m.RUnlock()
	front := m.ll.Front()
	if front == nil {
		return nil
	}

	element := front.Value.(*orderedMapElement)

	return &Element{
		element: front,
		Key:     element.key,
		Value:   element.value,
	}
}

// Back will return the element that is the last (most recent Set element). If
// there are no elements this will return nil.
func (m *OrderedMap) Back() *Element {
	m.RLock()
	defer m.RUnlock()
	back := m.ll.Back()
	if back == nil {
		return nil
	}

	element := back.Value.(*orderedMapElement)

	return &Element{
		element: back,
		Key:     element.key,
		Value:   element.value,
	}
}

// marshal json to save
func (m *OrderedMap) MarshalJSON() ([]byte, error) {
	var keys = m.Keys()
	var collection = make([]interface{}, 0, len(keys)*2)
	var data interface{}
	for _, key := range keys {
		data, _ = m.Get(key)
		collection = append(collection, key)
		collection = append(collection, data)
	}

	var buf = new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(collection)
	if err != nil {
		return nil, err
	}
	return json.Marshal(buf.Bytes())
}

// unmarshal json to load byte
func (m *OrderedMap) UnmarshalJSON(data []byte) error {
	var bys []byte
	err := json.Unmarshal(data, &bys)
	if err != nil {
		return err
	}

	var collection []interface{}
	var buf = bytes.NewReader(bys)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&collection)
	if err != nil {
		return err
	}

	length := len(collection)
	count := length >> 1
	if count<<1 != length {
		return errors.New("invalid data, key-value doesn't match")
	}

	var idx int
	for i := 0; i < count; i++ {
		idx = i << 1
		m.Set(collection[idx], collection[idx+1])
	}

	return nil
}
