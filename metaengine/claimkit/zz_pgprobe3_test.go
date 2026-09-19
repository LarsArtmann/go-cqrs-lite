package claimkit_test

import "time"

func timeNowNano() int64          { return time.Now().UnixNano() }
func timeNowPlus(ns int64) string { return time.Now().Add(time.Duration(ns)).Format(time.RFC3339Nano) }
