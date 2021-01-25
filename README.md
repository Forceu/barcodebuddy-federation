# Barcode Buddy Federation Server

Server for running a Barcode Buddy Federation instance. Documentation WIP.

Redis needs to be installed.

To build: Download the latest release (alternatively clone this repository and run `go build`). Start the executable and then quit the application. Adjust the newly created file `config/config.json` before restarting the server.

To use the server in Barcode Buddy, adjust the variable `HOST` in `incl/modules/barcodeFederation.php` of your Barcode Buddy instance. The admin menu can be accessed at `localhost:18900/admin`

