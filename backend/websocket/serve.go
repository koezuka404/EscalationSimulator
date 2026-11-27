package websocket

import "context"

//画面への配信と、他のサーバーからの知らせを始める
func Serve(ctx context.Context, hub *Hub, remote *RemoteNotices) {
	go hub.Run()
	remote.Run(ctx)
}
