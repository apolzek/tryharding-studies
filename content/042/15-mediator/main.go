package main

import (
	"context"
	"fmt"
)

func main() {
	ctx := context.Background()
	room := NewAuctionRoom(10)

	alice := NewBidder("alice")
	bob := NewBidder("bob")
	carol := NewBidder("carol")

	_ = room.Join(alice)
	_ = room.Join(bob)
	_ = room.Join(carol)

	_ = room.Broadcast(ctx, "alice", "bom dia!")
	_ = room.Bid(ctx, "alice", 100)
	_ = room.Bid(ctx, "bob", 120)
	if err := room.Bid(ctx, "carol", 125); err != nil {
		fmt.Println("rejeitado:", err)
	}
	_ = room.Bid(ctx, "carol", 150)

	room.Close(ctx)

	winner, bid := room.HighestBid()
	fmt.Printf("vencedor=%s valor=%d\n", winner, bid)
	fmt.Printf("alice recebeu %d msgs\n", len(alice.Inbox()))
	fmt.Printf("bob recebeu %d msgs\n", len(bob.Inbox()))
	fmt.Printf("carol recebeu %d msgs\n", len(carol.Inbox()))
}
