Leerlauf
========

Leerlauf is a rate limiter for Google Cloud AppEngine. It uses the
AppEngine Memcache backend as persistence.

Usage
-----

Note: The maximum keysize for Memcache is 250 bytes. Make sure the
description plus the id (plus one) does not exceed this limit.

1) Create a rate limiter with `NewLimit`. Give a description and max
hits per minute.
2) On each request, call `l.Limited` including an Id. For AppEngine, it
might be handy to use `r.RemoteAddr` as Id, but any other string would
do.

```
if l := limit.Limited(ctx, r.RemoteAddr); l != nil {
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte("Access Denied"))
	return
}
```

Technical Details
-----------------

The mitigation strategy is basically counting hits over the period of
the past 60 seconds, obviously. The technique behind is described
in a pretty awesome as usual blog post by the friends at
[cloudflare](https://blog.cloudflare.com/counting-things-a-lot-of-different-things/)