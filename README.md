# Spitcurl 

A reverse proxy and mock server that logs request in cURL format

usage:

```sh
Usage of spitcurl:
  -address string
    	Bind address (default "127.0.0.1")
  -mode string
    	server type to use. options: http, tls. (default "http")
  -port uint
    	listen port (default 8080)
  -tlsCert string
    	tls certificate to use. will use self-signed if empty
  -tlsKey string
    	tls certificate key to use. will use self-signed if empty
  -upstream string
    	upstream URL. Empty will return an empty 200 for all requests, Example: https://www.youtube.com
```