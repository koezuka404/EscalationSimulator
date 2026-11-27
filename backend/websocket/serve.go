package websocket

import "context"

//画面への配信と、他のサーバーからの知らせを始め、停止で画面を切る
func Serve(ctx context.Context, hub *Hub, remote *RemoteNotices) {
	go hub.Run()
	if remote != nil {
		remote.Run(ctx)
	}
	<-ctx.Done()
	hub.Close()
}
