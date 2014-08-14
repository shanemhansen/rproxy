rproxy
======

A Reverse proxy to multiple backends with shared secret auth

# why

rproxy is used to have a single authenticating server to proxy requests
to multiple backend servers. It supports ssl and basic shared secret auth.

# configuring

rproxy is configured with a toml file. see the example. The Host can
be specified multiple times. ssl is optional.
