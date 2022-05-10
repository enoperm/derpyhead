Quick and dirty `tailscaled` API mock for `derper`,
allows you to run derper on a host without joining it to a tailnet,
as well as to serve multiple tailnets with a single derper instance,
so long as your control server is a [headscale](https://github.com/juanfont/headscale) instance backed by sqlite.

You *probably do not want* to use this without modifications, much less so in production.
