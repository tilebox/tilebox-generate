<h1 align="center">
  <img src="https://storage.googleapis.com/tbx-web-assets-2bad228/banners/tilebox-banner.svg" alt="Tilebox Logo">
  <br>
</h1>

<p align="center">
  <a href="https://docs.tilebox.com/introduction"><b>Documentation</b></a>
  |
  <a href="https://console.tilebox.com/"><b>Console</b></a>
  |
  <a href="https://tilebox.com/discord"><b>Discord</b></a>
</p>

# Tilebox Generate

CLI tool to generate Tilebox datasets types for Go. It is intended be used alongside [tilebox-go](https://github.com/tilebox/tilebox-go).

## Installation

```bash
go install github.com/tilebox/tilebox-generate@latest
```

## Examples

Generate a dataset type for [Sentinel-1 SAR](https://docs.tilebox.com/datasets/open-data) dataset to `./protogen` using $TILEBOX_API_KEY api key:

```bash
tilebox-generate --dataset open_data.copernicus.sentinel1_sar
```

Same as above, but override the output directory, package name and message name:

```bash
tilebox-generate --dataset open_data.copernicus.sentinel1_sar --out ./protogen --package tilebox.v1 --name MyDataset
```

## Usage

```
NAME:
   tilebox-generate - Generate Tilebox datasets types for Go

USAGE:
   tilebox-generate [global options]

GLOBAL OPTIONS:
   --dataset value          A valid dataset slug e.g. 'open_data.copernicus.sentinel1_sar'
   --out value              The directory to generate output files in (default: protogen)
   --package value          Package name (default: tilebox.v1)
   --name value             Protobuf message name for the dataset (optional)
   --tilebox-api-key value  A Tilebox API key [$TILEBOX_API_KEY]
   --tilebox-api-url value  The Tilebox API URL (default: https://api.tilebox.com) [$TILEBOX_API_URL]
   --help, -h               show help
```

## Usage with tilebox-go

Usage example to [load typed data](https://github.com/tilebox/tilebox-go/blob/main/examples/load/main.go) from Tilebox.

To have typed custom datasets in Go you need to replace `tileboxdatasets.CollectAs` type with the generated one.

```go
package main

import (
	tileboxdatasets "github.com/tilebox/tilebox-go/datasets/v1"
	"log"
	// TODO: replace with your own path to the generated package
	tileboxv1 "path/to/protogen/tilebox/v1"
)

func main() {
	// ...

	// Load data of my custom datasets
	// TODO: replace tileboxv1.Sentinel1Sar with your own dataset type
	datapoints, err := tileboxdatasets.CollectAs[*tileboxv1.Sentinel1Sar](collection.Load(ctx, loadInterval))
	if err != nil {
		log.Fatalf("Failed to load and collect datapoints: %v", err)
	}
	_ = datapoints // now datapoints are typed using tileboxv1.Sentinel1Sar
	
	// ...
}
```
