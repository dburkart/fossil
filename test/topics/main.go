/*
 * Copyright (c) 2022, Dana Burkart <dana.burkart@gmail.com>
 *
 * SPDX-License-Identifier: BSD-2-Clause
 */

package main

import (
	"fmt"
	fossil "github.com/dburkart/fossil/api"
	"github.com/google/uuid"
	"os"
	"sync"
)

/*
 * This tests aggressive message spamming as well as agressive topic creation.
 * Each message sent to the server contains a completely new and unique topic.
 */

func main() {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := uuid.NewString()
			client, err := fossil.NewClientPool("fossil://localhost:8001/default", 10)
			if err != nil {
				os.Exit(1)
			}

			for i := 0; i < 1000; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					err := client.Append(fmt.Sprintf("%s/%d", id, i), []byte("some garbage"))
					if err != nil {
						os.Exit(1)
					}
				}(i)
			}
		}()
	}

	wg.Wait()
}
