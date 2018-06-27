# graceful-restart-go

# 使い方
`Server`側と`Client`側の二つのターミナルを使う。

#### Server側
ビルドしてHTTPサーバを起動する
```
$ go build
$ ./graceful-restart-go
2018/06/27 22:43:48 master pid: 11109
2018/06/27 22:43:48 worker pid: 11110
```

#### Client側
curlを叩いてみると、`Hello, World!`が返ってくる
```
$ curl localhost:8888
Hello, World!
```
ソースコードの書き換え。
```
vim main.go

## indexHandlerを書き換えた
func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Graceful Restart!")
}
```
ビルドしてmasterプロセスにHUPを送信
```
$ go build
$ kill -HUP 11109
```
もう一度curlを叩いてみる
```
$ curl localhost:8888
Graceful Restart!
```
変更が反映されていた

# 参考
- [miyabi](https://github.com/naoina/miyabi)
- Goならわかるシステムプログラミング | 渋川よしき
- [Server::Starterから学ぶhot deployの仕組み](https://blog.shibayu36.org/entry/2012/05/07/201556)