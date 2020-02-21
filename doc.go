// Copyright (C) 2020 Storj Labs, Inc.
// Copyright (C) 2014 Space Monkey, Inc.
// See LICENSE for copying information.

/*
Package zipkin provides a monkit plugin for sending traces to Zipkin.

Example usage

Your main method gets set up something like this:

  package main

  import (
	  "net/http"

	  "gopkg.in/spacemonkeygo/monkit-zipkin.v2"
	  "gopkg.in/spacemonkeygo/monkit.v2"
	  "gopkg.in/spacemonkeygo/monkit.v2/environment"
	  "gopkg.in/spacemonkeygo/monkit.v2/present"
  )

  func main() {
	  environment.Register(monkit.Default)
	  go http.ListenAndServe("localhost:9000", present.HTTP(monkit.Default))
	  collector, err := zipkin.NewScribeCollector("zipkin.whatever:9042")
	  if err != nil {
		  panic(err)
	  }
	  zipkin.RegisterZipkin(monkit.Default, collector, zipkin.Options{
		  Fraction: 1})

		...
  }

Once you've done that, you need to make sure your HTTP handlers pull Zipkin
Context info from the Request. That's easy to do with zipkin.ContextWrapper.

  func HandleRequest(ctx context.Context, w http.ResponseWriter,
    r *http.Request) {
    defer mon.Task()(&ctx)(nil)

    ... whatever
  }

  func DoTheThing(ctx context.Context) (err error) {
    defer mon.Task()(&ctx)(&err)
    return http.Serve(listener, zipkin.ContextWrapper(
      zipkin.TraceHandler(zipkin.ContextHTTPHandlerFunc(HandleRequest))))
  }

Last, your outbound HTTP requests need to pass through Context info:

  func MakeRequest(ctx context.Context) (err error) {
    defer mon.Task()(&ctx)(&err)
    req, err := http.NewRequest(...)
    if err != nil {
      return err
    }
    resp, err := zipkin.TraceRequest(ctx, http.DefaultClient, req)
    ...
  }
*/
package zipkin // import "gopkg.in/spacemonkeygo/monkit-zipkin.v2"
