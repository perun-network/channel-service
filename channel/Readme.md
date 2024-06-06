### Perun Channel Demo
This example demostrates how a client can find its peer using a discovery service and how they can restore their channels from persistence

### Setup
Clone the entire repository Channel-Service (branch persistence)
```bash
    git clone https://github.com/perun-network/channel-service.git
    cd Channel-Service/channel
```
Running this test requires a running local ckb  blockchain.
To do this, navigate to the devnet subdirectory and execute `make dev`
```bash
cd /channel/devnet
make dev
```
This will open `tmux` session.

Now open a new terminal and navigate to channel directory again. Then compile the executable and run
```
cd ./channel
go build -o channel
./channel
```

