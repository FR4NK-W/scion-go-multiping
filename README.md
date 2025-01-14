# SCION Go Multiping

## Static linking with native SQLite driver
At the moment we use the native SQLite driver in CGO for writing to the database, which is considered much more stable and reliable than the pure go-based one, which might be very helpful in our case of writing a large amount of data to the database. To compile and link it statically, the following command (based on zig) can be used. 

```
CGO_ENABLED=1 CC="zig cc -target native-native-musl" CXX="zig cc -target native-native-musl" go build
```

To compile it with the pure go-based driver, comment out `gorm.io/driver/sqlite` in `exporter_sqlite.go` and use `github.com/glebarez/sqlite` instead 