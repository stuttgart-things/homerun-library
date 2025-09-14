# TESTS

## START REDIS

```bash
export REDIS_PASSWORD=<PASSWORD>

docker run -d \
--name redis-stack-server \
-p 6379:6379 \
-e REDIS_ARGS="--requirepass ${REDIS_PASSWORD}" \
redis/redis-stack-server:7.2.0-v18
```

## RUN TESTS

```bash

export REDIS_PASSWORD=<PASSWORD>
go run tests/pitcher/pitch_message.go
```