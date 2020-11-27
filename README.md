# bolt-shell

A simple BoltDB shell for exploring buckets and keys.

## Install

```
$ go get github.com/liubin/bolt-shell

$ bolt-shell meta.db
```

## Support commands

- `create <bucket>`: create bucket
- `delete <bucket>`: delete bucket
- `put <key> <value>`: put key value
- `delete_key <key>`: delete key.(alias of `put key`, that is putting key with empty value)
- `cd <bucket>` or `cd ..` or `cd /`
- `ls`: list buckets or keys under current path
- `pwd`: show current path(under which bucket)
- `int <key>`: show int value of `<key>`
- `time <key>`: show golang time value of `<key>`

## Demo

![](images/demo.gif)

converted by https://dstein64.github.io/gifcast/
