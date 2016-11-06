# gohessian

gohessian implements  Hessian   Protocol by golang

### Install
```sh
$ go get github.com/hackez/gohessian
```

### Usage

```go
package main

import (
    "fmt"
    gh "github.com/hackez/gohessian"
)

func main() {
    c := gh.NewClient("http://www.example.com", "/helloworld")
    res, err := c.Invoke("sendInt", 1)
    if err != nil {
        fmt.Printf("Hessian Invoke error:%s\n",err)
        return
    }
    fmt.Printf("Hessian Invoke Success, result:%s\n", res)
}
```
