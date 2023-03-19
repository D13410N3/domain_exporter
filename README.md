# domain_exporter

## I've done this just for me

1. export `CF_TOKEN` (and `LISTEN_ADDR`, `LISTEN_PORT` if you need them)
2. run app.go

It will get list of domains from your CF account and parse their whois data via shell to find "Expiry date"
