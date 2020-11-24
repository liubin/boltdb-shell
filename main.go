package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/abiosoft/ishell"
	bolt "go.etcd.io/bbolt"
)

type ItemType string

const ItemTypeBucket = "bucket"
const ItemTypeKey = "key"

type StackItem struct {
	ItemType ItemType
	Name     string
	Parents  []string
}

var stack []*StackItem
var currentStackItem *StackItem

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

	shell.Println("BoltDB shell")

	shell.AddCmd(&ishell.Cmd{
		Name: "ls",
		Help: "list bucket or keys",
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

	// run shell
	shell.Run()
}

func cmdPWD(ic *ishell.Context) {
	if len(stack) == 0 {
		ic.Println("/")
	} else {
		last := stack[len(stack)-1]
		p := append(last.Parents, last.Name)
		ic.Printf("/ -> %s\n", strings.Join(p, " -> "))
	}
}

func cmdINT(ic *ishell.Context, db *bolt.DB) {
	if len(ic.Args) != 1 {
		ic.Println("Must use int <key>")
		return
	}

	key := ic.Args[0]

	if currentStackItem == nil {
		// FIXME
	} else {
		db.View(func(tx *bolt.Tx) error {
			bk := findBucket(tx, currentStackItem.Parents, currentStackItem.Name)
			if bk == nil {
				ic.Printf("Not found bucket: %s\n", currentStackItem.Name)
				return nil
			}
			value := bk.Get([]byte(key))
			id, _ := binary.Uvarint(value)
			ic.Printf("%d\n", id)
			return nil
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
			currentStackItem = stack[len(stack)-2]
			stack = stack[:len(stack)-1]
		} else {
			currentStackItem = nil
			stack = []*StackItem{}
		}
		return
	} else if bucketName == "/" {
		currentStackItem = nil
		stack = []*StackItem{}
	}

	db.View(func(tx *bolt.Tx) error {
		var bk *bolt.Bucket
		parents := []string{}
		if currentStackItem == nil {
			// first
			bk = tx.Bucket([]byte(bucketName))
		} else {
			pbk := findBucket(tx, currentStackItem.Parents, currentStackItem.Name)
			if pbk != nil {
				// FIXME
			}
			bk = pbk.Bucket([]byte(bucketName))
		}

		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			parents = append(parent.Parents, parent.Name)
		}

		if bk != nil {
			// found
			currentStackItem = &StackItem{
				ItemType: ItemTypeBucket,
				Name:     bucketName,
				Parents:  parents,
			}
			stack = append(stack, currentStackItem)
		} else {
			// FIXME
		}

		return nil
	})
}

func cmdLS(ic *ishell.Context, db *bolt.DB) {
	db.View(func(tx *bolt.Tx) error {
		if currentStackItem == nil {
			tx.ForEach(func(name []byte, bucket *bolt.Bucket) error {
				ic.Println(s(name))
				return nil
			})
		} else {
			bk := findBucket(tx, currentStackItem.Parents, currentStackItem.Name)
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

func findBucket(tx *bolt.Tx, parents []string, name string) *bolt.Bucket {
	var bk *bolt.Bucket
	if len(parents) == 0 {
		bk = tx.Bucket([]byte(name))
	} else {
		for i, s := range parents {
			if i == 0 {
				bk = tx.Bucket([]byte(s))
			} else {
				bk = bk.Bucket([]byte(s))
			}
		}
		if bk != nil {
			bk = bk.Bucket([]byte(name))
		}

	}

	return bk
}
