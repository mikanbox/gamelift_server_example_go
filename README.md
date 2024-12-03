# GameLift Server Example

このパッケージは、Amazon GameLift サーバーの設定と HTTP サーバーの起動を行う Go 言語のサンプルです。

## 概要

このパッケージは、GameLift サーバーの設定を行い、HTTP サーバーを起動するための関数を提供します。主な機能は以下の通りです。

- GameLift サーバーの設定
- HTTP サーバーの起動
- ログの設定
- シャットダウン処理

## 使い方

### インストール

このパッケージを使用するには、まず Go 言語の環境をセットアップしてください。その後、以下のコマンドでパッケージをインストールします。

```sh
go get github.com/mikanbox/gamelift_server_example_go
```

### コード例

以下に、このパッケージを使用して HTTP サーバーを起動するコード例を示します。
AWS にサーバーを上げる場合は、port だけ値をセットすれば問題ないです。
ANYWHERE フリートを使いローカルで起動する場合は各種引数をセットする必要があります。

```go
package main

import (
    "github.com/mikanbox/gamelift-server-example-go"
)


func main() {
	webSocketURLArg := flag.String("webSocketURL", "wss://ap-northeast-1.api.amazongamelift.com", "WebSocket URL for sync gamelift status")
	hostIDArg := flag.String("hostID", "", "Compute name with RegisterCompute API")
	fleetIDArg := flag.String("fleetID", "", "Fleet ID")
	authTokenArg := flag.String("authToken", "", "Auth Token")
	portArg := flag.String("port", "8080", "Port")
	fleetTypeArg := flag.String("fleetType", "MANAGED", "Fleet type")

	flag.Parse()

	gamelift_server_example.AddExampleHTTPServer(
		*webSocketURLArg,
		*hostIDArg,
		*fleetIDArg,
		*authTokenArg,
		*portArg,
		*fleetTypeArg,
	)
}
```

### 関数の説明
#### AddExampleHTTPServer

```go
func AddExampleHTTPServer(webSocketURLArg, hostIDArg, fleetIDArg, authTokenArg, portArg, fleetTypeArg string)
```

この関数は、指定された引数を使用して HTTP サーバーを起動します。

- `webSocketURLArg`: WebSocket URL
- `hostIDArg`: ホスト ID
- `fleetIDArg`: フリート ID
- `authTokenArg`: 認証トークン
- `portArg`: ポート番号
- `fleetTypeArg`: フリートタイプ ("MANAGED" または "ANYWHERE")

### ログの設定

`setupLogging` 関数は、ログの設定を行います。フリートタイプが "MANAGED" の場合、特定のパスにログを出力します。

