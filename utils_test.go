package main

import (
    "github.com/memcachier/mc"
    "testing"
)

func TestFetch(t *testing.T) {

    url := "http://example.com"
    cache := mc.Client {}
    result := fetchUrl(url, &cache)

    if result == nil {
        t.Fatalf("Failure")
    }
}