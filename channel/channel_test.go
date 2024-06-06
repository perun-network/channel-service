package main

import (
	"fmt"
	"testing"

	"github.com/perun-network/channel-service/channel/test"
	"perun.network/go-perun/channel"
)

func TestRestoreChannels(t *testing.T) {
	t.Run("payment-channels on ckb example", func(t *testing.T) {
		runRestoreChannels(t)
	})
}

func runRestoreChannels(t *testing.T) {
	setup := test.NewTestSetup(t)
	alice := setup.PaymenClients[0]
	bob := setup.PaymenClients[1]
	ckbAsset := setup.Asset
	str := "'s account balance"

	fmt.Println("Opening channel and depositing funds")
	chAlice := alice.OpenChannel(bob.WireAddress(), bob.PeerID(), map[channel.Asset]float64{
		&ckbAsset: 100.0,
	})
	fmt.Println("Alice sent proposal")
	chBob := bob.AcceptedChannel()
	fmt.Println("Bob accepted proposal")
	fmt.Println("Sending payments....")

	chAlice.SendPayment(map[channel.Asset]float64{
		&ckbAsset: 10.0,
	})
	fmt.Println("Alice sent Bob a payment")

	chBob.SendPayment(map[channel.Asset]float64{
		&ckbAsset: 10.0,
	})
	fmt.Println("Bob sent Alice a payment")

	chAlice.SendPayment(map[channel.Asset]float64{
		&ckbAsset: 10.0,
	})
	fmt.Println("Alice sent Bob a payment")

	fmt.Println("Payments completed")

	fmt.Println("Skip Settling Channel and force client shutdown")
	//chAlice.Settle()

	fmt.Println(alice.Name, str, alice.GetBalances())
	fmt.Println(bob.Name, str, bob.GetBalances())

	//cleanup
	alice.Shutdown()
	bob.Shutdown()
	fmt.Println("Clients shutdown, exiting method")

	setup.NewPaymentClients(t)
	alice2 := setup.PaymenClients[0]
	bob2 := setup.PaymenClients[1]
	fmt.Println("Starting restoring channels")
	chansAlice := alice2.Restore(bob2.WireAddress(), bob2.PeerID())
	fmt.Println("Alice's channel restored")
	chansBob := bob2.Restore(alice2.WireAddress(), alice2.PeerID())
	fmt.Println("Alice and Bob's channels successfully restored")

	// Print balances after transactions.
	fmt.Println(alice.Name, str, alice.GetBalances())
	fmt.Println(bob.Name, str, bob.GetBalances())

	fmt.Println("Alice sending payment to Bob")
	chansAlice[0].SendPayment(map[channel.Asset]float64{
		&ckbAsset: 10.0,
	})
	fmt.Println("Bob sending payment to Alice")
	chansBob[0].SendPayment(map[channel.Asset]float64{
		&ckbAsset: 10.0,
	})

	chansAlice[0].Settle()
	fmt.Println("Balances after settling channel")
	fmt.Println(alice.Name, str, alice.GetBalances())
	fmt.Println(bob.Name, str, bob.GetBalances())

}
