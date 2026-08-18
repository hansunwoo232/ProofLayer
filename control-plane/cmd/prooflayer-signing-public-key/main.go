// prooflayer-signing-public-key derives the public key for the local Day 30
// Control Plane seed without printing the seed itself.
package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
)

func main() {
	seed, err := base64.RawURLEncoding.DecodeString(os.Getenv("PROOFLAYER_SIGNING_SEED"))
	if err != nil || len(seed) != ed25519.SeedSize {
		fmt.Fprintln(os.Stderr, "PROOFLAYER_SIGNING_SEED is invalid")
		os.Exit(1)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	fmt.Println(base64.RawURLEncoding.EncodeToString(publicKey))
}
