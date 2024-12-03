# GameLift Server Example

このパッケージは、Amazon GameLift サーバーの設定と HTTP サーバーの起動を行う Go 言語のサンプルです。

## 概要

このサンプルは、GameLift サーバーの設定を行い、HTTP サーバーを起動します。

- GameLift サーバーの設定
- HTTP サーバーの起動
- ログの設定
- シャットダウン処理


## インストール
```sh
git clone https://github.com/mikanbox/gamelift_server_example_go.git
cd gamelift_server_example_go
go mod tidy
```

### 引数
MANAGED Fleet の場合は portArg のみ必要です。


- webSocketURLArg: WebSocket URL
- hostIDArg: ホスト ID
- fleetIDArg: フリート ID
- authTokenArg: 認証トークン
- portArg: ポート番号
- fleetTypeArg: フリートタイプ ("MANAGED" または "ANYWHERE")


### ログの設定

`setupLogging` 関数は、ログの設定を行います。フリートタイプが "MANAGED" の場合、特定のパスにログを出力します。

