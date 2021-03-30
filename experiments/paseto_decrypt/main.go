package main

import (
	"fmt"
	"github.com/o1egl/paseto"
)

func main() {
	symmetricKey := []byte("91381577                        ") // Must be 32 bytes

	// Decrypt data
	var newJsonToken paseto.JSONToken
	var newFooter string
	var token = "v2.local.EdrOpzWazESWXbR9G10u_C3zZHp7zOzYg4nqTN9E1l_zZ7VsL3RmDpY_M9AiHvKfU0_pgQgm9JlfqYrP52whEjY8YTc"

	_, err := paseto.Parse(token, &newJsonToken, &newFooter, symmetricKey, nil)

	fmt.Println(err)
}
