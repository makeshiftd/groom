package main

import (
    "os"
    "fmt"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Print("DEFAULT")
        return
    }
    fmt.Print(os.Args[1])
}
