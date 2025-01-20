# /!bin/bash

mkdir -p bin/v11-appliance
go build -ldflags "-X main.versionString=$(git describe --tags --dirty --always)"
cp scion-go-multiping bin/v11-appliance/

mkdir -p bin/v11
CGO_ENABLED=1 CC="zig cc -target native-native-musl" CXX="zig cc -target native-native-musl" go build -ldflags "-X main.versionString=$(git describe --tags --dirty --always)"
cp scion-go-multiping bin/v11/

mkdir -p bin/v12
git checkout scion-v12
CGO_ENABLED=1 CC="zig cc -target native-native-musl" CXX="zig cc -target native-native-musl" go build -ldflags "-X main.versionString=$(git describe --tags --dirty --always)"
cp scion-go-multiping bin/v12/

git checkout master