package x_test

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/manishrjain/gocrud/x"
)

func ExampleUniqueString() {
	u := x.UniqueString(3)
	fmt.Println(len(u))
	// Output: 3
}

func ExampleParseIdFromUrl() {
	r, err := http.NewRequest("GET", "https://localhost/users/uid_12345", nil)
	if err != nil {
		panic(err)
	}
	uid, ok := x.ParseIdFromUrl(r, "/users/")
	if !ok {
		panic("Unable to parse uid")
	}
	fmt.Println(uid)

	r, err = http.NewRequest("GET", "https://localhost/users/uid_12345/", nil)
	if err != nil {
		panic(err)
	}
	uid, ok = x.ParseIdFromUrl(r, "/users/")
	if !ok {
		panic("Unable to parse uid")
	}
	fmt.Println(uid)

	// Output:
	// uid_12345
	// uid_12345/
}

func ExampleInstruction() {
	var i x.Instruction
	i.SubjectId = "sid"
	i.SubjectType = "stype"

	b, err := i.GobEncode()
	if err != nil {
		panic(err)
	}

	var o x.Instruction
	if err := o.GobDecode(b); err != nil {
		panic(err)
	}

	fmt.Println(o.SubjectId)
	fmt.Println(o.SubjectType)
	// Output:
	// sid
	// stype
}

func ExampleIts() {
	var its []x.Instruction

	for t := 0; t < 10; t++ {
		var i x.Instruction
		i.NanoTs = int64(100 - t)
		its = append(its, i)
	}

	sort.Sort(x.Its(its))
	fmt.Println(its[0].NanoTs)
	// Output: 91
}
