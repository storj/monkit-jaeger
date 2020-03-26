// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2014 Space Monkey, Inc.
// See LICENSE for copying information.

/*
Package jaeger provides a monkit plugin for sending traces to Jaeger Agent.

Example usage

Your main method gets set up something like this:

  package main

  import (
	  "net/http"

	  jaeger "storj.io/monkit-jaeger"
	  "github.com/spacemonkeygo/monkit/v3"
	  "github.com/spacemonkeygo/monkit/v3/environment"
	  "github.com/spacemonkeygo/monkit/v3/present"
  )

  func main() {
	  environment.Register(monkit.Default)
	  collector, err := jaeger.NewUDPCollector("zipkin.whatever:9042", 200, "service name", []jaeger.Tag{
		  jaeger.Tag{
			  ....
		  }
	  })
	  if err != nil {
		  panic(err)
	  }
	  jaeger.RegisterJaeger(monkit.Default, collector, jaeger.Options{
		  Fraction: 1})

		...
  }
*/
package jaeger // import "storj.io/monkit-jaeger"
