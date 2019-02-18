package main

import (
	"fmt"
)

type testModule string

func (tm testModule) Name() string {
	return "TestModule"
}

func (tm testModule) EnvVars() []string {
	return []string{"One", "Two"}
}

func (tm testModule) Run(in interface{}, ev map[string]string, interactions map[string]string) error {
	fmt.Println("******* MODULE RUNNING!")
	fmt.Printf("%v\n", in)
	fmt.Printf("%v\n", ev)
	fmt.Printf("Type of in is %T\n", in)

	fmt.Println("Iterating EnvVars:")
	for k, v := range ev {
		fmt.Printf("'%s': '%s'\n", k, v)
	}

	switch i := in.(type) {
	case map[string]string:
		fmt.Println("Iterating over response:")
		for k, v := range i {
			fmt.Printf("'%s': '%s'\n", k, v)
		}
	}

	return nil
}

// Module is exported to be picked up by the plugin system
var Module testModule
