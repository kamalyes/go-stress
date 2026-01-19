# Proto 文件编译脚本

# 编译 proto 文件
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    stress.proto

echo "Proto files compiled successfully"
