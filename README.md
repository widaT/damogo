# damogo



## docker

```
docker build -t damogo:v1 .
docker run -id -p 8009:8009 -v /home/wida/dockerdata/damogo:/data/app/log:rw  --name damogo damogo:v1
```