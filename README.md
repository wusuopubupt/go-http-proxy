# A simple http proxy written by Golang

## Usage:


### install
``` shell
go get github.com/wusuopubupt/go-http-proxy
```

### run proxy server on you vps which can reach google.com
```
cd ${GOPATH}/src/github.com/wusuopubupt/go-http-proxy && go run main.go -addr localhost:6666
```

### test
``` shell
curl -x localhost:6666 www.google.com
```

