package codec_test

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/codec/v3"
)

func ExampleJSONCodec() {
	c := codec.JSONCodec{}

	type User struct {
		Name string `json:"name"`
	}

	data, err := c.Encode(User{Name: "Alice"})
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	var user User

	err = c.Decode(data, &user)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(user.Name)

	// Output:
	// Alice
}

func ExampleCBORCodec() {
	c := codec.CBORCodec{}

	type User struct {
		Name string `json:"name"`
	}

	data, err := c.Encode(User{Name: "Alice"})
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	var user User

	err = c.Decode(data, &user)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(user.Name)

	// Output:
	// Alice
}

func ExampleRawCodec() {
	c := codec.RawCodec{}

	raw := []byte(`{"name":"Bob"}`)

	data, err := c.Encode(raw)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	var decoded []byte

	err = c.Decode(data, &decoded)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(string(decoded))

	// Output:
	// {"name":"Bob"}
}
