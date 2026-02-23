//go:build windows

package main

// maybeDaemonize is a no-op on Windows (no fork/setsid). Use a service manager or
// start the process in the background instead.
func maybeDaemonize(parsed parsedArgs) {}
