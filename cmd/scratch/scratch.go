package main

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

func main() {
	t1 := time.Now()

	fmt.Println(t1.UnixMilli())
	fmt.Println(math.MaxUint32)
	mu := strconv.FormatUint(math.MaxUint64, 10)
	fmt.Println(mu)
	
}
