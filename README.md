# OPC DA in Go

[![License](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://github.com/konimarti/opc/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/konimarti/observer?status.svg)](https://godoc.org/github.com/konimarti/opc)
[![goreportcard](https://goreportcard.com/badge/github.com/konimarti/observer)](https://goreportcard.com/report/github.com/konimarti/opc)

Read and write process and automation data in Go from an OPC server for monitoring and data analysis purposes (OPC DA protocol).

```go get github.com/konimarti/opc```

## Usage
```go
client, _ := opc.NewConnection(
	"Graybox.Simulator", 		// ProgId
	[]string{"localhost"}, 		// Nodes
	[]string{"numeric.sin.float"}, 	// Tags
)
defer client.Close()
client.ReadItem("numeric.sin.float")
```

```go
browser, _ := opc.CreateBrowser(
	"Graybox.Simulator", 		// ProgId
	[]string{"localhost"}, 		// Nodes	
)
opc.PrettyPrint(browser)
```


## Installation

* ```go get github.com/konimarti/opc```

### Troubleshooting

* OPC DA Automation Wrapper 2.02 should be installed on your system (```OPCDAAuto.dll``` or ```gbda_aut.dll```); the automation wrapper is usually shipped as part of the OPC Core Components of your OPC Server.
* You can get the Graybox DA Automation Wrapper [here](http://gray-box.net/download_daawrapper.php?lang=en). Follow the [installation instruction](http://gray-box.net/daawrapper.php) for this wrapper. 
* Depending on whether your OPC server and automation wrapper are 32-bit or 64-bit, set the Go architecture correspondingly:
  - For 64-bit OPC servers and wrappers: DLL should be in ```C:\Windows\System32```, use ```$ENV:GOARCH="amd64"```
  - For 32-bit OPC servers and wrappers: DLL should be in ```C:\Windows\SysWOW64```, use ```$ENV:GOARCH="386"```
* Make sure to have correct DCOM settings on your local and remote computers: ```Dcomcnfg.exe```

### Reporting Issues

* If you find a bug in the code, please create an issue and suggest a solution how to fix it.
* Issues that are related to connection problems are mostly because of a faulty installation of your OPC automation wrapper or some peculiarties of your specific setup and OPC installation. Since we cannot debug your specific situation, these issues will be directly closed.

### Debugging

* Add ```opc.Debug()``` before the ```opc.NewConnection``` call to print more debug-related information.

### Testing

* Start Graybox Simulator v1.8. This is a free OPC simulation server and require for testing this package. It can be downloaded [here](http://www.gray-box.net/download_graysim.php).
* If you use the Graybox Simulator, set $GOARCH environment variable to "386", i.e. enter ```$ENV:GOARCH=386``` in Powershell.
* Test code with ```go test -v```

## Example 

```go
package main

import (
	"fmt"
	"github.com/konimarti/opc"
)

func main() {
	client, _ := opc.NewConnection(
		"Graybox.Simulator", // ProgId
		[]string{"localhost"}, //  OPC servers nodes
		[]string{"numeric.sin.int64", "numeric.saw.float"}, // slice of OPC tags
	)
	defer client.Close()

	// read single tag: value, quality, timestamp
	fmt.Println(client.ReadItem("numeric.sin.int64"))

	// read all added tags
	fmt.Println(client.Read())
}
``` 

with the following output:

```
{91 192 2019-06-21 15:23:08 +0000 UTC}
map[numeric.sin.int64:{91 192 2019-06-21 15:23:08 +0000 UTC} numeric.saw.float:{-36.42 192 2019-06-21 15:23:08 +0000 UTC
}]
```

## Credits

This software package has been developed for and is in production at [Kalkfabrik Netstal](http://www.kfn.ch/en).
