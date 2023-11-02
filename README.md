```shell
git clone teatweet
# your twitter credentials in JSON
# can be written to .env file instead
export TWITTER_CREDENTIALS='[{"username":"u","password":"p"}]'
go run cmd/teatweet/main.go serve --addr 127.0.0.1:8001
```

```shell
curl http://127.0.0.1:8001/following?id=1415522287126671363
```