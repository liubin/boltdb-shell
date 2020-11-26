# bolt-shell

A simple BoltDB shell for exploring buckets and keys.

## Install

```
$ go get github.com/liubin/bolt-shell

$ bolt-shell meta.db
```

## Support commands

- `cd <bucket>` or `cd ..` or `cd /`
- ls
- pwd
- `int <key>`: show int value of `<key>`
- `time <key>`: show golang time value of `<key>`


## Demo

![](images/demo.gif)

converted by https://dstein64.github.io/gifcast/
