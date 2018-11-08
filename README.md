# damogo

## feature

- 采用grpc

- 采用etcd 做group 分片路由




## docker

```
docker build -t damogo:v1 .
docker run -id -p 8009:8009 -v /home/wida/dockerdata/damogo:/data/app/log:rw  --name damogo damogo:v1
```