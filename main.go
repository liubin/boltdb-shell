package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/abiosoft/ishell"
	bolt "go.etcd.io/bbolt"
)

type StackItem struct {
	Name string
}

var stack []*StackItem

func currentItem() *StackItem {
	if len(stack) == 0 {
		return nil
	}
	return stack[len(stack)-1]
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("db file needed")
		os.Exit(-1)
	}

	dbFile := os.Args[1]
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	shell := ishell.New()

	shell.Println("Simple BoltDB shell")

	shell.AddCmd(&ishell.Cmd{
		Name: "ls",
		Help: "list buckets or keys",
		Func: func(c *ishell.Context) {
			cmdLS(c, db)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "cd",
		Help: "change position to a bucket",
		Func: func(c *ishell.Context) {
			cmdCD(c, db)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "pwd",
		Help: "show current position",
		Func: func(c *ishell.Context) {
			cmdPWD(c)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "int",
		Help: "show int value",
		Func: func(c *ishell.Context) {
			cmdINT(c, db)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name: "time",
		Help: "show time value",
		Func: func(c *ishell.Context) {
			cmdTIME(c, db)
		},
	})

	// run shell
	shell.Run()
}

func cmdPWD(ic *ishell.Context) {
	if len(stack) == 0 {
		ic.Println("/")
	} else {
		p := make([]string, len(stack))
		for i := range stack {
			p[i] = stack[i].Name
		}
		ic.Printf("/ -> %s\n", strings.Join(p, " -> "))
	}
}

func cmdTIME(ic *ishell.Context, db *bolt.DB) {
	cmdConvret("tim", ic, db, func(value []byte) error {
		var t time.Time
		t.UnmarshalBinary(value)
		ic.Printf("%+v\n", t)
		return nil
	})
}

func cmdINT(ic *ishell.Context, db *bolt.DB) {
	cmdConvret("int", ic, db, func(value []byte) error {
		id, _ := binary.Uvarint(value)
		ic.Printf("%d\n", id)
		return nil
	})
}

func cmdConvret(cmd string, ic *ishell.Context, db *bolt.DB, fn func([]byte) error) {
	if len(ic.Args) != 1 {
		ic.Printf("Must use %s <key>\n", cmd)
		return
	}

	key := ic.Args[0]
	currentStackItem := currentItem()
	if currentStackItem == nil {
		// FIXME
	} else {
		db.View(func(tx *bolt.Tx) error {
			bk := findBucket(tx)
			if bk == nil {
				ic.Printf("Not found key: %s\n", key)
				return nil
			}
			value := bk.Get([]byte(key))
			return fn(value)
		})
	}
}

func cmdCD(ic *ishell.Context, db *bolt.DB) {
	if len(ic.Args) != 1 {
		ic.Println("Must use cd <bucket>")
		return
	}

	defer cmdPWD(ic)

	bucketName := ic.Args[0]
	if bucketName == ".." {
		if len(stack) > 1 {
			stack = stack[:len(stack)-1]
		} else {
			stack = []*StackItem{}
		}
		return
	} else if bucketName == "/" {
		stack = []*StackItem{}
	}

	db.View(func(tx *bolt.Tx) error {
		var bk *bolt.Bucket

		currentStackItem := currentItem()
		if currentStackItem == nil {
			// first
			bk = tx.Bucket([]byte(bucketName))
		} else {
			pbk := findBucket(tx)
			if pbk != nil {
				// FIXME
			}
			bk = pbk.Bucket([]byte(bucketName))
		}

		if bk != nil {
			// found
			stack = append(stack, &StackItem{
				Name: bucketName,
			})
		} else {
			// FIXME
		}

		return nil
	})
}

func cmdLS(ic *ishell.Context, db *bolt.DB) {
	db.View(func(tx *bolt.Tx) error {
		currentStackItem := currentItem()
		if currentStackItem == nil {
			// at root
			tx.ForEach(func(name []byte, bucket *bolt.Bucket) error {
				ic.Println(s(name))
				return nil
			})
		} else {
			bk := findBucket(tx)
			if bk == nil {
				ic.Printf("Not found bucket: %s\n", currentStackItem.Name)
				return nil
			}

			c := bk.Cursor()
			if c != nil {
				for k, v := c.First(); k != nil; k, v = c.Next() {
					tt := bk.Bucket(k)
					if tt != nil {
						// bucket
						ic.Printf("[Bucket] %s\n", s(k))
					} else {
						// key
						ic.Printf("[Key] %s=%s\n", s(k), s(v))
					}
				}
			}
		}

		return nil
	})
}

func isASCII(s string) bool {
	for _, c := range s {
		if c > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func s(b []byte) string {
	s := string(b)
	if isASCII(s) {
		return s
	}
	return fmt.Sprintf("%+v", b)
}

func findBucket(tx *bolt.Tx) *bolt.Bucket {
	var bk *bolt.Bucket
	for i, s := range stack {
		if i == 0 {
			bk = tx.Bucket([]byte(s.Name))
		} else {
			bk = bk.Bucket([]byte(s.Name))
		}
	}

	return bk
}
