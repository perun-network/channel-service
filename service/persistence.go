package service

import (
	"context"
	"errors"
	"log"

	"perun.network/go-perun/client"
)

func (u *User) restoreChannels(ctx context.Context) {
	var err error

	u.PerunClient.OnNewChannel(func(ch *client.Channel) {
		id := ch.ID()
		if _, ok := u.Channels[id]; ok {
			// If the channel already exists, assign an error to err
			err = errors.New("channel already exists")
		} else {
			// If the channel does not exist, do something
			// bFor example, add the new channel to the map:
			u.Channels[id] = ch
		}
	})
	// Check if an error occurred in the OnNewChannel call
	if err != nil {
		// Handle the error, for example by logging it and returning
		log.Println(err)
		return
	}
	// call client's restore method
	err = u.PerunClient.Restore(ctx)
	// Check if an error occurred in the Restore call
	if err != nil {
		// Handle the error, for example by logging it and returning
		log.Println(err)
		return
	}
	// set OnNewChannel callback to zero value
	u.PerunClient.OnNewChannel(nil)
}
