server: keysrv.go
	CGO_LDFLAGS="-Xlinker -static" go build -o keysrv keysrv.go
	ssh -i ~/.ssh/keys/wpi -p 8236 student@secnet-gateway.cs.wpi.edu sudo systemctl stop keysrv
	scp -i ~/.ssh/keys/wpi -P 8236 keysrv student@secnet-gateway.cs.wpi.edu:~/
	ssh -i ~/.ssh/keys/wpi -p 8236 student@secnet-gateway.cs.wpi.edu sudo systemctl start keysrv

client: client.go
	CGO_LDFLAGS="-Xlinker -static" go build -o client client.go
	scp -i ~/.ssh/keys/wpi -P 8234 client student@secnet-gateway.cs.wpi.edu:~/
