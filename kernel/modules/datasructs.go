package modules

import (
    "fmt"
)

type ListEntry struct {
    next *ListEntry
    value interface{}
    prev *ListEntry
}

type List struct {
    head *ListEntry
    tail *ListEntry
    size int
}

// Adds entry to the front of the List
func (q *List) LPush(v interface{}) {
    var e ListEntry
    e.value = v
    
    if q.head != nil {
        e.prev = q.head
        q.head.next = &e
        q.head = &e
    } else {
        q.head = &e
    }
    if q.tail == nil {
        q.tail = &e
    }
    q.size += 1
}

// Adds entry to Back of List
func (q *List) RPush(v interface{}) {
    var e ListEntry
    e.value = v

    if q.tail != nil {
        e.next = q.tail
        q.tail.prev = &e
        q.tail = &e
    } else {
        q.tail = &e
    }
    if q.head == nil {
        q.head = &e
    }
    q.size += 1
}

// Removes item at front of List
func (q *List) LPop() interface{} {
    if q.head == nil {
        return nil
    }
    v := q.head.value
    if q.head.prev != nil {
        q.head = q.head.prev
        q.head.next = nil
    } else {
        q.head = nil
    }
    
    q.size -= 1
    return v
}

// Removes item at end of List
func (q *List) RPop() interface{} {
    if q.tail == nil {
        return nil
    }
    v := q.tail.value
    if q.tail.next != nil {
        q.tail = q.tail.next
        q.tail.prev = nil
    } else {
        q.tail = nil
    }
    q.size -= 1
    return v
}

func (q *List) LPeek() interface{} {
    if q.head == nil {
        return nil
    }
    return q.head.value
}

func (q *List) RPeek() interface{} {
    if q.tail == nil {
        return nil
    }
    return q.tail.value
}

func (q *List) Size() int {
    return q.size
}

func (q List) String() string {
    if q.size == 0 {
        return "--empty--"
    }

    var first, last string
    if q.head != nil {
        first = fmt.Sprint(q.head.value)
    }
    if q.tail != nil {
        last = fmt.Sprint(q.tail.value)
    }
    i := q.size -1 
    return fmt.Sprintf("(0) %s (%d) %s", first, i, last)
}

func (q *List) ToList() []interface{} {
    c := []interface{}{}
    curr := q.head
    for curr != nil {
        c = append(c, curr.value)
        curr = curr.prev
    }
    return c
}

func (q *List) ItemAtIndex(i int) interface{} {
    if q.size == 0 {
        return nil
    }

    var reverse bool
    // start scanning from end
    if i < 0 {
        i = q.size + i
        reverse = true
    }
    // out of index check
    if i > q.size -1 {
        return nil
    }

    if !reverse {
        // find out if index is closer to head or tail
        m := q.size / 2 - 1
        if i > m {
            // start from end
            reverse = true
        } 
    }

    if reverse {
        // start from end
        curr := q.tail
        for j := q.size -1; j > i; j-- {
            curr = curr.next
        }
        return curr.value
    }
    // start from beginning
    curr := q.head
    for j := 1; j <= i; j++ {
        curr = curr.prev
    }
    return curr.value
}