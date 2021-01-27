# Barcode Buddy Federation Server

### Available for:

- Bare Metal

## About
This is the repository of the official Barcode Buddy Federation server. It is built to be very efficient and can run on very low-end nodes. It was tested on a VM with 256MB Ram and a single core and was able to serve more than 10,000 requests per second. Serving 1,000,000 barcodes requires about 300MB RAM.

## Prerequisites

A redis instance and ideally a reverse proxy for SSL.

## Installing

Download the appropiate release binary to start the server. Alternatively you can build it from source by cloning this repository and running `go build`.

## Usage

During the first start, the file `config/config.json` will be created. Stop the server with CTRL+C and and adjust the file to your needs. Then restart the server to load the new configuration.

To use the server in Barcode Buddy, adjust the constant `HOST` in `incl/modules/barcodeFederation.php` of your Barcode Buddy instance.

An admin overview is available at `localhost:18900/admin`.

## License

This project is licensed under the GNU GPL3 - see the [LICENSE.md](LICENSE.md) file for details
