// scripts/generate_test_data.go
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

type Transaction struct {
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

var products = []string{
	"laptop_pro", "phone_x", "tablet_s", "headphones_z",
	"charger_usb", "keyboard_mech", "mouse_wireless", "monitor_4k",
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: generate_test_data <count> <output_file>")
		os.Exit(1)
	}

	count, err := strconv.Atoi(os.Args[1])
	if err != nil || count <= 0 {
		fmt.Println("count must be a positive integer")
		os.Exit(1)
	}

	file, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Printf("failed to create file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	start := time.Now()

	for i := 0; i < count; i++ {
		tx := Transaction{
			OrderID:   fmt.Sprintf("order_%d", i+1),
			UserID:    fmt.Sprintf("user_%d", rng.Intn(10000)+1),
			ProductID: products[rng.Intn(len(products))],
			Quantity:  rng.Intn(5) + 1,
			Price:     float64(rng.Intn(99000)+100) / 100.0,
			Timestamp: time.Now(),
		}
		if err := encoder.Encode(tx); err != nil {
			fmt.Printf("encode error: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("generated %d transactions in %s to %s\n", count, time.Since(start), os.Args[2])
}
