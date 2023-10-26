module perun.network/channel-service

go 1.19

require (
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0
	github.com/nervosnetwork/ckb-sdk-go/v2 v2.2.0
	github.com/stretchr/testify v1.8.4
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
	perun.network/go-perun v0.10.7-0.20230808153546-74844191e56e
	perun.network/perun-ckb-backend v0.0.0-20231026110519-7a17b08740d8
)

require (
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/Pilatuz/bigz v1.2.1 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/deckarep/golang-set/v2 v2.3.1 // indirect
	github.com/ethereum/go-ethereum v1.13.4 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/holiman/uint256 v1.2.3 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/stretchr/objx v0.5.1 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.3 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/mod v0.13.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sync v0.4.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/tools v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231016165738-49dd2c1f3d0b // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	polycry.pt/poly-go v0.0.0-20220301085937-fb9d71b45a37 // indirect
)

replace github.com/nervosnetwork/ckb-sdk-go/v2 v2.2.0 => github.com/perun-network/ckb-sdk-go/v2 v2.2.1-0.20230601140721-2bf596fddd80
