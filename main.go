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

	shell.AddCmd(&ishell.Cmd{
		Name:    "create",
		Help:    "create new bucket",
		Aliases: []string{"create_bucket"},
		Func: func(c *ishell.Context) {
			cmdCreateBucket(c, db)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name:    "delete",
		Help:    "delete a bucket",
		Aliases: []string{"delete_bucket"},
		Func: func(c *ishell.Context) {
			cmdDeleteBucket(c, db)
		},
	})

	shell.AddCmd(&ishell.Cmd{
		Name:    "put",
		Help:    "put key value pair under a bucket. Blank value will delete the key.",
		Aliases: []string{"delete_key"},
		Func: func(c *ishell.Context) {
			cmdPut(c, db)
		},
	})

	// run shell
	shell.Run()
}

func cmdPut(ic *ishell.Context, db *bolt.DB) {
	if len(ic.Args) == 0 {
		ic.Println("Must use put <key> [<value>]")
		return
	}

	key := ic.Args[0]

	// blank value will delete the key.
	value := ""
	if len(ic.Args) == 2 {
		value = ic.Args[1]
	}

	db.Update(func(tx *bolt.Tx) error {
		var err error
		bk := getCurrentBucket(tx)
		if bk == nil {
			// in root
			err = fmt.Errorf("Can't put key/value under root")
		} else {
			// under some bucket
			if value == "" {
				oldValue := bk.Get([]byte(key))
				// check if the key exist.
				if oldValue != nil {
					err = bk.Delete([]byte(key))
				} else {
					err = fmt.Errorf("Key %s not exist", key)
				}
			} else {
				err = bk.Put([]byte(key), []byte(value))
			}
		}

		if err != nil {
			ic.Printf("Put key/value failed: %s\n", err.Error())
		}

		return err
	})
}

func cmdDeleteBucket(ic *ishell.Context, db *bolt.DB) {
	if len(ic.Args) != 1 {
		ic.Println("Must use delete <bucket_name>")
		return
	}
	bucketName := ic.Args[0]

	db.Update(func(tx *bolt.Tx) error {
		var err error
		bk := getCurrentBucket(tx)
		if bk == nil {
			// in root
			err = tx.DeleteBucket([]byte(bucketName))
		} else {
			// under some bucket
			err = bk.DeleteBucket([]byte(bucketName))
		}

		if err != nil {
			ic.Printf("Delete bucket %s failed: %s\n", bucketName, err.Error())
		}

		return err
	})
}

func cmdCreateBucket(ic *ishell.Context, db *bolt.DB) {
	if len(ic.Args) != 1 {
		ic.Println("Must use create <bucket_name>")
		return
	}
	bucketName := ic.Args[0]

	db.Update(func(tx *bolt.Tx) error {
		var err error
		bk := getCurrentBucket(tx)
		if bk == nil {
			// in root
			_, err = tx.CreateBucket([]byte(bucketName))
		} else {
			// under some bucket
			newBk := bk.Bucket([]byte(bucketName))
			if newBk != nil {
				err = fmt.Errorf("Bucket %s already exist", bucketName)
			} else {
				// create if not exist
				_, err = bk.CreateBucket([]byte(bucketName))
			}
		}

		if err != nil {
			ic.Printf("Create bucket %s failed: %s\n", bucketName, err.Error())
		}

		return err
	})
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
	cmdConvret("time", ic, db, func(value []byte) error {
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
			bk := getCurrentBucket(tx)
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
		return
	}

	db.View(func(tx *bolt.Tx) error {
		var bk *bolt.Bucket

		currentStackItem := currentItem()
		if currentStackItem == nil {
			// first
			bk = tx.Bucket([]byte(bucketName))
		} else {
			pbk := getCurrentBucket(tx)
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
			ic.Printf("Bucket not found: %s\n", bucketName)
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
			bk := getCurrentBucket(tx)
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

// getCurrentBucket get current bucket, it will return nil if in the root
func getCurrentBucket(tx *bolt.Tx) *bolt.Bucket {
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
