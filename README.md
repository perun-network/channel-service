<h1 align="center"><br>
    <a href="https://perun.network/"><img src=".assets/go-perun.png" alt="Perun" width="196"></a>
<br></h1>

<h2 align="center">Perun-Channel-Service</h2>

<p align="center">
  <a href="https://www.apache.org/licenses/LICENSE-2.0.txt"><img src="https://img.shields.io/badge/license-Apache%202-blue" alt="License: Apache 2.0"></a>
</p>

This repository contains contains an implementation of the Channel Service, which is standardized to allow implementing Perun channels into any CKB wallet ready to conform with the standard.

The specification can be found in the repository [perun-wallet-spec](https://github.com/perun-network/perun-wallet-spec). 


## Table of Contents

- [Table of Contents](#table-of-contents)
- [Introduction](#introduction)
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [Demo](#demo)
- [License](#license)

## Introduction

The Perun-Channel-Service is a core component designed to facilitate the integration of Perun payment channels into CKB wallets. By adhering to the standardized implementation, developers can seamlessly integrate this service into their wallet applications, ensuring compatibility and enhanced functionality.

## Features

- Perun Payment Channels interactions (Open, Update, Close, Restore)
- Built-in persistence
- Compatible with any CKB wallet
- Follows the [perun-wallet-spec](https://github.com/perun-network/perun-wallet-spec)
- Open-source under the Apache 2.0 license

## Installation

Import the Perun-Channel-Service module:

```
go get github.com/perun-network/channel-service
```

You might need to replace the github path with the actual module name in your `go.mod`:

```
require (
    github.com/perun-network/perun-channel-service v0.1.0
)

replace perun.network/channel-service v0.0.0 => github.com/perun-network/channel-service v0.0.0-20240621104817-22b8b42f7eb8
```
## Usage

Start a Channel-Service instance in a GRPC Server:

```
```

## Demo

The [Perun-Nervos-Demo](https://github.com/perun-network/perun-nervos-demo) demonstrates the use of how a Wallet User can interact with the Channel-Service through RPC Messages and manage Perun Payment Channels on Nervos Network.


## License
This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

