 #!/bin/sh

 GOOS=linux go build -o gorep *.go
 GOOS=windows go build -o gorep.exe *.go
 