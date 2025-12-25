// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

// Package main provides the entry point for the Azure Functions Go Worker.
//
// This is the worker executable that the Azure Functions Host launches.
// User functions are compiled into this binary by importing the user's
// function package which registers handlers via azfunc.RegisterHttpFunction().
//
// Usage (by Azure Functions Host):
//
//	./worker --host 127.0.0.1 --port 12345 --workerId abc123 --requestId def456
package main

import (
	"log"
	"os"

	"github.com/Azure/azure-functions-go-worker/pkg/azfunc"
)

func main() {
	// Configure logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.SetOutput(os.Stderr)

	log.Println("Azure Functions Go Worker starting...")

	// Note: In a real deployment, user functions would be registered here
	// by importing a package that calls azfunc.RegisterHttpFunction() in its init()
	//
	// For example:
	//   import _ "myapp/functions"
	//
	// The functions package would have:
	//   func init() {
	//       azfunc.RegisterHttpFunction("HttpTrigger", handleHttp)
	//   }

	// Start the worker
	if err := azfunc.Start(); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}
