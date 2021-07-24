package genetic

import (
	"crypto/rand"
	"log"
	"math/big"
)

// sample uniform from 0 - 1
func generateRandom() float64 {
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		log.Fatal("unable to sample from the distribution: ", err)
	}
	f, _ := new(big.Float).SetInt(n).Float64()

	return f / 100.0
}
