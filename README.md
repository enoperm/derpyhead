Quick and dirty `tailscaled` substitute for `derper`,
which should enable running non-public derper services on a host without joining it to a tailnet,
as well as serving multiple tailnets with a single derper instance.

It functions by asking a command (provided by you) about the node keys allowed to use the derper instance.
Said command *MUST* return zero or more hex encoded node keys without any pre- or suffixes, and return zero on success, or non-zero on failure, in which case this service will retain the previously cached keys until the key-retrieval command succeeds again.

This designs allows for using *arbitrary* sources for the keys.

Some examples:

* Hardcoded
```sh
#!/usr/bin/env sh
echo key1
echo key2
```

* Fetching from a remote service over HTTP:
```sh
#!usr/bin/env sh
exec curl https://trusted-server.example.com/headscale-client-keys.txt
```

* Reading from a local sqlite3 DB, as used by [Headscale](https://github.com/juanfont/headscale)
```sh
#!usr/bin/env sh
exec sqlite3 "${headscale_db_path}" 'select node_key from machines;' 
```
