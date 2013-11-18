package main

import (
    "testing"
)

type A struct {
    n string
    datestr string
    slug string
    ext string
    err error
}

var nameMatcherCases = []A {
    {"18-11-2013-first-item.md", "18-11-2013", "first-item", "md", nil},
}

func TestNameMatcher(t *testing.T) {
    for _, c := range nameMatcherCases {
        datestr, slug, ext, err := matchName(c.n)
        if datestr != c.datestr {
            t.Error("Expected datestr", c.datestr, "but got", datestr)
        }
        if slug != c.slug {
            t.Error("Expected slug", c.slug, "but got", slug)
        }
        if ext != c.ext {
            t.Error("Expected ext", c.ext, "but got", ext)
        }
        if err != c.err {
            t.Error("Expected err", c.err, "but got", err)
        }
    }
}